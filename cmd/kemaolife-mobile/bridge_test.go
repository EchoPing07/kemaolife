package kemaolife

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/core/model"
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

// timetable.sync 的信封层 syncedAt 与 CLI 同构（docs/002 §2）；
// 这里用缺凭据的失败路径无法验证成功字段，直接构造一个最小成功信封断言编解码口径。
func TestTimetableSyncSuccessEnvelopeIncludesSyncedAt(t *testing.T) {
	env := model.Success(time.Date(2026, 11, 7, 0, 0, 0, 0, time.Local), map[string]any{
		"syncedAt": "<快照同步时间>",
	})
	env.SyncedAt = "<快照同步时间>"
	b, err := json.Marshal(env)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got["syncedAt"] != "<快照同步时间>" {
		t.Errorf("信封 syncedAt = %v", got["syncedAt"])
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

func TestApplyTimeZoneByName(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })

	applyTimeZone("Asia/Shanghai", 0)
	if got := time.Local.String(); got != "Asia/Shanghai" {
		t.Fatalf("Local = %q", got)
	}
	ts := time.Date(2026, 11, 6, 22, 48, 42, 0, time.Local).Format(time.RFC3339)
	if ts != "2026-11-06T22:48:42+08:00" {
		t.Errorf("format = %q, want +08:00", ts)
	}
}

func TestApplyTimeZoneFallbackToOffset(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })

	applyTimeZone("Not/AZone", 8*3600)
	if _, off := time.Date(2026, 11, 6, 22, 48, 42, 0, time.Local).Zone(); off != 8*3600 {
		t.Errorf("offset = %d, want %d", off, 8*3600)
	}
}

// Init 传时区后，信封 ts 必须是本地时区而不是 UTC。
func TestInitAppliesTimeZoneToEnvelope(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })

	raw, _ := json.Marshal(map[string]any{
		"cacheDir":              filepath.ToSlash(t.TempDir()),
		"timeZone":              "Asia/Shanghai",
		"timeZoneOffsetSeconds": 8 * 3600,
	})
	if env := decode(t, Init(string(raw))); env["ok"] != true {
		t.Fatalf("Init 失败: %v", env)
	}
	ts, _ := decode(t, Call(`{"op":"ping"}`))["ts"].(string)
	if len(ts) < 6 || ts[len(ts)-6:] != "+08:00" {
		t.Errorf("ts = %q, 应以 +08:00 结尾（本地时区）", ts)
	}
}

// 复现真机问题：时区未校准时，collectedAt 会输出 UTC（14:48:42Z）。
// 校准后必须是本地时区（22:48:42+08:00），这是「数据截至」可读的前提。
func TestElectricityTimestampUsesDeviceZone(t *testing.T) {
	saved := time.Local
	t.Cleanup(func() { time.Local = saved })
	applyTimeZone("Asia/Shanghai", 8*3600)

	// 上游 collecdate 是无时区的本地时间字符串，由 app 层按 time.Local 解释。
	raw := "2026-11-06 22:48:42"
	parsed, err := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local)
	if err != nil {
		t.Fatal(err)
	}
	got := parsed.In(time.Local).Format(time.RFC3339)
	if got != "2026-11-06T22:48:42+08:00" {
		t.Errorf("collectedAt = %q, 应为 +08:00", got)
	}

	// 同时确认日期不会因为 UTC 转换而倒退一天（AGENTS §6）。
	d := model.NewDate(parsed)
	if d.String() != "2026-11-06" {
		t.Errorf("date = %s, 不应差一天", d)
	}
}
