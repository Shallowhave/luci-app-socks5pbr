package router

import (
	"context"
	"errors"
	"io"
	"log"
	"net"
	"sync"
	"time"

	"socks5pbr/internal/config"
	"socks5pbr/internal/socks5"
)

type Router struct {
	logger *log.Logger

	mu        sync.RWMutex
	nodesByID map[string]config.Node
	rulesByIP map[string]string // src_ip -> nodeName

	udpMu     sync.Mutex
	udpSess   map[string]*udpSession
}

func New(logger *log.Logger) *Router {
	if logger == nil {
		logger = log.Default()
	}
	return &Router{
		logger:    logger,
		nodesByID: map[string]config.Node{},
		rulesByIP: map[string]string{},
		udpSess:   map[string]*udpSession{},
	}
}

func (r *Router) LoadConfig(cfg *config.Config) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	nodes := map[string]config.Node{}
	for _, n := range cfg.Nodes {
		nodes[n.Name] = n
	}
	rules := map[string]string{}
	for _, rule := range cfg.Rules {
		if !rule.Enabled {
			continue
		}
		if _, ok := nodes[rule.Node]; !ok {
			continue
		}
		rules[rule.SrcIP] = rule.Node
	}

	r.mu.Lock()
	r.nodesByID = nodes
	r.rulesByIP = rules
	r.mu.Unlock()
	return nil
}

func (r *Router) pickNodeForIP(ip net.IP) (config.Node, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	nid, ok := r.rulesByIP[ip.String()]
	if !ok {
		return config.Node{}, false
	}
	n, ok := r.nodesByID[nid]
	return n, ok
}

func (r *Router) HandleTCP(client *net.TCPConn, src *net.TCPAddr, dst *net.TCPAddr) {
	n, ok := r.pickNodeForIP(src.IP)
	if !ok {
		r.logger.Printf("tcp no rule for %s", src.IP)
		return
	}

	dialer := socks5.Dialer{
		Server: socks5.Server{
			Host: n.Server,
			Port: n.Port,
			Auth: &socks5.Auth{Username: n.Username, Password: n.Password},
		},
		Timeout: 10 * time.Second,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	up, err := dialer.DialTCP(ctx, dst)
	if err != nil {
		r.logger.Printf("tcp dial via %s failed: %v", n.Name, err)
		return
	}
	defer up.Close()

	// Bidirectional copy
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(up, client)
		if tc, ok := up.(*net.TCPConn); ok {
			_ = tc.CloseWrite()
		}
	}()
	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, up)
		_ = client.CloseWrite()
	}()
	wg.Wait()
}

type udpSession struct {
	key       string
	lastSeen  time.Time
	assoc     *socks5.UDPAssoc
	replyConn *net.UDPConn
}

func (r *Router) HandleUDP(sendReply func(localSrc *net.UDPAddr, remote *net.UDPAddr, payload []byte) error, src *net.UDPAddr, dst *net.UDPAddr, payload []byte) {
	n, ok := r.pickNodeForIP(src.IP)
	if !ok {
		return
	}

	key := src.String() + "->" + dst.String() + "@" + n.Name

	r.udpMu.Lock()
	sess := r.udpSess[key]
	if sess != nil {
		sess.lastSeen = time.Now()
	}
	r.udpMu.Unlock()

	if sess == nil {
		dialer := socks5.Dialer{
			Server: socks5.Server{
				Host: n.Server,
				Port: n.Port,
				Auth: &socks5.Auth{Username: n.Username, Password: n.Password},
			},
			Timeout: 10 * time.Second,
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		assoc, err := dialer.DialUDPAssociate(ctx)
		cancel()
		if err != nil {
			r.logger.Printf("udp associate via %s failed: %v", n.Name, err)
			return
		}
		sess = &udpSession{
			key:      key,
			lastSeen: time.Now(),
			assoc:    assoc,
		}
		r.udpMu.Lock()
		r.udpSess[key] = sess
		r.udpMu.Unlock()

		go r.udpRecvLoop(sendReply, sess, src, dst)
	}

	_ = sess.assoc.SendTo(dst, payload)
}

func (r *Router) udpRecvLoop(sendReply func(localSrc *net.UDPAddr, remote *net.UDPAddr, payload []byte) error, sess *udpSession, src *net.UDPAddr, dst *net.UDPAddr) {
	buf := make([]byte, 64*1024)
	for {
		_ = sess.assoc.PC.SetReadDeadline(time.Now().Add(60 * time.Second))
		remote, n, err := sess.assoc.ReadFrom(buf)
		if err != nil {
			// timeout: check idle
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				r.udpMu.Lock()
				last := sess.lastSeen
				r.udpMu.Unlock()
				if time.Since(last) > 2*time.Minute {
					break
				}
				continue
			}
			break
		}
		// For transparent UDP, the reply must appear as if from the original dst.
		// Some servers may reply from a different IP/port; we still use `dst` as local source.
		_ = sendReply(dst, src, buf[:n])
		_ = remote
	}

	r.udpMu.Lock()
	delete(r.udpSess, sess.key)
	r.udpMu.Unlock()
	_ = sess.assoc.Close()
}

