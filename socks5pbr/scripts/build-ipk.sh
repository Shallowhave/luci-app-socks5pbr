#!/bin/sh
# Build socks5pbr daemon (Go) and pack .ipk without OpenWrt SDK.
# Usage: ./build.sh [GOARCH]   default GOARCH=amd64
# Output: out/socks5pbr_*.ipk, out/luci-app-socks5pbr_*.ipk

set -e
PKG_VERSION="${PKG_VERSION:-1.0.0}"
PKG_RELEASE="${PKG_RELEASE:-1}"
GOARCH="${1:-amd64}"
FEED_DIR="$(cd "$(dirname "$0")/../.." && pwd)"
OUT_DIR="${OUT_DIR:-$FEED_DIR/out}"
SOCKS5PBR_DIR="$FEED_DIR/socks5pbr"

mkdir -p "$OUT_DIR"

# Build daemon
echo "Building socks5pbrd for linux/$GOARCH..."
cd "$SOCKS5PBR_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -ldflags "-s -w" -o "$OUT_DIR/socks5pbrd" ./cmd/socks5pbrd

# Pack socks5pbr.ipk (ar: debian-binary + control.tar.gz + data.tar.gz)
echo "Packing socks5pbr.ipk..."
IPK_ROOT="$OUT_DIR/ipk_socks5pbr"
rm -rf "$IPK_ROOT"
mkdir -p "$IPK_ROOT/usr/sbin" "$IPK_ROOT/etc/config" "$IPK_ROOT/etc/init.d" "$IPK_ROOT/usr/lib/socks5pbr"
cp "$OUT_DIR/socks5pbrd" "$IPK_ROOT/usr/sbin/"
cp "$SOCKS5PBR_DIR/root/etc/config/socks5pbr" "$IPK_ROOT/etc/config/"
cp "$SOCKS5PBR_DIR/root/etc/init.d/socks5pbr" "$IPK_ROOT/etc/init.d/"
cp "$SOCKS5PBR_DIR/root/usr/lib/socks5pbr/gen_nft.sh" "$IPK_ROOT/usr/lib/socks5pbr/"
chmod 755 "$IPK_ROOT/usr/sbin/socks5pbrd" "$IPK_ROOT/etc/init.d/socks5pbr" "$IPK_ROOT/usr/lib/socks5pbr/gen_nft.sh"

mkdir -p "$IPK_ROOT/CONTROL"
cat > "$IPK_ROOT/CONTROL/control" << EOF
Package: socks5pbr
Version: $PKG_VERSION-$PKG_RELEASE
Architecture: x86_64
Maintainer: Socks5 PBR
Section: net
Description: Socks5 PBR daemon (TPROXY + policy by LAN IP). Uses nftables TPROXY; supports TCP and UDP.
Depends: kmod-nft-tproxy
EOF

( cd "$IPK_ROOT" && tar --owner=0 --group=0 -czf "$OUT_DIR/control.socks5pbr.tar.gz" CONTROL )
( cd "$IPK_ROOT" && tar --owner=0 --group=0 -czf "$OUT_DIR/data.socks5pbr.tar.gz" usr etc )
echo "2.0" > "$OUT_DIR/debian-binary"
( cd "$OUT_DIR" && ar cr "socks5pbr_${PKG_VERSION}-${PKG_RELEASE}_x86_64.ipk" debian-binary control.socks5pbr.tar.gz data.socks5pbr.tar.gz )
rm -f "$OUT_DIR/control.socks5pbr.tar.gz" "$OUT_DIR/data.socks5pbr.tar.gz"
rm -rf "$IPK_ROOT"

# Pack luci-app-socks5pbr.ipk (no binary, just Lua/JSON)
echo "Packing luci-app-socks5pbr.ipk..."
IPK_ROOT="$OUT_DIR/ipk_luci"
rm -rf "$IPK_ROOT"
mkdir -p "$IPK_ROOT/usr/lib/lua/luci/controller" \
         "$IPK_ROOT/usr/lib/lua/luci/model/cbi/socks5pbr" \
         "$IPK_ROOT/usr/lib/lua/luci/view/socks5pbr" \
         "$IPK_ROOT/usr/share/luci/menu.d" \
         "$IPK_ROOT/usr/share/luci/acl.d" \
         "$IPK_ROOT/CONTROL"

cp "$SOCKS5PBR_DIR/root/usr/lib/lua/luci/controller/socks5pbr.lua" "$IPK_ROOT/usr/lib/lua/luci/controller/"
cp "$SOCKS5PBR_DIR/root/usr/lib/lua/luci/model/cbi/socks5pbr/"*.lua "$IPK_ROOT/usr/lib/lua/luci/model/cbi/socks5pbr/"
cp "$SOCKS5PBR_DIR/root/usr/lib/lua/luci/view/socks5pbr/"*.htm "$IPK_ROOT/usr/lib/lua/luci/view/socks5pbr/"
cp "$SOCKS5PBR_DIR/root/usr/share/luci/menu.d/luci-app-socks5pbr.json" "$IPK_ROOT/usr/share/luci/menu.d/"
cp "$SOCKS5PBR_DIR/root/usr/share/luci/acl.d/luci-app-socks5pbr.json" "$IPK_ROOT/usr/share/luci/acl.d/"

cat > "$IPK_ROOT/CONTROL/control" << EOF
Package: luci-app-socks5pbr
Version: $PKG_VERSION-$PKG_RELEASE
Architecture: all
Maintainer: Socks5 PBR
Section: luci
Description: LuCI for Socks5 PBR: nodes, rules, batch import, status.
Depends: socks5pbr, luci-base
EOF

( cd "$IPK_ROOT" && tar --owner=0 --group=0 -czf "$OUT_DIR/control.luci.tar.gz" CONTROL )
( cd "$IPK_ROOT" && tar --owner=0 --group=0 -czf "$OUT_DIR/data.luci.tar.gz" usr )
( cd "$OUT_DIR" && ar cr "luci-app-socks5pbr_${PKG_VERSION}-${PKG_RELEASE}_all.ipk" debian-binary control.luci.tar.gz data.luci.tar.gz )
rm -f "$OUT_DIR/control.luci.tar.gz" "$OUT_DIR/data.luci.tar.gz"
rm -rf "$IPK_ROOT"

echo "Done. IPKs in $OUT_DIR:"
ls -la "$OUT_DIR"/*.ipk
