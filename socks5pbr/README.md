# socks5pbr (OpenWrt Socks5 Policy Routing)

- **socks5pbrd**: Go daemon (TPROXY TCP/UDP → Socks5), installs to `/usr/sbin/socks5pbrd`.
- **luci-app-socks5pbr**: LuCI UI (nodes, rules, batch import, status).
- fw4/nftables integration: init script + `gen_nft.sh` → `/etc/nftables.d/socks5pbr.nft`.

## Batch import format (LuCI)

One line per node: **`ip/端口/用户名/密码/备注`** (slash-separated). Empty fields allowed; lines starting with `#` and blank lines are ignored.

Example:
```
1.2.3.4/1080/user/pass/HK节点
5.6.7.8/1080///US
```

## Build (developer)

```bash
cd socks5pbr
go test ./...
go build -o socks5pbrd ./cmd/socks5pbrd
```

## OpenWrt feed (compile and install)

1. Add this repo as a feed (e.g. in `feeds.conf.default`):
   ```
   src-git socks5pbr https://github.com/yourname/socks5pbr.git
   ```
   Or copy the repo into your SDK tree, e.g. `openwrt/feeds/socks5pbr/` with contents: `socks5pbr/` and `luci-app-socks5pbr/`.

2. Install feed and compile:
   ```bash
   ./scripts/feeds update socks5pbr
   ./scripts/feeds install -p socks5pbr socks5pbr luci-app-socks5pbr
   make package/feeds/socks5pbr/socks5pbr/compile
   make package/feeds/socks5pbr/luci-app-socks5pbr/compile
   ```
   Or from menuconfig: Network → socks5pbr; LuCI → Applications → luci-app-socks5pbr.

3. Install layout:
   - Binary: `/usr/sbin/socks5pbrd`
   - Config: `/etc/config/socks5pbr`
   - Init: `/etc/init.d/socks5pbr`
   - Script: `/usr/lib/socks5pbr/gen_nft.sh`
   - LuCI: under `/usr/lib/lua/luci/...` and `/usr/share/luci/...`

## Sync to GitHub and CI build

1. **Create a new repository** on GitHub (e.g. `socks5pbr` or `openclash`).

2. **Add remote and push** (from your local repo root, where `socks5pbr/` and `luci-app-socks5pbr/` live):
   ```bash
   git init
   git add .
   git commit -m "Socks5 PBR feed for OpenWrt"
   git branch -M main
   git remote add origin https://github.com/YOUR_USER/YOUR_REPO.git
   git push -u origin main
   ```

3. **GitHub Actions** (`.github/workflows/build-openwrt.yml`):
   - Trigger: push/PR to `main`, `master`, or `openwrt-24.10`.
   - Uses [ImmortalWrt openwrt-24.10](https://github.com/immortalwrt/immortalwrt/tree/openwrt-24.10) as build tree.
   - Adds this repo as feed, builds `socks5pbr` and `luci-app-socks5pbr` for **x86_64**.
   - Uploads `.ipk` artifacts (download from the Actions run "Artifacts").

4. **Use the built IPKs** on any ImmortalWrt/OpenWrt x86_64 device: download the artifact, extract, then `opkg install socks5pbr_*.ipk luci-app-socks5pbr_*.ipk` (or copy to device and install).

