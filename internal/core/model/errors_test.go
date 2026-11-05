package model

import "testing"

// 退出码与 error.code 必须同域映射（AGENTS §4.4）。
func TestExitCodeMapping(t *testing.T) {
	cases := []struct {
		code ErrorCode
		want int
	}{
		{CodeOK, 0},
		{CodeBusinessFailed, 1},
		{CodeNotInSemester, 1},
		{CodeRoomNotBound, 1},
		{CodeInvalidArgument, 2},
		{CodeCredentialMissing, 3},
		{CodeSessionExpired, 3},
		{CodeSnapshotMissing, 3},
		{CodeNetworkDown, 4},
		{CodeMalformedResponse, 4},
		{CodeUpstreamRejected, 4},
	}
	for _, c := range cases {
		if got := ExitCode(NewError(c.code, "x")); got != c.want {
			t.Errorf("code %d exit = %d, want %d", c.code, got, c.want)
		}
	}
}

func TestExitCodeNilIsZero(t *testing.T) {
	if got := ExitCode(nil); got != 0 {
		t.Errorf("nil exit = %d, want 0", got)
	}
}

func TestCodeOfUnknownIsNetworkDown(t *testing.T) {
	if got := CodeOf(errDummy{}); got != CodeNetworkDown {
		t.Errorf("unknown = %d, want %d", got, CodeNetworkDown)
	}
}

type errDummy struct{}

func (errDummy) Error() string { return "boom" }

// 成功信封不得出现 error，也不得序列化出 code:0。
func TestEnvelopeFailureHasCode(t *testing.T) {
	env := Failure(fixedTime, NewError(CodeSessionExpired, "会话失效，需重新同步"))
	if env.OK {
		t.Fatal("失败信封 ok 应为 false")
	}
	if env.Data != nil {
		t.Error("失败信封不应带 data")
	}
	if env.Error == nil || env.Error.Code != int(CodeSessionExpired) {
		t.Fatalf("error = %+v", env.Error)
	}
}
