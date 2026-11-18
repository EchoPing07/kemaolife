package desktop

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/EchoPing07/kemaolife/internal/app"
	"github.com/EchoPing07/kemaolife/internal/core/porttest"
)

func TestQueryElectricityDoesNotChangeFailures(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		switch r.Form.Get("method") {
		case "getbindroom":
			writeQueryOuter(w, `{"result":"0","existflag":"1","detaillist":[{"businesstype":"0","odd":"50.00"}]}`)
		case "getusedetail":
			writeQueryOuter(w, `{"result":"0","yearmonth":"2026-10","monthuse":"1.00","dayuselist":[]}`)
		default:
			writeQueryOuter(w, `{"result":"0"}`)
		}
	}))
	defer srv.Close()

	clock := porttest.NewFakeClock(time.Date(2026, 11, 5, 12, 0, 0, 0, time.Local))
	settings := DefaultSettings()
	settings.StudentID = "<学号>"
	e := NewEngine(app.Deps{
		Clock:               clock,
		HTTP:                srv.Client(),
		Store:               porttest.NewMemStore(),
		ConfigStore:         porttest.NewMemStore(),
		DefaultCustomerCode: "3126",
		ElectricityEndpoint: srv.URL,
	}, app.CampusDeps{}, nil, func() Settings { return settings }, nil)

	e.mu.Lock()
	e.failures = 3
	e.mu.Unlock()
	if err := e.QueryElectricity(context.Background(), "2026-10"); err != nil {
		t.Fatalf("QueryElectricity success: %v", err)
	}
	e.mu.Lock()
	got := e.failures
	e.mu.Unlock()
	if got != 3 {
		t.Errorf("成功不应重置 failures: got %d, want 3", got)
	}

	if err := e.QueryElectricity(context.Background(), "invalid"); err == nil {
		t.Fatal("非法月份应返回错误")
	}
	e.mu.Lock()
	got = e.failures
	e.mu.Unlock()
	if got != 3 {
		t.Errorf("失败不应推进 failures: got %d, want 3", got)
	}
}

func writeQueryOuter(w http.ResponseWriter, body string) {
	fmt.Fprintf(w, `{"code_":0,"result_":"true","body":%q}`, body)
}
