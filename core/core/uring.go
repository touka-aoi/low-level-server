//go:build linux

package core

import (
	"errors"
	"log/slog"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	toukaerrors "github.com/touka-aoi/low-level-server/core/errors"
	"golang.org/x/sys/unix"
)

const (
	MaxBufferSize = 20 * 1024 // 20kib
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
	Fd                 int32
	SQ                 SQ
	CQ                 CQ
	Buffer             []byte
	pRingRegBuffer     []byte     // 使用しない GC対策
	pRingBuffer        []uringBuf // mmapしたバッファのポインタ
	pRingData          []byte     // mmapしたデータのポインタ
	pRingBufferBasePtr uintptr    // バッファのベースアドレス

	// ヘッダーとかやってみるかぁ
	Msghdr unix.Msghdr
	Addr   []byte
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
		slog.Error("IO_URING_SETUP failed", "errno", errno, "err", errno.Error())
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
		slog.Error("Mmap failed", "err", err, "errno", err.Error())
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
		slog.Error("Mmap failed", "err", err, "errno", err.Error())
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
	}

	return uring

}

func (u *Uring) RegisterRingBuffer(entries, maxBufferSize, bufferGroupID int) {
	ringSize := (unsafe.Sizeof(uringBuf{}) + MaxBufferSize) * uintptr(entries)
	data, err := unix.Mmap(-1, 0, int(ringSize), unix.PROT_READ|unix.PROT_WRITE, unix.MAP_ANONYMOUS|unix.MAP_PRIVATE)
	if err != nil {
		slog.Error("Mmap failed for ring buffer", "err", err, "errno", err.Error())
		panic(err)
	}
	pRingPtr := unsafe.Pointer(unsafe.SliceData(data))
	pRingBuffer := unsafe.Slice((*uringBuf)(pRingPtr), entries)
	bufferBasePtr := uintptr(pRingPtr) + (uintptr(entries) * unsafe.Sizeof(uringBuf{}))

	reg := &uringBufReg{
		RingAddr:    uint64(uintptr(pRingPtr)),
		RingEntries: uint32(entries),
		Bgid:        uint16(bufferGroupID),
	}

	_, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_REGISTER,
		uintptr(u.Fd),
		IORING_REGISTER_PBUF_RING,
		uintptr(unsafe.Pointer(reg)),
		1,
		0,
		0,
	)
	if errno != 0 {
		slog.Error("IO_URING_REGISTER failed", "errno", errno, "err", errno.Error())
		panic(errno)
	}

	for i := 0; i < entries; i++ {
		index := (int(pRingBuffer[0].Resv) + i) & (entries - 1)
		pRingBuffer[index].Addr = uint64(bufferBasePtr + uintptr(i)*MaxBufferSize)
		pRingBuffer[index].Len = uint32(maxBufferSize)
		pRingBuffer[index].Bid = uint16(i)
	}

	u.pRingData = data
	u.pRingBuffer = pRingBuffer
	u.pRingBufferBasePtr = bufferBasePtr
	slog.Debug("Registered ring buffer", "entries", entries, "maxBufferSize", maxBufferSize, "bufferGroupID", bufferGroupID)
	u.advancePbufRing(uint16(entries))
}

func (u *Uring) GetRingBuffer(index uint16) []byte {
	ptr := u.pRingBufferBasePtr + uintptr(index)*MaxBufferSize
	return unsafe.Slice((*byte)(unsafe.Pointer(ptr)), MaxBufferSize)
}

func (u *Uring) advancePbufRing(count uint16) {
	newTail := u.pRingBuffer[0].Resv + count
	u.pRingBuffer[0].Resv = newTail
}

func (u *Uring) AcceptMultishot(fd int32, userData uint64) *UringSQE {
	op := &UringSQE{
		Opcode:   IORING_OP_ACCEPT,
		Ioprio:   IORING_ACCEPT_MULTISHOT, // https://lore.kernel.org/lkml/a41a1f47-ad05-3245-8ac8-7d8e95ebde44@kernel.dk/t/
		Fd:       fd,
		UserData: userData,
	}
	return op
}

