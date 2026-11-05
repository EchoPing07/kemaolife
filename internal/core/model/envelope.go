package model

import "time"

// Envelope 是 docs/002 §2 定义的统一 JSON 信封，CLI --json 与 Android 门面共用。
//
// Ok:false 时 Data 缺省（omitempty），Ok:true 时 Error 缺省；
// Stale / SyncedAt 只在课表数据来自快照时出现。
type Envelope struct {
	OK       bool      `json:"ok"`
	Data     any       `json:"data,omitempty"`
	Error    *APIError `json:"error,omitempty"`
	TS       string    `json:"ts"`
	Stale    bool      `json:"stale,omitempty"`
	SyncedAt string    `json:"syncedAt,omitempty"`
}

// APIError 是信封里的错误对象；Code 取 1001–5001，不含 CLI 退出码。
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Success 构造成功信封。ts 由调用方经 Clock 取得，保持 core 不读时钟。
func Success(now time.Time, data any) Envelope {
	return Envelope{OK: true, Data: data, TS: now.In(time.Local).Format(time.RFC3339)}
}

// SuccessStale 构造「沿用旧快照」的成功信封。
func SuccessStale(now time.Time, data any, syncedAt time.Time) Envelope {
	return Envelope{
		OK:       true,
		Data:     data,
		TS:       now.In(time.Local).Format(time.RFC3339),
		Stale:    true,
		SyncedAt: syncedAt.In(time.Local).Format(time.RFC3339),
	}
}

// Failure 构造失败信封。
func Failure(now time.Time, err error) Envelope {
	return Envelope{
		OK: false,
		Error: &APIError{
			Code:    int(CodeOf(err)),
			Message: MessageOf(err),
		},
		TS: now.In(time.Local).Format(time.RFC3339),
	}
}
