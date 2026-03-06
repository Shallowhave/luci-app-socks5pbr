# OpenWrt feed package: socks5pbr (daemon + init + config + nft script)
# Build: from OpenWrt SDK or build root, add this dir as feed and run:
#   make package/feeds/<feed>/socks5pbr/compile
# Install: binary -> /usr/sbin/socks5pbrd

include $(TOPDIR)/rules.mk

PKG_NAME:=socks5pbr
PKG_VERSION:=1.0.0
PKG_RELEASE:=1
PKG_LICENSE:=MIT

PKG_BUILD_DIR:=$(BUILD_DIR)/$(PKG_NAME)-$(PKG_VERSION)
PKG_SOURCE:=$(PKG_NAME)-$(PKG_VERSION).tar.gz
PKG_SOURCE_URL:=file:///
PKG_SOURCE_VERSION:=
PKG_MIRROR_HASH:=skip

include $(INCLUDE_DIR)/package.mk

# Map OpenWrt ARCH to Go GOARCH
GO_ARCH:=$(ARCH)
ifeq ($(ARCH),mipsel)
  GO_ARCH:=mipsle
endif
ifeq ($(ARCH),i386)
  GO_ARCH:=386
endif
ifeq ($(ARCH),x86_64)
  GO_ARCH:=amd64
endif
ifeq ($(ARCH),aarch64)
  GO_ARCH:=arm64
endif

define Package/socks5pbr
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Socks5 PBR daemon (TPROXY + policy by LAN IP)
  DEPENDS:=+kmod-nft-tproxy
  URL:=https://github.com/openwrt/socks5pbr
endef

define Package/socks5pbr/description
 Transparent Socks5 proxy with policy routing by LAN client IP.
 Uses nftables TPROXY; supports TCP and UDP (UDP ASSOCIATE).
endef

define Build/Prepare
	mkdir -p $(PKG_BUILD_DIR)
	$(CP) ./cmd $(PKG_BUILD_DIR)/
	$(CP) ./internal $(PKG_BUILD_DIR)/
	$(CP) ./go.mod $(PKG_BUILD_DIR)/
	$(CP) ./go.sum $(PKG_BUILD_DIR)/ 2>/dev/null || true
	cd $(PKG_BUILD_DIR) && go mod download
endef

define Build/Compile
	cd $(PKG_BUILD_DIR) && \
	CGO_ENABLED=0 GOOS=linux GOARCH=$(GO_ARCH) go build -ldflags "-s -w" -o socks5pbrd ./cmd/socks5pbrd
endef

define Package/socks5pbr/install
	$(INSTALL_DIR) $(1)/usr/sbin
	$(INSTALL_BIN) $(PKG_BUILD_DIR)/socks5pbrd $(1)/usr/sbin/socks5pbrd
	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./root/etc/config/socks5pbr $(1)/etc/config/socks5pbr
	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./root/etc/init.d/socks5pbr $(1)/etc/init.d/socks5pbr
	$(INSTALL_DIR) $(1)/usr/lib/socks5pbr
	$(INSTALL_BIN) ./root/usr/lib/socks5pbr/gen_nft.sh $(1)/usr/lib/socks5pbr/gen_nft.sh
endef

$(eval $(call BuildPackage,socks5pbr))