func (u *Uring) RecvFrom(fd int32, userData uint64) *UringSQE {
	addr := make([]byte, unix.SizeofSockaddrInet6)
	u.Addr = addr
	u.Msghdr = unix.Msghdr{
		Name:       &addr[0],
		Namelen:    unix.SizeofSockaddrInet6,
		Control:    nil,
		Controllen: 0,
	}
	op := &UringSQE{
		Opcode:   IORING_OP_RECVMSG,
		Ioprio:   0, //NOTE: MultiShotを使うとMsgHdrがカーネル側で初期化されるので取得できない
		Flags:    IOSQE_BUFFER_SELECT,
		BufIndex: 1,
		Fd:       fd,
		UserData: userData,
		Address:  uint64(uintptr(unsafe.Pointer(&u.Msghdr))),
	}
	slog.Debug("RecvFrom", "opcode", op.Opcode, "ioprio", op.Ioprio, "fd", op.Fd)
	return op
}

func (u *Uring) Cancel(fd int32, cancelTarget uint64, userData uint64) {
	op := &UringSQE{
		Opcode:   IORING_OP_ASYNC_CANCEL,
		Address:  cancelTarget,
		UserData: userData,
	}
	u.Submit(op)
}

func (u *Uring) ReadMultishot(fd int32, userData uint64) *UringSQE {
	op := &UringSQE{
		// Opcode:   IORING_OP_READ_MULTISHOT,
		Opcode:   IORING_OP_READ, // 一旦通常のREADで試す
		Flags:    IOSQE_BUFFER_SELECT,
		BufIndex: 1,
		Fd:       fd,
		UserData: userData,
	}
	slog.Debug("ReadMultishot", "opcode", op.Opcode, "flags", op.Flags, "bufIndex", op.BufIndex)
	return op
}

func (u *Uring) Submit(op *UringSQE) {
	_ = u.pushSQE(op)
	u.sendSQE()
}

func (u *Uring) Write(fd int32, buffer []byte, userData uint64) {
	op := &UringSQE{
		Opcode:   IORING_OP_WRITE,
		Fd:       fd,
		Address:  uint64(uintptr(unsafe.Pointer(&buffer[0]))),
		Len:      uint32(len(buffer)),
		UserData: userData,
	}

	u.Submit(op)
}

func (u *Uring) RegisterRead(fd int32, userData uint64) {
	op := &UringSQE{
		Opcode:   IORING_OP_READ_MULTISHOT,
		Fd:       fd,
		Flags:    IOSQE_BUFFER_SELECT,
		UserData: userData,
		BufIndex: 1,
	}

	u.Submit(op)
}

func (u *Uring) pushSQE(op *UringSQE) error {

	head, tail := atomic.LoadUint32(u.SQ.Head), atomic.LoadUint32(u.SQ.Tail)
	if tail-head >= *u.SQ.Entries {
		slog.Warn("sq entries full", "tail", tail, "head", head)
		return nil
	}

	for {
		tail := atomic.LoadUint32(u.SQ.Tail)

		if atomic.CompareAndSwapUint32(u.SQ.Tail, tail, tail+1) {
			sqe := unsafe.Slice((*UringSQE)(unsafe.Pointer(u.SQ.SQEPtr)), *u.SQ.Entries)
			sqe[tail&*u.SQ.Mask] = *op

			array := unsafe.Slice((*uint32)(unsafe.Pointer(u.SQ.ArrayPtr)), *u.SQ.Entries)
			array[tail&*u.SQ.Mask] = tail

			break
		}
		runtime.Gosched()
	}
	return nil
}

