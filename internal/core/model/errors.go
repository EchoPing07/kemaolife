package model

import (
	"errors"
	"fmt"
)

// ErrorCode 是 docs/002 §4 定义的错误码域。
type ErrorCode int

const (
	// CodeOK 仅作内部枚举；成功信封不出现 error，故不会序列化出 code:0。
	CodeOK ErrorCode = 0

	CodeBusinessFailed    ErrorCode = 1001
	CodeNotInSemester     ErrorCode = 1002
	CodeInvalidArgument   ErrorCode = 2001
	CodeCredentialMissing ErrorCode = 3001
	CodeSessionExpired    ErrorCode = 3002
	CodeSnapshotMissing   ErrorCode = 3003
	CodeNetworkDown       ErrorCode = 4001
	CodeMalformedResponse ErrorCode = 4002
	CodeUpstreamRejected  ErrorCode = 4003
	CodeRoomNotBound      ErrorCode = 5001
)

// Error 是带错误码的领域错误。消息里不得出现学号、姓名、房间号、密码、Cookie。
type Error struct {
	Code    ErrorCode
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Err)
}

func (e *Error) Unwrap() error { return e.Err }

// NewError 构造带错误码的错误。
func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

// WrapError 保留错误码并附加上游错误。
func WrapError(code ErrorCode, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

// CodeOf 取错误码；非领域错误按 4001（网络 / 未知）归类，避免静默成功。
func CodeOf(err error) ErrorCode {
	if err == nil {
		return CodeOK
	}
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return CodeNetworkDown
}

// ExitCode 把错误码映射为 CLI 退出码（docs/002 §1.2）。
func ExitCode(err error) int {
	switch CodeOf(err) {
	case CodeOK:
		return 0
	case CodeInvalidArgument:
		return 2
	case CodeCredentialMissing, CodeSessionExpired, CodeSnapshotMissing:
		return 3
	case CodeNetworkDown, CodeMalformedResponse, CodeUpstreamRejected:
		return 4
	default:
		// 1001 / 1002 / 5001 等业务失败
		return 1
	}
}

// MessageOf 取领域错误的展示文案；非领域错误（含裸 err、context.Canceled、
// 反射 / JSON 内部错误）一律回落按码映射的固定中性文案，绝不把 err.Error()
// 原样透出去 —— 上游原始文案可能夹带学号 / 房间号 / 内网地址（P2 / docs/002 §2.1 规则 8）。
func MessageOf(err error) string {
	code := CodeOf(err)
	if code == CodeOK {
		return ""
	}
	var e *Error
	if errors.As(err, &e) && e.Message != "" {
		return e.Message
	}
	return CodeMessage(code)
}

// CodeMessage 是错误码 → 固定中性文案的兜底表；不带任何上游原文。
//
// 门面的 `*.probe` 也直接用它：测试按钮失败时不展示领域错误自带的动态文案
// （例如非法月份里夹着用户输入），只展示按码映射的固定中性原因（P2 / docs/002 §2.1 规则 8）。
func CodeMessage(code ErrorCode) string {
	switch code {
	case CodeBusinessFailed:
		return "业务处理失败，请稍后重试"
	case CodeNotInSemester:
		return "不在本学期范围"
	case CodeInvalidArgument:
		return "参数错误，请检查输入"
	case CodeCredentialMissing, CodeSessionExpired, CodeSnapshotMissing:
		return "凭据或本地数据不可用，请重新配置或同步"
	case CodeNetworkDown:
		return "网络不可达，请检查连接后重试"
	case CodeMalformedResponse:
		return "响应格式异常，请稍后重试"
	case CodeUpstreamRejected:
		return "请求被拒绝，请稍后重试"
	case CodeRoomNotBound:
		return "未绑定房间，请先在官方页面完成绑定"
	default:
		return "操作失败，请稍后重试"
	}
}
