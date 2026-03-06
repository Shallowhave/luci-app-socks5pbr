local m = SimpleForm("socks5pbr_status", "Socks5 PBR - Status")
m.reset = false
m.submit = false

local st = m:section(SimpleSection)

local function exec(cmd)
	local p = io.popen(cmd .. " 2>/dev/null")
	if not p then return "" end
	local out = p:read("*a") or ""
	p:close()
	return out
end

local running = (exec("pidof socks5pbrd") or ""):gsub("%s+", "") ~= ""
st:append(Template("cbi/nullsection"))

local s = m:section(SimpleSection)
s.template = "socks5pbr/status"
s.running = running

return m

