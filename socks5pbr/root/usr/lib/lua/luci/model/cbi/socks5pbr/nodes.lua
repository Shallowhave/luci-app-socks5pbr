local m = Map("socks5pbr", "Socks5 PBR - Nodes")

local s = m:section(TypedSection, "node", "Socks5 Nodes")
s.addremove = true
s.anonymous = true
s.template = "cbi/tblsection"

local name = s:option(Value, "name", "Name")
name.rmempty = false

local server = s:option(Value, "server", "Server")
server.rmempty = false

local port = s:option(Value, "port", "Port")
port.datatype = "port"
port.rmempty = false

local username = s:option(Value, "username", "Username")
username.rmempty = true

local password = s:option(Value, "password", "Password")
password.password = true
password.rmempty = true

local remark = s:option(Value, "remark", "Remark")
remark.rmempty = true

return m

