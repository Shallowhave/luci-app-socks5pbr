# OpenWrt feed package: luci-app-socks5pbr
# Depends on socks5pbr. Installs LuCI UI (menu, CBI, controller, view).
# Expects feed layout: feed/socks5pbr/ and feed/luci-app-socks5pbr/ (this dir).

include $(TOPDIR)/rules.mk

PKG_NAME:=luci-app-socks5pbr
PKG_VERSION:=1.0.0
PKG_RELEASE:=1
PKG_LICENSE:=MIT

# Same feed: .../socks5pbr and .../luci-app-socks5pbr
LUCI_ROOT_USR:=$(dir $(CURDIR))socks5pbr/root/usr

include $(INCLUDE_DIR)/package.mk

define Package/luci-app-socks5pbr
  SECTION:=luci
  CATEGORY:=LuCI
  SUBMENU:=Network
  TITLE:=LuCI for Socks5 PBR
  DEPENDS:=+socks5pbr +luci-base
  PKGARCH:=all
endef

define Package/luci-app-socks5pbr/description
 LuCI interface for Socks5 PBR: nodes, rules, batch import (ip/port/user/pass/remark per line), status.
endef

define Package/luci-app-socks5pbr/install
	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/controller
	$(INSTALL_DATA) $(LUCI_ROOT_USR)/lib/lua/luci/controller/socks5pbr.lua $(1)/usr/lib/lua/luci/controller/
	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/model/cbi/socks5pbr
	$(INSTALL_DATA) $(LUCI_ROOT_USR)/lib/lua/luci/model/cbi/socks5pbr/*.lua $(1)/usr/lib/lua/luci/model/cbi/socks5pbr/
	$(INSTALL_DIR) $(1)/usr/lib/lua/luci/view/socks5pbr
	$(INSTALL_DATA) $(LUCI_ROOT_USR)/lib/lua/luci/view/socks5pbr/*.htm $(1)/usr/lib/lua/luci/view/socks5pbr/
	$(INSTALL_DIR) $(1)/usr/share/luci/menu.d
	$(INSTALL_DATA) $(LUCI_ROOT_USR)/share/luci/menu.d/luci-app-socks5pbr.json $(1)/usr/share/luci/menu.d/
	$(INSTALL_DIR) $(1)/usr/share/luci/acl.d
	$(INSTALL_DATA) $(LUCI_ROOT_USR)/share/luci/acl.d/luci-app-socks5pbr.json $(1)/usr/share/luci/acl.d/
endef

$(eval $(call BuildPackage,luci-app-socks5pbr))
