import pytest
import pynvim
import os
import time

@pytest.fixture
def nvim():
    current_dir = os.path.dirname(os.path.abspath(__file__))
    root_dir = os.path.abspath(os.path.join(current_dir, "../.."))
    binary = os.path.join(root_dir, "bin", "nvim-ai-engine")
    init_lua_path = os.path.join(root_dir, "tests", "minimal_init.lua")

    n = pynvim.attach('child', argv=['nvim', '--embed', '--headless', '--clean'])

    n.vars['ai_engine_bin_path'] = binary
    n.vars['project_root'] = root_dir

    if os.path.exists(init_lua_path):
        with open(init_lua_path, 'r') as f:
            lua_config = f.read()
        n.exec_lua(lua_config)
    else:
        pytest.fail(f"Nie znaleziono pliku configu: {init_lua_path}")

    yield n
    n.close()

def test_ai_commit_flow(nvim):
    nvim.current.buffer[:] = ["diff --git a/main.go b/main.go", "+ func New() {}"]

    nvim.command("lua require('bridge').generate_commit()")

    success = False
    for _ in range(50):
        res = nvim.lua.get("_G.test_result")
        if res:
            if res.get('data') and len(res['data']) > 0:
                assert any("feat" in opt.lower() for opt in res['data'])
                success = True
                break
        time.sleep(0.1)

    assert success, "Engine did not return result in time"
