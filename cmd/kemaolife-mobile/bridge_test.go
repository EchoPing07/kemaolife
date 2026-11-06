package kemaolife

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// 门面是最窄的对外契约（docs/002 §2.1），这里只验证三件在 Android 上最容易错的事：
// 未初始化时不静默成功、未知 op 返回 2001、信封字段与 CLI 同构。

func TestCallBeforeInitFails(t *testing.T) {
	mu.Lock()
	saved := cur
	cur = nil
	mu.Unlock()
	t.Cleanup(func() {
		mu.Lock()
		cur = saved
		mu.Unlock()
	})

	env := decode(t, Call(`{"op":"ping"}`))
	if env["ok"] != false {
		t.Fatalf("未初始化时不应成功: %v", env)
	}
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 2001 {
		t.Errorf("code = %v, want 2001", errObj["code"])
	}
}

func TestInitPingAndUnknownOp(t *testing.T) {
	cacheDir := t.TempDir()
	raw, err := json.Marshal(map[string]string{
		"cacheDir":     filepath.ToSlash(cacheDir),
		"studentId":    "<学号>",
		"eduPassword":  "<密码>",
		"customerCode": "3126",
	})
	if err != nil {
		t.Fatal(err)
	}
	if env := decode(t, Init(string(raw))); env["ok"] != true {
		t.Fatalf("Init 失败: %v", env)
	}

	env := decode(t, Call(`{"op":"ping"}`))
	if env["ok"] != true {
		t.Fatalf("ping 失败: %v", env)
	}
	if env["ts"] == "" {
		t.Error("信封必须带 ts（docs/002 §2）")
	}
	// 失败信封不得带 data，成功信封不得带 error。
	if _, has := env["error"]; has {
		t.Error("成功信封不应出现 error")
	}

	bad := decode(t, Call(`{"op":"timetable.nope"}`))
	if bad["ok"] != false {
		t.Fatalf("未知 op 必须失败: %v", bad)
	}
	if _, has := bad["data"]; has {
		t.Error("失败信封不应出现 data")
	}
	errObj, _ := bad["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 2001 {
		t.Errorf("未知 op code = %v, want 2001", errObj["code"])
	}
}

func TestInitRejectsMissingCacheDir(t *testing.T) {
	env := decode(t, Init(`{"studentId":"<学号>"}`))
	if env["ok"] != false {
		t.Fatalf("缺少可写目录应失败: %v", env)
	}
}

func TestCallRejectsBrokenJSON(t *testing.T) {
	env := decode(t, Call(`{`))
	if env["ok"] != false {
		t.Fatalf("非法请求应失败: %v", env)
	}
}

// 课表查询在缺凭据时必须回 3001（引导去设置页），而不是 4001 之类的笼统网络错误。
func TestTimetableDateWithoutCredentialsReturns3001(t *testing.T) {
	cacheDir := t.TempDir()
	raw, _ := json.Marshal(map[string]string{
		"cacheDir":     filepath.ToSlash(cacheDir),
		"customerCode": "3126",
	})
	if env := decode(t, Init(string(raw))); env["ok"] != true {
		t.Fatalf("Init 失败: %v", env)
	}

	env := decode(t, Call(`{"op":"timetable.date","date":"2026-11-05"}`))
	if env["ok"] != false {
		t.Fatalf("无凭据且无快照必须失败: %v", env)
	}
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 3001 {
		t.Errorf("code = %v, want 3001", errObj["code"])
	}
	// stale / syncedAt 属于课表成功路径；失败信封不出现。
	if _, has := env["stale"]; has {
		t.Error("失败信封不应出现 stale")
	}
}

func TestDateArgumentValidation(t *testing.T) {
	cacheDir := t.TempDir()
	raw, _ := json.Marshal(map[string]string{
		"cacheDir":    filepath.ToSlash(cacheDir),
		"studentId":   "<学号>",
		"eduPassword": "<密码>",
	})
	if env := decode(t, Init(string(raw))); env["ok"] != true {
		t.Fatalf("Init 失败: %v", env)
	}

	// 日期格式错误属于参数问题（2001），必须在联网之前就返回。
	env := decode(t, Call(`{"op":"timetable.date","date":"2026/11/05"}`))
	errObj, _ := env["error"].(map[string]any)
	if code, _ := errObj["code"].(float64); int(code) != 2001 {
		t.Errorf("code = %v, want 2001", errObj["code"])
	}
}

func decode(t *testing.T, s string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		t.Fatalf("响应不是合法 JSON: %v\n%s", err, s)
	}
	return out
}
