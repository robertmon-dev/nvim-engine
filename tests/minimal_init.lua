local root = vim.g.project_root
local tests_dir = root .. "/tests"

vim.opt.rtp:append(root)
vim.opt.rtp:append(tests_dir)

package.path = package.path .. ";" .. tests_dir .. "/lua/?.lua"

_G.Tele = {
	log = function(msg, _, sys)
		print("[" .. (sys or "AI") .. "] " .. msg)
	end,
	error = function(msg, sys)
		print("ERROR [" .. (sys or "AI") .. "] " .. msg)
	end,
	warn = function(msg, sys)
		print("WARN [" .. (sys or "AI") .. "] " .. msg)
	end,
}

_G.test_result = nil
_G.on_ai_result = function(res)
	_G.test_result = res
end

vim.api.nvim_out_write("LUA: Config loaded successfully\n")
