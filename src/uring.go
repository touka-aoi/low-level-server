package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log"
	"log/slog"
	"runtime"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"
)

const (
	EVENT_TYPE_ACCEPT = iota
	EVENT_TYPE_READ
	EVENT_TYPE_WRITE
)

const (
	IORING_OFF_SQ_RING int64 = 0
	ORING_OFF_CQ_RING  int64 = 0x8000000
	IORING_OFF_SQES    int64 = 0x10000000
)

// IO_URING_ENTER flags
const (
	IORING_ENTER_GETEVENTS = 1 << iota
	IORING_ENTER_SQ_WAKEUP
	IORING_ENTER_SQ_WAIT
	IORING_ENTER_EXT_ARG
	IORING_ENTER_REGISTERED_RING
	IORING_ENTER_ABS_TIMER
	IORING_ENTER_EXT_ARG_REG
)

const (
	IORING_OP_NOP = iota
	IORING_OP_READV
	IORING_OP_WRITEV
	IORING_OP_FSYNC
	IORING_OP_READ_FIXED
	IORING_OP_WRITE_FIXED
	IORING_OP_POLL_ADD
	IORING_OP_POLL_REMOVE
	IORING_OP_SYNC_FILE_RANGE
	IORING_OP_SENDMSG
	IORING_OP_RECVMSG
	IORING_OP_TIMEOUT
	IORING_OP_TIMEOUT_REMOVE
	IORING_OP_ACCEPT
	IORING_OP_ASYNC_CANCEL
	IORING_OP_LINK_TIMEOUT
	IORING_OP_CONNECT
	IORING_OP_FALLOCATE
	IORING_OP_OPENAT
	IORING_OP_CLOSE
	IORING_OP_FILES_UPDATE
	IORING_OP_STATX
	IORING_OP_READ
	IORING_OP_WRITE
	IORING_OP_FADVISE
	IORING_OP_MADVISE
	IORING_OP_SEND
	IORING_OP_RECV
	IORING_OP_OPENAT2
	IORING_OP_EPOLL_CTL
	IORING_OP_SPLICE
	IORING_OP_PROVIDE_BUFFERS
	IORING_OP_REMOVE_BUFFERS
	IORING_OP_TEE
	IORING_OP_SHUTDOWN
	IORING_OP_RENAMEAT
	IORING_OP_UNLINKAT
	IORING_OP_MKDIRAT
	IORING_OP_SYMLINKAT
	IORING_OP_LINKAT
	IORING_OP_MSG_RING
	IORING_OP_FSETXATTR
	IORING_OP_SETXATTR
	IORING_OP_FGETXATTR
	IORING_OP_GETXATTR
	IORING_OP_SOCKET
	IORING_OP_URING_CMD
	IORING_OP_SEND_ZC
	IORING_OP_SENDMSG_ZC
	IORING_OP_READ_MULTISHOT
	IORING_OP_WAITID
	IORING_OP_FUTEX_WAIT
	IORING_OP_FUTEX_WAKE
	IORING_OP_FUTEX_WAITV
	IORING_OP_FIXED_FD_INSTALL
	IORING_OP_FTRUNCATE
	IORING_OP_BIND
	IORING_OP_LISTEN

	IORING_OP_LAST
)

// accept flags stored in sqe->ioprio
// https://lore.kernel.org/lkml/a41a1f47-ad05-3245-8ac8-7d8e95ebde44@kernel.dk/t/
const (
	IORING_ACCEPT_MULTISHOT = 1 << iota
	IORING_ACCEPT_DONTWAIT
	IORING_ACCEPT_POLL_FIRST
)

const (
	IORING_FEAT_SINGLE_MMAP = 1 << 0
)

type UringSQE struct {
	Opcode      uint8
	Flags       uint8
	Ioprio      uint16
	Fd          int32
	Offset      uint64 // addr2
	Address     uint64 // addr1
	Len         uint32
	UserFlags   uint32 // union
	UserData    uint64
	BufIndex    uint16
	Personality uint16
	SpliceFdIn  int32
	Pad2        [2]uint64 // addr3
}

type UringCQE struct {
	UserData uint64
	Res      int32
	Flags    uint32
}

type Uring struct {
	Fd         int32
	SQ         SQ
	CQ         CQ
	sockAddr   sockAddr
	sockLen    uint32
	AccpetChan chan Peer
	Buffer     []byte
}

type SQ struct {
	SQPtr    uintptr
	Head     *uint32
	Tail     *uint32
	Mask     *uint32
	Entries  *uint32
	ArrayPtr uintptr
	SQEPtr   uintptr
}

