# Testing on OpenWrt (fw4 + nftables)

## Preconditions
- OpenWrt 22.03/23.05/24.x (fw4 + nftables)
- Kernel modules/features for `tproxy`
- A Socks5 server that supports **TCP CONNECT** and **UDP ASSOCIATE**

## 1) Configure nodes + rules
Edit `/etc/config/socks5pbr` (or via LuCI):

- Add at least one `config node` with `name/server/port/(username/password)`
- Add `config rule` mapping a LAN client IP to a node

## 2) Enable and start

```sh
uci set socks5pbr.main.enabled='1'
uci commit socks5pbr
/etc/init.d/socks5pbr enable
/etc/init.d/socks5pbr restart
```

Confirm:

```sh
logread -e socks5pbrd
pidof socks5pbrd
nft list ruleset | grep -n socks5pbr
ip rule | grep 5200
ip route show table 100
```

## 3) TCP functional test
From the configured LAN client IP (e.g. `192.168.1.10`):

```sh
curl -4 ifconfig.me
curl -4 https://ipinfo.io/ip
```

On the router, you can observe:

```sh
logread -f | grep socks5pbrd
```

## 4) UDP functional test (DNS)
From the configured LAN client:

```sh
nslookup openwrt.org 1.1.1.1
```

On the router:

```sh
tcpdump -ni any udp and port 53
```

## 5) UDP functional test (QUIC)
From the configured LAN client:

```sh
curl --http3 -I https://cloudflare-quic.com/
```

If QUIC fails, test TCP fallback:

```sh
curl -I https://cloudflare-quic.com/
```

## 6) Basic stress
From the configured LAN client:

```sh
for i in $(seq 1 200); do (curl -s https://example.com/ >/dev/null &) ; done; wait
```

Observe CPU/mem on router:

```sh
top
logread -e socks5pbrd
```

