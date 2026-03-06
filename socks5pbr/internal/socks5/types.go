package socks5

import (
	"errors"
	"fmt"
	"net"
)

type Auth struct {
	Username string
	Password string
}

type Server struct {
	Host string
	Port int
	Auth *Auth
}

func (s Server) Addr() string {
	return net.JoinHostPort(s.Host, fmt.Sprintf("%d", s.Port))
}

var (
	ErrUnsupportedAuthMethod = errors.New("socks5: unsupported auth method")
	ErrAuthFailed            = errors.New("socks5: username/password auth failed")
	ErrBadReply              = errors.New("socks5: bad reply")
)

