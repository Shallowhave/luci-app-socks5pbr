# Socks5 PBR - UCI schema and batch import spec

## UCI schema: /etc/config/socks5pbr

### config globals 'main'
| Option      | Type   | Default  | Description                    |
|------------|--------|----------|--------------------------------|
| enabled    | bool   | 1        | Master switch                  |
| tcp_port   | port   | 12345    | Local TPROXY TCP port          |
| udp_port   | port   | 12346    | Local TPROXY UDP port          |
| log_level  | string | info     | debug/info/warn/error          |
| udp_enabled| bool   | 1        | Enable UDP transparent proxy   |

### config node
| Option   | Type   | Required | Description        |
|----------|--------|----------|--------------------|
| name     | string | yes      | Unique node id     |
| server   | string | yes      | Socks5 server IP   |
| port     | port   | yes      | Socks5 port        |
| username | string | no       | Auth username      |
| password | string | no       | Auth password      |
| remark   | string | no       | Display remark     |

### config rule
| Option   | Type   | Required | Description              |
|----------|--------|----------|--------------------------|
| src_ip   | string | yes      | LAN client IP            |
| node     | string | yes      | Node name (must exist)   |
| enabled  | bool   | yes      | 1=active, 0=disabled     |

## Batch import format (default)

**One line per node:** `ip/端口/用户名/密码/备注`

- Separator: `/` (slash). Empty fields allowed (omit or leave between slashes).
- Order: server_ip, port, username, password, remark.
- Encoding: UTF-8. Blank lines and lines starting with `#` are ignored.
- Node name: auto-generated as `node_<line_index>` (e.g. `node_1`, `node_2`). Duplicate server:port in one paste overwrite by line order.
- Validation: server non-empty; port 1-65535.

Example:
```
1.2.3.4/1080/user/pass/HK节点
5.6.7.8/1080///US
10.0.0.1/1080//
```
(Third line: no auth, no remark.)

## LuCI page structure
- **Nodes**: `/admin/network/socks5pbr/nodes` - list nodes, add/edit/delete, batch import button.
- **Rules**: `/admin/network/socks5pbr/rules` - list rules (src_ip -> node), add/edit/delete, enable/disable.
- **Settings**: `/admin/network/socks5pbr/settings` - globals (ports, log_level, udp_enabled).
- **Status**: `/admin/network/socks5pbr` - overview + daemon status, node count, rule count, recent errors (ubus or log).
