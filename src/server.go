package server

import (
  "context"
  "encoding/binary"
  "fmt"
  "golang.org/x/sys/unix"
  "log"
  "log/slog"
  "net"
  "net/netip"
  "runtime"
  "sync/atomic"
  "unsafe"
)

const maxConnection = 4096

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

const (
	IORING_FEAT_SINGLE_MMAP = 1 << 0
)

type server struct {
  connections chan Peer
}

type sockAddr struct {
	Family uint16
	Data   [14]byte
}

type UringParams struct {
	SqEntry      uint32 // エントリの数
	CqEntry      uint32 // エントリの数
	Flags        uint32 // uringのオプションフラグ
	SqThreadCPU  uint32
	SqThreadIdle uint32
	Features     uint32 // uringの機能フラグ
	WqFd         uint32
	Resv         [3]uint32
	SQOffsets    SQOffsets
	CQOffsets    CQOffsets
}

type SQOffsets struct {
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

type CQOffsets struct {
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
	Fd int32
	SQ SQ
	CQ CQ
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

type Peer struct {
	Fd      int32
	address []byte
	port    int
}

//func (p *Peer) Network() string {
//
//}
//
//func (p *Peer) String() string {
//
//}

func CreateUring(entries uint32) *Uring {
	params := UringParams{}
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
	}

	fmt.Println(uring)

	return uring

}

func Listen(ctx context.Context, address string) (*Socket, error) {
	addr, err := netip.ParseAddrPort(address)
	if err != nil {
		return nil, err
	}
	socket := CreateSocket()

	socket.Bind(addr)
	socket.Listen(ctx, maxConnection)

	return socket, nil
}

type Socket struct {
	Fd int32
}

func CreateSocket() *Socket {
	// システムコールを叩いてSocketを作る
	// ノンブロッキング、プロセス継承なしでよさそう、tcpを選びたいので、SOCK_STREAMの0番でいいのかな
	// 基本的にひとつのソケットタイプには一つのプロトコルが割り当てられる
	// AF_INET | SOCK_STREAM なので 0でいいはず
	// https://man7.org/linux/man-pages/man2/socket.2.html
	fd, _, errno := unix.Syscall6(
		unix.SYS_SOCKET,
		unix.AF_INET,
		unix.SOCK_STREAM|unix.SOCK_CLOEXEC,
		0,
		0,
		0,
		0)

	if fd < 0 {
		log.Printf("Socket failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}

	return &Socket{Fd: int32(fd)}
}

func (s *Socket) Bind(address netip.AddrPort) {
	// https://man7.org/linux/man-pages/man2/bind.2.html
	sockaddr := sockAddr{
		Family: unix.AF_INET,
	}

	// port
	port := address.Port()
	binary.BigEndian.PutUint16(sockaddr.Data[:], port)

	// address
	addr := address.Addr().AsSlice()
	for i := 0; i < len(addr); i++ {
		sockaddr.Data[2+i] = addr[i]
	}

	res, _, errno := unix.Syscall6(
		unix.SYS_BIND,
		uintptr(s.Fd),
		uintptr(unsafe.Pointer(&sockaddr)),
		uintptr(unsafe.Sizeof(sockaddr)),
		0,
		0,
		0)

	if res != 0 {
		log.Printf("Bind failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

func (s *Socket) Listen(ctx context.Context, maxConn int) {
	res, _, errno := unix.Syscall6(
		unix.SYS_LISTEN,
		uintptr(s.Fd),
		uintptr(maxConn),
		0,
		0,
		0,
		0)

	if res != 0 {
		log.Printf("Listen failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}
}

// クローズ処理？
func (s *Socket) Unbind() {

}

func Accept(ctx context.Context, socket *Socket) {
	uring := CreateUring(maxConnection)
	for {
		select {
		case <-ctx.Done():
			return
		default:
			uring.Accpet(socket)
		}
	}
	//TODO 受信したものをServeへ回す
}

func (u *Uring) Accpet(socket *Socket) {
	// Entryを作る
	//MEMO: ここで設定するfdはリスニングソケット(↑で作ったやつ)である必要がある
	sockaddr := sockAddr{}
	sockLen := uint32(unsafe.Sizeof(sockaddr))
	op := UringSQE{
		Opcode:   IORING_OP_ACCEPT,
		Fd:       socket.Fd,
		Offset:   uint64(uintptr(unsafe.Pointer(&sockLen))),
		Address:  uint64(uintptr(unsafe.Pointer(&sockaddr))),
		UserData: 1,
	}

	//MEMO: tailの取得, SQEの書き込みをatomicに行う
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

	slog.Debug("Send Accpet Operation")

	slog.Debug("Block Accpet Operation")
	res, _, errno = unix.Syscall6(
		unix.SYS_IO_URING_ENTER,
		uintptr(u.Fd),
		0,
		1,
		IORING_ENTER_GETEVENTS,
		0,
		0)
	slog.Debug("End Block")

	if res < 0 || errno != 0 {
		//TODO エラーとして返す
		log.Printf("Uring failed with errno: %d (%s)", errno, errno.Error())
		panic(errno)
	}

	// CQEを取得する
	for {
		head, tail := atomic.LoadUint32(u.CQ.Head), atomic.LoadUint32(u.CQ.Tail)
		if head == tail {
			slog.Debug("No completion events found, but io_uring_enter did not block")
			continue
		}

		if atomic.CompareAndSwapUint32(u.CQ.Head, head, head+1) {
			cqe := unsafe.Slice((*UringCQE)(unsafe.Pointer(u.CQ.CQEs)), *u.CQ.Entries)
			cq := cqe[head&*u.CQ.Mask]

			if cq.Res < 0 {
				err := unix.Errno(cq.Res)
				slog.DebugContext(context.TODO(), "err", fmt.Sprintf("%s : %s", unix.ErrnoName(err), err.Error()))
			}

			// IPv4
      switch sockaddr.Family {
      case unix.AF_INET:
        port := binary.BigEndian.Uint16(sockaddr.Data[0:2])
        addr := netip.AddrFrom4([4]byte(sockaddr.Data[2:6]))

        ip := netip.AddrPortFrom(addr, port)
        
        net.Conn()
        
        //TODO: net.Conn interfaceを満たすようにする
        
        peer := Peer{
        Fd: int32(cq.Res),
        Conn: conn
       }
       
       connections <- peer
      }
      
      
		}
		break
	}
}

func Serve() {

}
