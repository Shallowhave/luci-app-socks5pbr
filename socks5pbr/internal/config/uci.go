package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Globals struct {
	Enabled    bool
	TCPPort    int
	UDPPort    int
	UDPEnabled bool
	LogLevel   string
}

type Node struct {
	Name     string
	Server   string
	Port     int
	Username string
	Password string
	Remark   string
}

type Rule struct {
	SrcIP   string
	Node    string
	Enabled bool
}

type Config struct {
	Globals Globals
	Nodes   []Node
	Rules   []Rule
}

func LoadFromFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{
		Globals: Globals{
			Enabled:    true,
			TCPPort:    12345,
			UDPPort:    12346,
			UDPEnabled: true,
			LogLevel:   "info",
		},
	}

	sc := bufio.NewScanner(f)
	var curType string
	var cur map[string]string
	flush := func() error {
		if curType == "" || cur == nil {
			return nil
		}
		switch curType {
		case "globals":
			if v, ok := cur["enabled"]; ok {
				cfg.Globals.Enabled = v == "1" || strings.ToLower(v) == "true"
			}
			if v, ok := cur["tcp_port"]; ok {
				if p, e := strconv.Atoi(v); e == nil {
					cfg.Globals.TCPPort = p
				}
			}
			if v, ok := cur["udp_port"]; ok {
				if p, e := strconv.Atoi(v); e == nil {
					cfg.Globals.UDPPort = p
				}
			}
			if v, ok := cur["udp_enabled"]; ok {
				cfg.Globals.UDPEnabled = v == "1" || strings.ToLower(v) == "true"
			}
			if v, ok := cur["log_level"]; ok && v != "" {
				cfg.Globals.LogLevel = v
			}
		case "node":
			n := Node{
				Name:     cur["name"],
				Server:   cur["server"],
				Username: cur["username"],
				Password: cur["password"],
				Remark:   cur["remark"],
			}
			if v := cur["port"]; v != "" {
				p, e := strconv.Atoi(v)
				if e != nil {
					return fmt.Errorf("bad node port for %q", n.Name)
				}
				n.Port = p
			}
			if n.Name != "" && n.Server != "" && n.Port > 0 {
				cfg.Nodes = append(cfg.Nodes, n)
			}
		case "rule":
			r := Rule{
				SrcIP:   cur["src_ip"],
				Node:    cur["node"],
				Enabled: cur["enabled"] == "" || cur["enabled"] == "1" || strings.ToLower(cur["enabled"]) == "true",
			}
			if r.SrcIP != "" && r.Node != "" {
				cfg.Rules = append(cfg.Rules, r)
			}
		}
		return nil
	}

	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "config ") {
			if err := flush(); err != nil {
				return nil, err
			}
			cur = map[string]string{}
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				curType = parts[1]
			} else {
				curType = ""
			}
			continue
		}
		if strings.HasPrefix(line, "option ") {
			parts := strings.Fields(line)
			if len(parts) < 3 || cur == nil {
				continue
			}
			k := parts[1]
			v := strings.Join(parts[2:], " ")
			v = strings.Trim(v, "'\"")
			cur[k] = v
			continue
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if err := flush(); err != nil {
		return nil, err
	}
	return cfg, nil
}

