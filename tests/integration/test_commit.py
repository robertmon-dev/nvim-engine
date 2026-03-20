import time

def test_ai_commit_flow(nvim):
    nvim.current.buffer[:] = ["diff --git a/main.go b/main.go", "+ func New() {}"]

    nvim.command("lua require('bridge').generate_commit()")

    success = False
    for _ in range(50):
        res = nvim.exec_lua("return _G.test_result")
        if res:
            if res.get('data') and len(res['data']) > 0:
                assert any("feat" in opt.lower() for opt in res['data'])
                success = True
                break
        time.sleep(0.1)

    assert success, "Engine did not return result in time"
