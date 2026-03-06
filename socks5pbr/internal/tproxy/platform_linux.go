//go:build linux

package tproxy

import (
	"context"
	"encoding/binary"
	"errors"
	"net"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// These constants are not exposed by x/sys/unix on all platforms.
const (
	soOriginalDst      = 80 // SO_ORIGINAL_DST
	ipRecvOrigDstAddr  = 20 // IP_RECVORIGDSTADDR
	ipv6RecvOrigDstAddr = 74 // IPV6_RECVORIGDSTADDR
)

func listenTCPTransparent(addr string) (net.Listener, error) {
	var lc net.ListenConfig
	lc.Control = func(network, address string, c syscall.RawConn) error {
		var serr error
		if err := c.Control(func(fd uintptr) {
			serr = setTransparentSockopts(int(fd))
			if serr != nil {
				return
			}
			_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
			_ = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		}); err != nil {
			return err
		}
		return serr
	}
	return lc.Listen(context.Background(), "tcp", addr)
}

func listenUDPTransparent(addr string) (*net.UDPConn, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}
	// ensure close on error
	ok := false
	defer func() {
		if !ok {
			_ = unix.Close(fd)
		}
	}()

	if err := setTransparentSockopts(fd); err != nil {
		return nil, err
	}
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)

	// Request original destination address in control messages.
	_ = unix.SetsockoptInt(fd, unix.SOL_IP, ipRecvOrigDstAddr, 1)
	_ = unix.SetsockoptInt(fd, unix.SOL_IPV6, ipv6RecvOrigDstAddr, 1)

	sa := &unix.SockaddrInet4{Port: udpAddr.Port}
	if ip4 := udpAddr.IP.To4(); ip4 != nil {
		copy(sa.Addr[:], ip4)
	} else if udpAddr.IP == nil || udpAddr.IP.IsUnspecified() {
		// bind 0.0.0.0
	} else {
		return nil, errors.New("udp transparent listen: only ipv4 supported in first stage")
	}
	if err := unix.Bind(fd, sa); err != nil {
		return nil, err
	}

	f := os.NewFile(uintptr(fd), "udp-transparent")
	if f == nil {
		return nil, errors.New("os.NewFile returned nil")
	}
	defer f.Close()

	c, err := net.FileConn(f)
	if err != nil {
		return nil, err
	}
	uc, ok2 := c.(*net.UDPConn)
	if !ok2 {
		c.Close()
		return nil, errors.New("file conn is not UDPConn")
	}

	ok = true
	return uc, nil
}

func setTransparentSockopts(fd int) error {
	if err := unix.SetsockoptInt(fd, unix.SOL_IP, unix.IP_TRANSPARENT, 1); err != nil {
		return err
	}
	// We need to receive packets with non-local destination.
	_ = unix.SetsockoptInt(fd, unix.SOL_IP, unix.IP_RECVTOS, 1)
	return nil
}

func getOriginalDstTCP(conn *net.TCPConn) (*net.TCPAddr, error) {
	rc, err := conn.SyscallConn()
	if err != nil {
		return nil, err
	}
	var (
		dst *net.TCPAddr
		ge  error
	)
	if err := rc.Control(func(fd uintptr) {
		dst, ge = getOriginalDstTCP4(int(fd))
	}); err != nil {
		return nil, err
	}
	if ge != nil {
		return nil, ge
	}
	return dst, nil
}

func getOriginalDstTCP4(fd int) (*net.TCPAddr, error) {
	// SO_ORIGINAL_DST returns struct sockaddr_in.
	var buf [128]byte
	l := uint32(len(buf))
	_, _, errno := unix.Syscall6(
		unix.SYS_GETSOCKOPT,
		uintptr(fd),
		uintptr(unix.SOL_IP),
		uintptr(soOriginalDst),
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&l)),
		0,
	)
	if errno != 0 {
		return nil, errno
	}
	if l < 8 {
		return nil, errors.New("SO_ORIGINAL_DST: short response")
	}
	port := int(binary.BigEndian.Uint16(buf[2:4]))
	ip := net.IPv4(buf[4], buf[5], buf[6], buf[7])
	return &net.TCPAddr{IP: ip, Port: port}, nil
}

func readFromUDPOrigDst(pc *net.UDPConn, b []byte) (int, *net.UDPAddr, *net.UDPAddr, error) {
	oob := make([]byte, 256)
	n, oobn, _, src, err := pc.ReadMsgUDP(b, oob)
	if err != nil {
		return 0, nil, nil, err
	}
	msgs, err := unix.ParseSocketControlMessage(oob[:oobn])
	if err != nil {
		return n, src, nil, nil
	}
	for _, m := range msgs {
		// Linux: IP_RECVORIGDSTADDR delivers cmsg level SOL_IP type IP_ORIGDSTADDR (20).
		if m.Header.Level == unix.SOL_IP && m.Header.Type == ipRecvOrigDstAddr && len(m.Data) >= 8 {
			// Data is sockaddr_in:
			// [0:2] family, [2:4] port (network order), [4:8] addr
			port := int(binary.BigEndian.Uint16(m.Data[2:4]))
			ip := net.IPv4(m.Data[4], m.Data[5], m.Data[6], m.Data[7])
			return n, src, &net.UDPAddr{IP: ip, Port: port}, nil
		}
	}
	return n, src, nil, nil
}

func dialUDPTransparentConn(localSrc *net.UDPAddr, remote *net.UDPAddr) (*net.UDPConn, error) {
	if localSrc == nil || remote == nil {
		return nil, errors.New("nil udp addr")
	}
	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_DGRAM, unix.IPPROTO_UDP)
	if err != nil {
		return nil, err
	}
	ok := false
	defer func() {
		if !ok {
			_ = unix.Close(fd)
		}
	}()

	if err := setTransparentSockopts(fd); err != nil {
		return nil, err
	}
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)
	_ = unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)

	lsa := &unix.SockaddrInet4{Port: localSrc.Port}
	if ip4 := localSrc.IP.To4(); ip4 != nil {
		copy(lsa.Addr[:], ip4)
	} else {
		copy(lsa.Addr[:], net.IPv4zero.To4())
	}
	if err := unix.Bind(fd, lsa); err != nil {
		return nil, err
	}

	rsa := &unix.SockaddrInet4{Port: remote.Port}
	if ip4 := remote.IP.To4(); ip4 != nil {
		copy(rsa.Addr[:], ip4)
	} else {
		return nil, errors.New("remote must be ipv4 in first stage")
	}
	if err := unix.Connect(fd, rsa); err != nil {
		return nil, err
	}

	f := os.NewFile(uintptr(fd), "udp-transparent-reply")
	if f == nil {
		return nil, errors.New("os.NewFile returned nil")
	}
	defer f.Close()
	c, err := net.FileConn(f)
	if err != nil {
		return nil, err
	}
	uc, ok2 := c.(*net.UDPConn)
	if !ok2 {
		c.Close()
		return nil, errors.New("file conn is not UDPConn")
	}
	ok = true
	return uc, nil
}

