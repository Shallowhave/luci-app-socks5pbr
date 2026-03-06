package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"socks5pbr/internal/config"
	"socks5pbr/internal/router"
	"socks5pbr/internal/tproxy"
)

func main() {
	var (
		cfgPath = flag.String("config", "/etc/config/socks5pbr", "UCI config path")
		tcpAddr = flag.String("tcp", "", "TCP listen addr, e.g. :12345 (override config)")
		udpAddr = flag.String("udp", "", "UDP listen addr, e.g. :12346 (override config)")
		logLvl  = flag.String("log", "", "log level (override config)")
	)
	flag.Parse()

	logger := log.New(os.Stdout, "[socks5pbrd] ", log.LstdFlags|log.Lshortfile)
	logger.Println("starting")

	fileCfg, err := config.LoadFromFile(*cfgPath)
	if err != nil {
		logger.Fatalf("load config: %v", err)
	}

	if *logLvl != "" {
		fileCfg.Globals.LogLevel = *logLvl
	}

	r := router.New(logger)
	if err := r.LoadConfig(fileCfg); err != nil {
		logger.Fatalf("apply config: %v", err)
	}

	tcpListen := *tcpAddr
	if tcpListen == "" {
		tcpListen = net.JoinHostPort("", itoa(fileCfg.Globals.TCPPort))
	}
	udpListen := *udpAddr
	if udpListen == "" {
		udpListen = net.JoinHostPort("", itoa(fileCfg.Globals.UDPPort))
	}

	cfg := tproxy.Config{
		TCPListenAddr: tcpListen,
		UDPListenAddr: udpListen,
	}

	srv, err := tproxy.NewServer(cfg, logger, r)
	if err != nil {
		logger.Fatalf("create tproxy server: %v", err)
	}

	if err := srv.Start(); err != nil {
		logger.Fatalf("start tproxy server: %v", err)
	}

	// Wait for SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	logger.Println("shutting down")
	_ = srv.Close()
	logger.Println("exited")
}

func itoa(n int) string {
	return fmt.Sprintf("%d", n)
}