func (u *Uring) sendSQE() {
	_, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_ENTER,
		uintptr(u.Fd),
		1,
		0,
		0,
		0,
		0)

	if errno != 0 {
		slog.Error("Uring failed", "errno", errno, "err", errno.Error())
		panic(errno)
	}
}

func (u *Uring) WaitEvent() error {
	_, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_ENTER,
		uintptr(u.Fd),
		0,
		1,
		IORING_ENTER_GETEVENTS,
		0,
		0)

	if errno != 0 {
		slog.Error("syscall sys_io_uring_enter failed", "err", errno.Error(), "errno", int(errno))
		return errno
	}
	return nil
}

func (u *Uring) Timeout(d time.Duration, userData uint64) {
	timeSpec := unix.NsecToTimespec(d.Nanoseconds())
	op := &UringSQE{
		Opcode:    IORING_OP_TIMEOUT,
		Fd:        -1,
		Address:   uint64(uintptr(unsafe.Pointer(&timeSpec))),
		UserFlags: IORING_TIMEOUT_ETIME_SUCCESS, //FIXME 設定してもETIMEが必ず出る
		Len:       1,
		Offset:    0,
		UserData:  userData,
	}

	u.Submit(op)
}

func (u *Uring) TimeoutWithMultiShot(d time.Duration, userData uint64) {
	timeSpec := unix.NsecToTimespec(d.Nanoseconds())
	op := &UringSQE{
		Opcode:    IORING_OP_TIMEOUT,
		Fd:        -1,
		Address:   uint64(uintptr(unsafe.Pointer(&timeSpec))),
		UserFlags: IORING_TIMEOUT_ETIME_SUCCESS | IORING_TIMEOUT_MULTISHOT, //FIXME 設定してもETIMEが必ず出る
		Len:       1,
		Offset:    0,
		UserData:  userData,
	}

	u.Submit(op)
}

func (u *Uring) WaitEventWithTimeout(d time.Duration) error {
	timeSpec := unix.NsecToTimespec(d.Nanoseconds())
	getEventsArgs := &uringGetEventArgs{
		sigMask:    0,
		sigMask_sz: 0,
		pad:        0,
		ts:         uint64(uintptr(unsafe.Pointer(&timeSpec))),
	}

	t1 := time.Now()

	_, _, errno := unix.Syscall6(
		unix.SYS_IO_URING_ENTER,
		uintptr(u.Fd),
		0,
		1,
		IORING_ENTER_GETEVENTS|IORING_ENTER_EXT_ARG,
		uintptr(unsafe.Pointer(getEventsArgs)),
		unsafe.Sizeof(*getEventsArgs))

	if errno != 0 {
		if errors.Is(errno, unix.ETIME) {
			return toukaerrors.ErrWouldBlock
		}
		t2 := time.Now()
		slog.Debug("wait event", "d", d, "elapsed", t2.Sub(t1))
		slog.Error("syscall sys_io_uring_enter failed", "err", errno.Error(), "errno", int(errno))
		return errno
	}
	return nil
}

func (u *Uring) PeekBatchEvents(batch uint32) ([]*UringCQE, error) {
	ready := u.cqReady()

	if ready < 1 {
		return nil, toukaerrors.ErrWouldBlock
	}

	if batch > ready {
		batch = ready
	}

	cqes := make([]*UringCQE, 0, batch)
	for i := uint32(0); i < ready; i++ {
		cqe := u.getCQE()
		if cqe == nil {
			panic("unexpected erorr: cqe is nil")
		}
		cqes = append(cqes, cqe)
	}

	return cqes, nil
}

func (u *Uring) cqReady() uint32 {
	head, tail := atomic.LoadUint32(u.CQ.Head), atomic.LoadUint32(u.CQ.Tail)
	if head == tail {
		return 0
	}
	return tail - head
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

			return &cqe
		}

		runtime.Gosched()
	}

	return nil
}