type CQ struct {
	CQPtr   uintptr
	Head    *uint32
	Tail    *uint32
	Mask    *uint32
	Entries *uint32
	CQEs    *uint32
}

func CreateUring(entries uint32) *Uring {
	params := uringParams{}
	fd, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_SETUP,
		uintptr(entries),
		uintptr(unsafe.Pointer(&params)),
		0,
		0,
		0,
		0)

	if errno != 0 {
		log.Printf("IO_URING_SETUP failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}

	SQData, err := unix.Mmap(
		int(fd),
		IORING_OFF_SQ_RING,
		int(params.SQOffsets.Array+params.SqEntry*uint32(unsafe.Sizeof(uint32(0)))),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED|unix.MAP_POPULATE,
	)

	if err != nil {
		log.Printf("Mmap failed with errno: %d (%s)", err, err.Error())
		panic(err)
	}

	SQPtr := uintptr(unsafe.Pointer(unsafe.SliceData(SQData)))

	SQEData, err := unix.Mmap(
		int(fd),
		IORING_OFF_SQES,
		int(params.SqEntry)*int(unsafe.Sizeof(UringSQE{})),
		unix.PROT_READ|unix.PROT_WRITE,
		unix.MAP_SHARED|unix.MAP_POPULATE,
	)

	SQEPtr := uintptr(unsafe.Pointer(unsafe.SliceData(SQEData)))

	if err != nil {
		log.Printf("Mmap failed with errno: %d (%s)", err, err.Error())
		panic(err)
	}

	var CQPtr uintptr
	if params.Features&IORING_FEAT_SINGLE_MMAP == IORING_FEAT_SINGLE_MMAP {
		CQPtr = SQPtr
	} else {
		//TODO: kernel 5.4以前の対応
	}

	uring := &Uring{
		Fd: int32(fd),
		SQ: SQ{
			SQPtr:    SQPtr,
			Head:     (*uint32)(unsafe.Pointer(SQPtr + uintptr(params.SQOffsets.Head))),
			Tail:     (*uint32)(unsafe.Pointer(SQPtr + uintptr(params.SQOffsets.Tail))),
			Entries:  (*uint32)(unsafe.Pointer(SQPtr + uintptr(params.SQOffsets.RingEntries))),
			Mask:     (*uint32)(unsafe.Pointer(SQPtr + uintptr(params.SQOffsets.RingMask))),
			ArrayPtr: uintptr(unsafe.Pointer(SQPtr + uintptr(params.SQOffsets.Array))),
			SQEPtr:   SQEPtr,
		},
		CQ: CQ{
			CQPtr:   CQPtr,
			Head:    (*uint32)(unsafe.Pointer(CQPtr + uintptr(params.CQOffsets.Head))),
			Tail:    (*uint32)(unsafe.Pointer(CQPtr + uintptr(params.CQOffsets.Tail))),
			Entries: (*uint32)(unsafe.Pointer(CQPtr + uintptr(params.CQOffsets.RingEntries))),
			Mask:    (*uint32)(unsafe.Pointer(CQPtr + uintptr(params.CQOffsets.RingMask))),
			CQEs:    (*uint32)(unsafe.Pointer(CQPtr + uintptr(params.CQOffsets.CQEs))),
		},
		AccpetChan: make(chan Peer, 1024),
	}

	return uring

}

func (u *Uring) Accpet(socket *Socket, sockAddr *sockAddr, sockLen *uint32) {
	op := UringSQE{
		Opcode:   IORING_OP_ACCEPT,
		Ioprio:   IORING_ACCEPT_MULTISHOT, // https://lore.kernel.org/lkml/a41a1f47-ad05-3245-8ac8-7d8e95ebde44@kernel.dk/t/
		Fd:       socket.Fd,
		Offset:   uint64(uintptr(unsafe.Pointer(sockLen))),
		Address:  uint64(uintptr(unsafe.Pointer(sockAddr))),
		UserData: EVENT_TYPE_ACCEPT,
	}

	for {
		tail := atomic.LoadUint32(u.SQ.Tail)

		//TODO: 満杯になるケースを考慮する

		if atomic.CompareAndSwapUint32(u.SQ.Tail, tail, tail+1) {
			sqe := unsafe.Slice((*UringSQE)(unsafe.Pointer(u.SQ.SQEPtr)), *u.SQ.Entries)
			sqe[tail&*u.SQ.Mask] = op

			//TDOO: NO_ARRAYも試してみる
			array := unsafe.Slice((*uint32)(unsafe.Pointer(u.SQ.ArrayPtr)), *u.SQ.Entries)
			array[tail&*u.SQ.Mask] = tail

			break
		}
		runtime.Gosched()
	}

	u.sendSQE()

}

func (u *Uring) sendSQE() {
	res, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_ENTER,
		uintptr(u.Fd),
		1,
		0,
		0,
		0,
		0)

	if res < 0 || errno != 0 {
		//TODO エラーとして返す
		log.Printf("Uring failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

func (u *Uring) WatchRead(peer *Peer) error {
	u.Buffer = make([]byte, 1024)
	op := UringSQE{
		Opcode:   IORING_OP_READ,
		Fd:       peer.Fd,
		Address:  uint64(uintptr(unsafe.Pointer(unsafe.SliceData(u.Buffer)))),
		Len:      uint32(len(u.Buffer)),
		UserData: EVENT_TYPE_READ,
	}

	//TODO ここのforループを関数化する
	for {
		tail := atomic.LoadUint32(u.SQ.Tail)

		if atomic.CompareAndSwapUint32(u.SQ.Tail, tail, tail+1) {
			sqe := unsafe.Slice((*UringSQE)(unsafe.Pointer(u.SQ.SQEPtr)), *u.SQ.Entries)
			sqe[tail&*u.SQ.Mask] = op

			array := unsafe.Slice((*uint32)(unsafe.Pointer(u.SQ.ArrayPtr)), *u.SQ.Entries)
			array[tail&*u.SQ.Mask] = tail

			break
		}
		runtime.Gosched()
	}

	u.sendSQE()

	return nil
}

func (u *Uring) Wait() *UringCQE {
	res, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_ENTER,
		uintptr(u.Fd),
		0,
		1,
		IORING_ENTER_GETEVENTS,
		0,
		0)

	if res < 0 || errno != 0 {
		//TODO エラーとして返す
		log.Printf("Uring failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}

	cqe := u.getCQE()

	return cqe
}

func (u *Uring) Read(cqe *UringCQE) *bufio.Reader {
	//TODO fixed bufferを使うように変更
	copyBuffer := make([]byte, cqe.Res)
	copy(copyBuffer, u.Buffer[:cqe.Res])
	r := bufio.NewReader(bytes.NewReader(copyBuffer))
	return r
}

func (u *Uring) getCQE() *UringCQE {
	for {
		head, tail := atomic.LoadUint32(u.CQ.Head), atomic.LoadUint32(u.CQ.Tail)

		if head == tail {
			slog.Debug("No completion events found, but io_uring_enter did not block")
			break
		}

		if atomic.CompareAndSwapUint32(u.CQ.Head, head, head+1) {
			cqes := unsafe.Slice((*UringCQE)(unsafe.Pointer(u.CQ.CQEs)), *u.CQ.Entries)
			cqe := cqes[head&*u.CQ.Mask]

			if cqe.Res < 0 {
				err := unix.Errno(cqe.Res)
				slog.DebugContext(context.TODO(), "err", fmt.Sprintf("%s : %s", unix.ErrnoName(err), err.Error()))
				return nil
			}

			return &cqe
		}

		runtime.Gosched()
	}

	return nil
}

func (u *Uring) Close() {
	unix.Close(int(u.Fd))
}

type uringParams struct {
	SqEntry      uint32 // エントリの数
	CqEntry      uint32 // エントリの数
	Flags        uint32 // uringのオプションフラグ
	SqThreadCPU  uint32
	SqThreadIdle uint32
	Features     uint32 // uringの機能フラグ
	WqFd         uint32
	Resv         [3]uint32
	SQOffsets    sqOffsets
	CQOffsets    cqOffsets
}

type sqOffsets struct {
	Head        uint32 // カーネルが処理済みのSQEの位置
	Tail        uint32 // ユーザーがSQEを追加する位置
	RingMask    uint32 // リング循環用のマスク ( 最大値 - 1 )
	RingEntries uint32 // リングの総容量
	Flags       uint32 // フラグ
	Dropped     uint32 // 処理されなかったリクエストの数
	Array       uint32 // SQEへのポインタ
	Resv1       uint32
	UserAddr    uint64
}

type cqOffsets struct {
	Head        uint32 // ユーザーが読み取った位置
	Tail        uint32 // カーネルが完了した位置
	RingMask    uint32
	RingEntries uint32
	Overflow    uint32
	CQEs        uint32 // CQEへのポインタ
	Flags       uint32
	Resv1       uint32
	UserAddr    uint64
}
