import pytest
import pynvim
import os

@pytest.fixture
def nvim():
    binary = os.path.abspath("../../bin/nvim-ai-engine")

    n = pynvim.attach('child', argv=['nvim', '--embed', '--headless', '-u', 'minimal_init.lua'])
    n.vars['ai_engine_bin_path'] = binary

    yield n
    n.close()

def test_ai_commit_flow(nvim):
    nvim.current.buffer[:] = ["diff --git a/main.go b/main.go", "+ func New() {}"]
    nvim.command("lua require('bridge').generate_commit()")

    success = False
    for _ in range(50):
        res = nvim.lua.get("_G.test_result")
        if res:
            assert "feat" in res[0].lower()
            success = True
            break
        import time; time.sleep(0.1)

    assert success, "Engine did not return result in time"
