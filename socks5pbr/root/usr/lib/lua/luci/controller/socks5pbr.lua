module("luci.controller.socks5pbr", package.seeall)

function index()
	entry({"admin", "network", "socks5pbr", "import"}, call("import"), _("Batch Import"), 50).leaf = true
	if not nixio.fs.access("/etc/config/socks5pbr") then
		return
	end
end

-- The menu is defined via menu.d JSON. This controller only provides actions.

function import()
	local http = require "luci.http"
	local uci = require "luci.model.uci".cursor()

	if http.getenv("REQUEST_METHOD") == "POST" then
		local body = http.formvalue("payload") or ""
		local imported, err = do_import_oneline(uci, body)
		if err then
			http.status(400, "Bad Request")
			http.prepare_content("application/json")
			http.write_json({ ok = false, error = err })
			return
		end

		uci:commit("socks5pbr")
		http.prepare_content("application/json")
		http.write_json({ ok = true, imported = imported })
		return
	end

	-- Simple HTML form: one line per node = ip/port/username/password/remark
	http.prepare_content("text/html")
	http.write([[
<!doctype html>
<html>
<head><meta charset="utf-8"><title>Socks5 PBR - Batch Import</title></head>
<body style="font-family: sans-serif; max-width: 900px; margin: 24px auto;">
  <h2>Socks5 PBR - Batch Import</h2>
  <form method="post">
    <p><label>One line per node: <code>ip/端口/用户名/密码/备注</code></label></p>
    <p><textarea name="payload" rows="18" style="width:100%;" placeholder="1.2.3.4/1080/user/pass/HK&#10;5.6.7.8/1080///US"></textarea></p>
    <p><button type="submit">Import</button></p>
  </form>
  <p>Empty fields allowed. Lines starting with # and blank lines are ignored.</p>
</body></html>
]])
end

-- One line per node: ip/port/username/password/remark (slash-separated)
function do_import_oneline(uci, body)
	if not body or #body == 0 then
		return 0, "empty payload"
	end

	local n = 0
	for line in body:gmatch("[^\r\n]+") do
		line = line:gsub("^%s+", ""):gsub("%s+$", "")
		if #line == 0 or line:sub(1, 1) == "#" then
			goto continue
		end
		local parts = {}
		for part in (line .. "/"):gmatch("(.-)/") do
			table.insert(parts, part)
		end
		-- need at least ip and port (parts[1], parts[2])
		local server = parts[1] and parts[1]:gsub("^%s+", ""):gsub("%s+$", "") or ""
		local port_s = parts[2] and parts[2]:gsub("^%s+", ""):gsub("%s+$", "") or "0"
		local port = tonumber(port_s) or 0
		local username = (parts[3] or ""):gsub("^%s+", ""):gsub("%s+$", "")
		local password = (parts[4] or ""):gsub("^%s+", ""):gsub("%s+$", "")
		local remark = (parts[5] or ""):gsub("^%s+", ""):gsub("%s+$", "")

		local err = validate_node("node", server, port)
		if err then return n, ("line: %s"):format(err) end
		local name = ("node_%d"):format(n + 1)
		upsert_node(uci, name, server, port, username, password, remark)
		n = n + 1
		::continue::
	end
	return n, nil
end

function validate_node(_name, server, port)
	if not server or #server == 0 then return "missing ip" end
	if not port or port < 1 or port > 65535 then return "bad port (1-65535)" end
	return nil
end

function upsert_node(uci, name, server, port, username, password, remark)
	local sid = nil
	uci:foreach("socks5pbr", "node", function(s)
		if s.name == name then sid = s[".name"] end
	end)
	if not sid then
		sid = uci:add("socks5pbr", "node")
	end
	uci:set("socks5pbr", sid, "name", name)
	uci:set("socks5pbr", sid, "server", server)
	uci:set("socks5pbr", sid, "port", tostring(port))
	uci:set("socks5pbr", sid, "username", username)
	uci:set("socks5pbr", sid, "password", password)
	uci:set("socks5pbr", sid, "remark", remark)
end

