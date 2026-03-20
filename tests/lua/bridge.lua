local M = {}
local job_id = 0

local bin_path = vim.g.ai_engine_bin_path or "bin/nvim-ai-engine"

function M.ensure_engine()
	if job_id > 0 and vim.fn.jobwait({ job_id }, 0)[1] == -1 then
		return job_id
	end

	job_id = vim.fn.jobstart({ bin_path }, {
		rpc = true,
		on_stderr = function(_, data)
			local msg = table.concat(data, "\n")
			if msg ~= "" then
				_G.Tele.error("Engine Stderr: " .. msg, "Go-Engine")
			end
		end,
		on_exit = function()
			job_id = 0
		end,
	})
	return job_id
end

function M.generate_commit()
	local id = M.ensure_engine()
	if id <= 0 then
		_G.Tele.error("Could not start AI engine", "Bridge")
		return
	end

	local lines = vim.api.nvim_buf_get_lines(0, 0, -1, false)
	local diff = table.concat(lines, "\n")

	vim.rpcnotify(id, "submit_task", {
		id = "test-task-123",
		action = "commit",
		payload = diff,
	})
end

return M
