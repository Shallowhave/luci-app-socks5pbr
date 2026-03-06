package socks5

import (
	"encoding/binary"
	"errors"
	"fmt"
	"net"
)

var ErrUDPFragmentNotSupported = errors.New("socks5: UDP fragmentation not supported")

// PackUDPDatagram builds a SOCKS5 UDP request datagram.
func PackUDPDatagram(dst *net.UDPAddr, payload []byte) ([]byte, error) {
	if dst == nil {
		return nil, errors.New("socks5: nil dst")
	}
	buf := make([]byte, 0, 32+len(payload))
	// RSV(2) + FRAG(1)
	buf = append(buf, 0x00, 0x00, 0x00)

	ip := dst.IP
	if ip4 := ip.To4(); ip4 != nil {
		buf = append(buf, atypV4)
		buf = append(buf, ip4...)
	} else if ip6 := ip.To16(); ip6 != nil {
		buf = append(buf, atypV6)
		buf = append(buf, ip6...)
	} else {
		return nil, fmt.Errorf("socks5: bad dst ip %v", ip)
	}
	var pb [2]byte
	binary.BigEndian.PutUint16(pb[:], uint16(dst.Port))
	buf = append(buf, pb[:]...)
	buf = append(buf, payload...)
	return buf, nil
}

// UnpackUDPDatagram parses a SOCKS5 UDP response datagram and returns (dst, payload).
func UnpackUDPDatagram(b []byte) (*net.UDPAddr, []byte, error) {
	if len(b) < 4 {
		return nil, nil, ErrBadReply
	}
	// RSV
	if b[0] != 0 || b[1] != 0 {
		return nil, nil, ErrBadReply
	}
	frag := b[2]
	if frag != 0x00 {
		return nil, nil, ErrUDPFragmentNotSupported
	}
	atyp := b[3]
	off := 4

	var ip net.IP
	switch atyp {
	case atypV4:
		if len(b) < off+4+2 {
			return nil, nil, ErrBadReply
		}
		ip = net.IPv4(b[off], b[off+1], b[off+2], b[off+3])
		off += 4
	case atypV6:
		if len(b) < off+16+2 {
			return nil, nil, ErrBadReply
		}
		ip = net.IP(b[off : off+16])
		off += 16
	case atypDomain:
		if len(b) < off+1 {
			return nil, nil, ErrBadReply
		}
		l := int(b[off])
		off++
		if len(b) < off+l+2 {
			return nil, nil, ErrBadReply
		}
		// Domain replies are allowed; we drop domain and return nil IP.
		off += l
	default:
		return nil, nil, ErrBadReply
	}
	port := int(binary.BigEndian.Uint16(b[off : off+2]))
	off += 2
	return &net.UDPAddr{IP: ip, Port: port}, b[off:], nil
}

