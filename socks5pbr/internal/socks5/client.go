package socks5

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

const (
	socksVer5 = 0x05

	methodNoAuth  = 0x00
	methodUserPass = 0x02
	methodNoAccept = 0xff

	cmdConnect      = 0x01
	cmdUDPAssociate = 0x03

	atypV4   = 0x01
	atypDomain = 0x03
	atypV6   = 0x04

	repSucceeded = 0x00
)

type Dialer struct {
	Server  Server
	Timeout time.Duration
}

func (d *Dialer) dialServer(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	to := d.Timeout
	if to == 0 {
		to = 10 * time.Second
	}
	nd := net.Dialer{Timeout: to}
	conn, err := nd.DialContext(ctx, "tcp", d.Server.Addr())
	if err != nil {
		return nil, nil, err
	}
	_ = conn.SetDeadline(time.Now().Add(to))
	return conn, bufio.NewReader(conn), nil
}

func (d *Dialer) DialTCP(ctx context.Context, target *net.TCPAddr) (net.Conn, error) {
	conn, br, err := d.dialServer(ctx)
	if err != nil {
		return nil, err
	}
	if err := d.handshake(br, conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := sendRequest(conn, cmdConnect, tcpAddrToSocksAddr(target)); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := readReply(br); err != nil {
		_ = conn.Close()
		return nil, err
	}
	// clear deadline for long-lived stream
	_ = conn.SetDeadline(time.Time{})
	return conn, nil
}

type UDPAssoc struct {
	Relay   *net.UDPAddr
	PC      *net.UDPConn
	Dialer  *Dialer
}

func (a *UDPAssoc) Close() error {
	if a == nil || a.PC == nil {
		return nil
	}
	return a.PC.Close()
}

func (a *UDPAssoc) SendTo(dst *net.UDPAddr, payload []byte) error {
	pkt, err := PackUDPDatagram(dst, payload)
	if err != nil {
		return err
	}
	_, err = a.PC.Write(pkt)
	return err
}

func (a *UDPAssoc) ReadFrom(buf []byte) (*net.UDPAddr, int, error) {
	n, err := a.PC.Read(buf)
	if err != nil {
		return nil, 0, err
	}
	dst, payload, err := UnpackUDPDatagram(buf[:n])
	if err != nil {
		return nil, 0, err
	}
	copy(buf, payload)
	return dst, len(payload), nil
}

func (d *Dialer) DialUDPAssociate(ctx context.Context) (*UDPAssoc, error) {
	conn, br, err := d.dialServer(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	if err := d.handshake(br, conn); err != nil {
		return nil, err
	}

	// Request UDP associate; BND.ADDR may be 0.0.0.0:0.
	if err := sendRequest(conn, cmdUDPAssociate, socksAddr{Atyp: atypV4, Host: net.IPv4zero, Port: 0}); err != nil {
		return nil, err
	}

	relayHost, relayPort, err := readReplyAddr(br)
	if err != nil {
		return nil, err
	}

	relayUDP, err := net.ResolveUDPAddr("udp", net.JoinHostPort(relayHost, fmt.Sprintf("%d", relayPort)))
	if err != nil {
		return nil, err
	}

	pc, err := net.DialUDP("udp", nil, relayUDP)
	if err != nil {
		return nil, err
	}

	_ = pc.SetDeadline(time.Time{})
	return &UDPAssoc{Relay: relayUDP, PC: pc, Dialer: d}, nil
}

func (d *Dialer) handshake(br *bufio.Reader, w io.Writer) error {
	needAuth := d.Server.Auth != nil && (d.Server.Auth.Username != "" || d.Server.Auth.Password != "")
	methods := []byte{methodNoAuth}
	if needAuth {
		methods = append(methods, methodUserPass)
	}

	// greeting: VER, NMETHODS, METHODS...
	g := make([]byte, 0, 2+len(methods))
	g = append(g, socksVer5, byte(len(methods)))
	g = append(g, methods...)
	if _, err := w.Write(g); err != nil {
		return err
	}

	ver, err := br.ReadByte()
	if err != nil {
		return err
	}
	if ver != socksVer5 {
		return ErrBadReply
	}
	m, err := br.ReadByte()
	if err != nil {
		return err
	}
	switch m {
	case methodNoAuth:
		return nil
	case methodUserPass:
		if !needAuth {
			return ErrUnsupportedAuthMethod
		}
		return userPassAuth(br, w, d.Server.Auth.Username, d.Server.Auth.Password)
	case methodNoAccept:
		return ErrUnsupportedAuthMethod
	default:
		return ErrUnsupportedAuthMethod
	}
}

func userPassAuth(br *bufio.Reader, w io.Writer, u, p string) error {
	// RFC1929: VER=0x01, ULEN, UNAME, PLEN, PASSWD
	if len(u) > 255 || len(p) > 255 {
		return fmt.Errorf("socks5: username/password too long")
	}
	req := make([]byte, 0, 3+len(u)+len(p))
	req = append(req, 0x01, byte(len(u)))
	req = append(req, []byte(u)...)
	req = append(req, byte(len(p)))
	req = append(req, []byte(p)...)
	if _, err := w.Write(req); err != nil {
		return err
	}
	ver, err := br.ReadByte()
	if err != nil {
		return err
	}
	if ver != 0x01 {
		return ErrBadReply
	}
	st, err := br.ReadByte()
	if err != nil {
		return err
	}
	if st != 0x00 {
		return ErrAuthFailed
	}
	return nil
}

type socksAddr struct {
	Atyp byte
	Host any // net.IP or string
	Port int
}

func tcpAddrToSocksAddr(a *net.TCPAddr) socksAddr {
	if a.IP == nil {
		return socksAddr{Atyp: atypDomain, Host: "0.0.0.0", Port: a.Port}
	}
	if ip4 := a.IP.To4(); ip4 != nil {
		return socksAddr{Atyp: atypV4, Host: ip4, Port: a.Port}
	}
	return socksAddr{Atyp: atypV6, Host: a.IP, Port: a.Port}
}

func sendRequest(w io.Writer, cmd byte, addr socksAddr) error {
	buf := make([]byte, 0, 32)
	buf = append(buf, socksVer5, cmd, 0x00, addr.Atyp)
	switch addr.Atyp {
	case atypV4:
		ip := addr.Host.(net.IP).To4()
		buf = append(buf, ip...)
	case atypV6:
		ip := addr.Host.(net.IP).To16()
		buf = append(buf, ip...)
	case atypDomain:
		host := addr.Host.(string)
		if len(host) > 255 {
			return fmt.Errorf("socks5: domain too long")
		}
		buf = append(buf, byte(len(host)))
		buf = append(buf, []byte(host)...)
	default:
		return fmt.Errorf("socks5: unknown atyp %d", addr.Atyp)
	}
	var p [2]byte
	binary.BigEndian.PutUint16(p[:], uint16(addr.Port))
	buf = append(buf, p[:]...)
	_, err := w.Write(buf)
	return err
}

func readReply(br *bufio.Reader) error {
	_, _, err := readReplyAddr(br)
	return err
}

func readReplyAddr(br *bufio.Reader) (host string, port int, err error) {
	h := make([]byte, 4)
	if _, err := io.ReadFull(br, h); err != nil {
		return "", 0, err
	}
	if h[0] != socksVer5 {
		return "", 0, ErrBadReply
	}
	if h[1] != repSucceeded {
		return "", 0, fmt.Errorf("socks5: reply code %d", h[1])
	}
	atyp := h[3]
	switch atyp {
	case atypV4:
		b := make([]byte, 4)
		if _, err := io.ReadFull(br, b); err != nil {
			return "", 0, err
		}
		host = net.IP(b).String()
	case atypV6:
		b := make([]byte, 16)
		if _, err := io.ReadFull(br, b); err != nil {
			return "", 0, err
		}
		host = net.IP(b).String()
	case atypDomain:
		l, err := br.ReadByte()
		if err != nil {
			return "", 0, err
		}
		b := make([]byte, int(l))
		if _, err := io.ReadFull(br, b); err != nil {
			return "", 0, err
		}
		host = string(b)
	default:
		return "", 0, ErrBadReply
	}
	var pb [2]byte
	if _, err := io.ReadFull(br, pb[:]); err != nil {
		return "", 0, err
	}
	port = int(binary.BigEndian.Uint16(pb[:]))
	return host, port, nil
}