func (u *Uring) Read(buffer []byte) {
	// copy(buffer, u.pRingBuffer[:len(buffer)])
}

func (u *Uring) Close() error {
	err := unix.Close(int(u.Fd))
	if err != nil {
		return err
	}
	return nil
}

// sendCQEはadvanceCQEとセットで使用する。スレッドセーフではないので注意
// CQEを取得する (headは進めない)
func (u *Uring) seenCQE() *UringCQE {
	head, tail := atomic.LoadUint32(u.CQ.Head), atomic.LoadUint32(u.CQ.Tail)

	if head == tail {
		slog.Debug("No completion events found, but io_uring_enter did not block")
		return nil
	}

	cqes := unsafe.Slice((*UringCQE)(unsafe.Pointer(u.CQ.CQEs)), *u.CQ.Entries)
	cqe := cqes[head&*u.CQ.Mask]

	return &cqe
}

// CQEを進める
func (u *Uring) advanceCQE() {
	for {
		head := atomic.LoadUint32(u.CQ.Head)
		if atomic.CompareAndSwapUint32(u.CQ.Head, head, head+1) {
			break
		}
		runtime.Gosched()
	}
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

type uringBufReg struct {
	RingAddr    uint64
	RingEntries uint32
	Bgid        uint16
	Flags       uint16
	Resv        [3]uint64
}

// 16バイト
type uringBuf struct {
	Addr uint64
	Len  uint32
	Bid  uint16
	Resv uint16
}

type uringBufTail struct {
	Resv1 uint64
	Resv2 uint32
	Resv3 uint16
	Tail  uint16
}

type uringGetEventArgs struct {
	sigMask    uint64
	sigMask_sz uint32
	pad        uint32
	ts         uint64
}

const (
	IORING_OFF_SQ_RING    int64 = 0
	ORING_OFF_CQ_RING     int64 = 0x8000000
	IORING_OFF_SQES       int64 = 0x10000000
	IORING_OFF_PBUF_RING  int64 = 0x80000000
	IORING_OFF_PBUF_SHIFT int64 = 16
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
	IORING_CQE_F_BUFFER = 1 << iota
	IORING_CQE_F_MORE
	IORING_CQE_F_SOCK_NONEMPTY
	IORING_CQE_F_NOTIF
	IORING_CQE_F_BUF_MORE
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

const (
	IOSQE_FIXED_FILE = 1 << iota
	IOSQE_IO_DRAIN
	IOSQE_IO_LINK
	IOSQE_IO_HARDLINK
	IOSQE_ASYNC
	IOSQE_BUFFER_SELECT
	IOSQE_CQE_SKIP_SUCCESS
)

// accept flags stored in sqe->ioprio
// https://lore.kernel.org/lkml/a41a1f47-ad05-3245-8ac8-7d8e95ebde44@kernel.dk/t/
const (
	IORING_ACCEPT_MULTISHOT = 1 << iota
	IORING_ACCEPT_DONTWAIT
	IORING_ACCEPT_POLL_FIRST
)

const (
	IORING_RECVSEND_POLL_FIRST = 1 << iota
	IORING_RECV_MULTISHOT
)

const (
	IORING_FEAT_SINGLE_MMAP = 1 << 0
)

// https://github.com/axboe/liburing/blob/c5eead2659ef5ea86ef8c78410fa42d9bea976c9/src/include/liburing/io_uring.h#L565
const (
	IORING_REGISTER_PBUF_RING = 22
)

const (
	IOU_PBUF_RING_MMAP = 1 << iota
	IOU_PBUF_RING_INC
)

const (
	IORING_CQE_BUFFER_SHIFT = 16
)

const (
	IORING_TIMEOUT_ABS           = 1 << 0
	IORING_TIMEOUT_REALTIME      = 1 << 3
	IORING_TIMEOUT_ETIME_SUCCESS = 1 << 5
	IORING_TIMEOUT_MULTISHOT     = 1 << 6
)
