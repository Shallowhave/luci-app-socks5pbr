package tproxy

import (
	"log"
	"net"
	"sync"
)

type Config struct {
	TCPListenAddr string
	UDPListenAddr string
}

type Server struct {
	cfg    Config
	logger *log.Logger
	h      Handler

	tcpLn net.Listener
	udpPc *net.UDPConn

	closeOnce sync.Once
	closed    chan struct{}
}

func NewServer(cfg Config, logger *log.Logger, h Handler) (*Server, error) {
	if logger == nil {
		logger = log.Default()
	}
	return &Server{
		cfg:    cfg,
		logger: logger,
		h:      h,
		closed: make(chan struct{}),
	}, nil
}

func (s *Server) Start() error {
	tcpLn, err := listenTCPTransparent(s.cfg.TCPListenAddr)
	if err != nil {
		return err
	}
	s.tcpLn = tcpLn
	s.logger.Printf("TCP listening on %s", tcpLn.Addr())

	udpPc, err := listenUDPTransparent(s.cfg.UDPListenAddr)
	if err != nil {
		_ = tcpLn.Close()
		return err
	}
	s.udpPc = udpPc
	s.logger.Printf("UDP listening on %s", udpPc.LocalAddr())

	go s.acceptLoop()
	go s.udpLoop()

	return nil
}

func (s *Server) Close() error {
	var err error
	s.closeOnce.Do(func() {
		close(s.closed)
		if s.tcpLn != nil {
			if e := s.tcpLn.Close(); e != nil && err == nil {
				err = e
			}
		}
		if s.udpPc != nil {
			if e := s.udpPc.Close(); e != nil && err == nil {
				err = e
			}
		}
	})
	return err
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.tcpLn.Accept()
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			s.logger.Printf("tcp accept error: %v", err)
			continue
		}
		go s.handleTCP(conn)
	}
}

func (s *Server) handleTCP(conn net.Conn) {
	defer conn.Close()
	s.logger.Printf("new TCP conn from %s", conn.RemoteAddr())
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		return
	}
	origDst, err := getOriginalDstTCP(tcpConn)
	if err != nil {
		s.logger.Printf("get original dst (tcp): %v", err)
		return
	}
	if s.h != nil {
		src, _ := tcpConn.RemoteAddr().(*net.TCPAddr)
		s.h.HandleTCP(tcpConn, src, origDst)
	}
}

func (s *Server) udpLoop() {
	buf := make([]byte, 64*1024)
	for {
		n, src, origDst, err := readFromUDPOrigDst(s.udpPc, buf)
		if err != nil {
			select {
			case <-s.closed:
				return
			default:
			}
			s.logger.Printf("udp read error: %v", err)
			continue
		}
		data := make([]byte, n)
		copy(data, buf[:n])
		go s.handleUDP(src, origDst, data)
	}
}

func (s *Server) handleUDP(src *net.UDPAddr, origDst *net.UDPAddr, data []byte) {
	if origDst == nil {
		s.logger.Printf("UDP packet from %s, len=%d (no orig dst)", src, len(data))
		return
	}
	if s.h == nil {
		return
	}
	sendReply := func(localSrc *net.UDPAddr, remote *net.UDPAddr, payload []byte) error {
		c, err := dialUDPTransparentConn(localSrc, remote)
		if err != nil {
			return err
		}
		defer c.Close()
		_, err = c.Write(payload)
		return err
	}
	s.h.HandleUDP(sendReply, src, origDst, data)
}

