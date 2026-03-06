//go:build !linux

package tproxy

import (
	"errors"
	"net"
)

var errNotSupported = errors.New("tproxy is only supported on linux")

// listenTCPTransparent returns a listener suitable for TPROXY.
func listenTCPTransparent(addr string) (net.Listener, error) {
	return nil, errNotSupported
}

// listenUDPTransparent returns a UDPConn suitable for TPROXY and able to receive original dst info.
func listenUDPTransparent(addr string) (*net.UDPConn, error) {
	return nil, errNotSupported
}

// getOriginalDstTCP extracts the original destination for a TPROXY-accepted TCP connection.
func getOriginalDstTCP(conn *net.TCPConn) (*net.TCPAddr, error) {
	return nil, errNotSupported
}

// readFromUDPOrigDst reads one UDP datagram and returns (n, src, origDst).
func readFromUDPOrigDst(pc *net.UDPConn, b []byte) (int, *net.UDPAddr, *net.UDPAddr, error) {
	return 0, nil, nil, errNotSupported
}

// dialUDPTransparentConn creates a UDPConn with local address set to localSrc (spoof source),
// connected to remote.
func dialUDPTransparentConn(localSrc *net.UDPAddr, remote *net.UDPAddr) (*net.UDPConn, error) {
	return nil, errNotSupported
}

