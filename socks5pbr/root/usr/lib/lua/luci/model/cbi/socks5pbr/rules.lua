local uci = require "luci.model.uci".cursor()
local m = Map("socks5pbr", "Socks5 PBR - Rules")

local node_list = {}
uci:foreach("socks5pbr", "node", function(s)
	if s.name then
		node_list[#node_list+1] = s.name
	end
end)

local s = m:section(TypedSection, "rule", "LAN IP Policy Rules")
s.addremove = true
s.anonymous = true
s.template = "cbi/tblsection"

local enabled = s:option(Flag, "enabled", "Enable")
enabled.default = enabled.enabled

local src_ip = s:option(Value, "src_ip", "Source IP")
src_ip.datatype = "ip4addr"
src_ip.rmempty = false

local node = s:option(ListValue, "node", "Node")
node.rmempty = false
for _, n in ipairs(node_list) do
	node:value(n, n)
end

return m

