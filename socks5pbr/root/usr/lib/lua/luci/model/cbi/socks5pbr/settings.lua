local m = Map("socks5pbr", "Socks5 PBR - Settings")

local s = m:section(NamedSection, "main", "globals", "Globals")
s.addremove = false

local enabled = s:option(Flag, "enabled", "Enable")
enabled.default = enabled.enabled

local tcp_port = s:option(Value, "tcp_port", "Local TCP TPROXY port")
tcp_port.datatype = "port"
tcp_port.default = "12345"

local udp_enabled = s:option(Flag, "udp_enabled", "Enable UDP transparent proxy")
udp_enabled.default = udp_enabled.enabled

local udp_port = s:option(Value, "udp_port", "Local UDP TPROXY port")
udp_port.datatype = "port"
udp_port.default = "12346"
udp_port:depends("udp_enabled", "1")

local log_level = s:option(ListValue, "log_level", "Log level")
log_level:value("debug", "debug")
log_level:value("info", "info")
log_level:value("warn", "warn")
log_level:value("error", "error")
log_level.default = "info"

return m

