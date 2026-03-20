import pytest
import pynvim
import os

@pytest.fixture
def nvim():
    current_dir = os.path.dirname(os.path.abspath(__file__))
    root_dir = os.path.abspath(os.path.join(current_dir, "../.."))
    tests_dir = os.path.join(root_dir, "tests")
    binary = os.path.join(root_dir, "bin", "nvim-ai-engine")
    init_lua_path = os.path.join(root_dir, "tests", "minimal_init.lua")

    n = pynvim.attach('child', argv=['nvim', '--embed', '--headless', '--clean'])

    n.command(f"set rtp+={tests_dir}")

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
