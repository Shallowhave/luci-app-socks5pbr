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

pack_ipk() {
	# pack_ipk <pkgname> <version> <release> <arch> <depends> <section> <description> <data_root_dir>
	PKGNAME="$1"
	VER="$2"
	REL="$3"
	ARCH="$4"
	DEPENDS="$5"
	SECTION="$6"
	DESC="$7"
	DATA_ROOT="$8"

	WORK="$OUT_DIR/.work_${PKGNAME}"
	rm -rf "$WORK"
	mkdir -p "$WORK/CONTROL"

	cat > "$WORK/CONTROL/control" << EOF
Package: ${PKGNAME}
Version: ${VER}-${REL}
Architecture: ${ARCH}
Maintainer: Socks5 PBR
Section: ${SECTION}
Description: ${DESC}
Depends: ${DEPENDS}
EOF

	# Standard ipk member names MUST be: debian-binary, control.tar.gz, data.tar.gz
	printf '2.0\n' > "$WORK/debian-binary"
	( cd "$WORK/CONTROL" && tar --owner=0 --group=0 -czf "$WORK/control.tar.gz" control )
	( cd "$DATA_ROOT" && tar --owner=0 --group=0 -czf "$WORK/data.tar.gz" . )

	OUT_IPK="$OUT_DIR/${PKGNAME}_${VER}-${REL}_${ARCH}.ipk"
	( cd "$WORK" && ar cr "$OUT_IPK" debian-binary control.tar.gz data.tar.gz )

	rm -rf "$WORK"
}

# Build daemon
echo "Building socks5pbrd for linux/$GOARCH..."
cd "$SOCKS5PBR_DIR"
CGO_ENABLED=0 GOOS=linux GOARCH="$GOARCH" go build -ldflags "-s -w" -o "$OUT_DIR/socks5pbrd" ./cmd/socks5pbrd

# Pack socks5pbr.ipk
echo "Packing socks5pbr.ipk..."
DATA_ROOT="$OUT_DIR/.data_socks5pbr"
rm -rf "$DATA_ROOT"
mkdir -p "$DATA_ROOT/usr/sbin" "$DATA_ROOT/etc/config" "$DATA_ROOT/etc/init.d" "$DATA_ROOT/usr/lib/socks5pbr"
cp "$OUT_DIR/socks5pbrd" "$DATA_ROOT/usr/sbin/"
cp "$SOCKS5PBR_DIR/root/etc/config/socks5pbr" "$DATA_ROOT/etc/config/"
cp "$SOCKS5PBR_DIR/root/etc/init.d/socks5pbr" "$DATA_ROOT/etc/init.d/"
cp "$SOCKS5PBR_DIR/root/usr/lib/socks5pbr/gen_nft.sh" "$DATA_ROOT/usr/lib/socks5pbr/"
chmod 755 "$DATA_ROOT/usr/sbin/socks5pbrd" "$DATA_ROOT/etc/init.d/socks5pbr" "$DATA_ROOT/usr/lib/socks5pbr/gen_nft.sh"

pack_ipk \
	"socks5pbr" \
	"$PKG_VERSION" \
	"$PKG_RELEASE" \
	"x86_64" \
	"kmod-nft-tproxy" \
	"net" \
	"Socks5 PBR daemon (TPROXY + policy by LAN IP). Uses nftables TPROXY; supports TCP and UDP." \
	"$DATA_ROOT"

rm -rf "$DATA_ROOT"

# Pack luci-app-socks5pbr.ipk (no binary, just Lua/JSON)
echo "Packing luci-app-socks5pbr.ipk..."
DATA_ROOT="$OUT_DIR/.data_luci"
rm -rf "$DATA_ROOT"
mkdir -p "$DATA_ROOT/usr/lib/lua/luci/controller" \
         "$DATA_ROOT/usr/lib/lua/luci/model/cbi/socks5pbr" \
         "$DATA_ROOT/usr/lib/lua/luci/view/socks5pbr" \
         "$DATA_ROOT/usr/share/luci/menu.d" \
         "$DATA_ROOT/usr/share/luci/acl.d"

cp "$SOCKS5PBR_DIR/root/usr/lib/lua/luci/controller/socks5pbr.lua" "$DATA_ROOT/usr/lib/lua/luci/controller/"
cp "$SOCKS5PBR_DIR/root/usr/lib/lua/luci/model/cbi/socks5pbr/"*.lua "$DATA_ROOT/usr/lib/lua/luci/model/cbi/socks5pbr/"
cp "$SOCKS5PBR_DIR/root/usr/lib/lua/luci/view/socks5pbr/"*.htm "$DATA_ROOT/usr/lib/lua/luci/view/socks5pbr/"
cp "$SOCKS5PBR_DIR/root/usr/share/luci/menu.d/luci-app-socks5pbr.json" "$DATA_ROOT/usr/share/luci/menu.d/"
cp "$SOCKS5PBR_DIR/root/usr/share/luci/acl.d/luci-app-socks5pbr.json" "$DATA_ROOT/usr/share/luci/acl.d/"

pack_ipk \
	"luci-app-socks5pbr" \
	"$PKG_VERSION" \
	"$PKG_RELEASE" \
	"all" \
	"socks5pbr, luci-base" \
	"luci" \
	"LuCI for Socks5 PBR: nodes, rules, batch import, status." \
	"$DATA_ROOT"

rm -rf "$DATA_ROOT"

echo "Done. IPKs in $OUT_DIR:"
ls -la "$OUT_DIR"/*.ipk
