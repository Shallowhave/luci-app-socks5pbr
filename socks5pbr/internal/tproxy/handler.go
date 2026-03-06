package tproxy

import "net"

type Handler interface {
	HandleTCP(client *net.TCPConn, src *net.TCPAddr, dst *net.TCPAddr)
	HandleUDP(sendReply func(localSrc *net.UDPAddr, remote *net.UDPAddr, payload []byte) error, src *net.UDPAddr, dst *net.UDPAddr, payload []byte)
}

