package platform

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RequestLog 是可开关的请求记录器：把每次 HTTP 往返的方法、URL、标头、正文与响应元数据
// 留在**进程内存**里，供 Android 设置页「日志」查看（docs/android.md）。
//
// 为什么落在 platform 而不是 core：记录是传输层能力（它包住 http.RoundTripper），
// core 不该知道「有没有人在看请求」；也不该为了记录多一层参数（AGENTS §3）。
//
// 隐私口径：记录的是**完整原始数据**（标头、正文都是原样），因为排查协议恰恰需要
// 看清真实报文。约束因此落在「记录不许离开本机、不许入仓」这一侧：
//   - 默认关闭，只有用户在设置页显式打开才记录；
//   - 只存内存，不落盘、不进 logcat，进程结束即消失；
//   - 与仓库无关：代码、测试夹具、文档里一律用占位符（P1 / P3）。
type RequestLog struct {
	mu      sync.Mutex
	entries []LogEntry // 固定容量环形缓冲区（按需分配，Clear 时释放）
	head    int        // 最旧条目所在槽位
	count   int        // 当前有效条目数
	enabled atomic.Bool
	max     int
	bodyCap int
	seq     int64
	now     func() time.Time
}

// 默认容量：一次排查够用的窗口，同时限制常驻内存。
const (
	DefaultLogEntries = 100
	defaultBodyCap    = 64 << 10
)

// 截断标记：正文超过上限时追加，让使用者知道「这不是完整的」。
// 这是内存保护，不是脱敏 —— 一次查询的响应可能有几十 MB。
const logTruncatedMark = "\n…（已截断）"

// LogEntry 是一条完整的请求记录；body 与标头都是原始文本。
type LogEntry struct {
	Seq             int64  `json:"seq"`
	At              string `json:"at"`
	Method          string `json:"method"`
	URL             string `json:"url"`
	Status          int    `json:"status"`
	DurationMs      int64  `json:"durationMs"`
	RequestHeaders  string `json:"requestHeaders,omitempty"`
	RequestBody     string `json:"requestBody,omitempty"`
	ResponseHeaders string `json:"responseHeaders,omitempty"`
	ResponseBody    string `json:"responseBody,omitempty"`
	Error           string `json:"error,omitempty"`
}

// LogSummary 是列表项：只给列表需要的字段，正文留在条目里等查看详情时再取。
type LogSummary struct {
	Seq        int64  `json:"seq"`
	At         string `json:"at"`
	Method     string `json:"method"`
	URL        string `json:"url"`
	Status     int    `json:"status"`
	DurationMs int64  `json:"durationMs"`
	Error      string `json:"error,omitempty"`
}

// NewRequestLog 构造记录器；now 由调用方注入（core 之外也不直接读时钟）。
func NewRequestLog(now func() time.Time) *RequestLog {
	if now == nil {
		now = time.Now
	}
	return &RequestLog{max: DefaultLogEntries, bodyCap: defaultBodyCap, now: now}
}

// SetEnabled 开关记录；关闭时清空已记录内容，避免开关关掉后还留着上一次的正文。
func (l *RequestLog) SetEnabled(enabled bool) {
	if l == nil {
		return
	}
	l.enabled.Store(enabled)
	if !enabled {
		l.Clear()
	}
}

// Enabled 报告当前是否在记录。
func (l *RequestLog) Enabled() bool { return l != nil && l.enabled.Load() }

// Clear 清空已记录条目，返回清掉的条数。
func (l *RequestLog) Clear() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.count
	l.entries = nil
	l.head = 0
	l.count = 0
	return n
}

// Count 返回当前条目数。
func (l *RequestLog) Count() int {
	if l == nil {
		return 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.count
}

// Summaries 返回最近 limit 条（新的在前）；limit<=0 表示全部。
func (l *RequestLog) Summaries(limit int) []LogSummary {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	n := l.count
	if limit > 0 && limit < n {
		n = limit
	}
	out := make([]LogSummary, 0, n)
	for i := l.count - 1; i >= 0 && len(out) < n; i-- {
		e := l.entries[(l.head+i)%len(l.entries)]
		out = append(out, LogSummary{
			Seq: e.Seq, At: e.At, Method: e.Method, URL: e.URL,
			Status: e.Status, DurationMs: e.DurationMs, Error: e.Error,
		})
	}
	return out
}

// Find 取一条完整记录。
func (l *RequestLog) Find(seq int64) (LogEntry, bool) {
	if l == nil {
		return LogEntry{}, false
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	for i := l.count - 1; i >= 0; i-- {
		e := l.entries[(l.head+i)%len(l.entries)]
		if e.Seq == seq {
			return e, true
		}
	}
	return LogEntry{}, false
}

// append 写入一条记录：固定容量环形覆写最旧槽位（`max` 条窗口），满窗后不再整片重分配或搬移。
func (l *RequestLog) append(e LogEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()
	capMax := l.max
	if capMax <= 0 {
		capMax = DefaultLogEntries
	}
	if len(l.entries) != capMax {
		next := make([]LogEntry, capMax)
		keep := l.count
		if keep > capMax {
			keep = capMax
		}
		start := l.count - keep
		for i := 0; i < keep; i++ {
			next[i] = l.entries[(l.head+start+i)%len(l.entries)]
		}
		l.entries = next
		l.head = 0
		l.count = keep
	}
	l.seq++
	e.Seq = l.seq
	if l.count < len(l.entries) {
		l.entries[(l.head+l.count)%len(l.entries)] = e
		l.count++
		return
	}
	l.entries[l.head] = e
	l.head = (l.head + 1) % len(l.entries)
}

// clip 只做长度截断（内存保护），不改动内容。
func (l *RequestLog) clip(s string) string {
	if l.bodyCap > 0 && len(s) > l.bodyCap {
		return s[:l.bodyCap] + logTruncatedMark
	}
	return s
}

// clipBytes 在转 string 之前先按字节上限截断，避免为超大报文分配整份临时字符串。
func (l *RequestLog) clipBytes(b []byte) string {
	if l.bodyCap > 0 && len(b) > l.bodyCap {
		var sb strings.Builder
		sb.Grow(l.bodyCap + len(logTruncatedMark))
		sb.Write(b[:l.bodyCap])
		sb.WriteString(logTruncatedMark)
		return sb.String()
	}
	return string(b)
}

// Wrap 把记录能力套在既有 RoundTripper 外层。
//
// 只包装而不替换：强制 IPv4 的 DialContext、超时、连接池都仍由原 Transport 决定
// （api-docs/001-timetable §2、§7），记录不影响真实请求行为。
func (l *RequestLog) Wrap(next http.RoundTripper) http.RoundTripper {
	if l == nil {
		return next
	}
	if next == nil {
		next = http.DefaultTransport
	}
	return &recordingTransport{log: l, next: next}
}

type recordingTransport struct {
	log  *RequestLog
	next http.RoundTripper
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// 关闭时零开销直通：记录不能给正常查询添任何成本。
	if !t.log.Enabled() {
		return t.next.RoundTrip(req)
	}

	start := t.log.now()
	entry := LogEntry{
		At:             start.In(time.Local).Format("2006-01-02T15:04:05.000"),
		Method:         req.Method,
		URL:            req.URL.String(),
		RequestHeaders: formatHeaders(req.Header),
	}

	// 请求正文先读进内存再放回：记录不得改变真正发出去的字节。
	if req.Body != nil {
		body, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			entry.Error = "读取请求正文失败"
			entry.DurationMs = t.log.now().Sub(start).Milliseconds()
			t.log.append(entry)
			return nil, err
		}
		entry.RequestBody = t.log.clipBytes(body)
		req.Body = io.NopCloser(bytes.NewReader(body))
		req.ContentLength = int64(len(body))
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(body)), nil
		}
	}

	resp, err := t.next.RoundTrip(req)
	entry.DurationMs = t.log.now().Sub(start).Milliseconds()
	if err != nil {
		entry.Error = err.Error()
		t.log.append(entry)
		return nil, err
	}

	entry.Status = resp.StatusCode
	entry.ResponseHeaders = formatHeaders(resp.Header)
	// 整段读完再交给调用方：记录不能截断真实响应，否则解析会莫名其妙失败。
	body, readErr := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	resp.ContentLength = int64(len(body))
	if readErr != nil {
		entry.Error = "读取响应正文失败"
	} else {
		entry.ResponseBody = t.log.clipBytes(body)
	}
	t.log.append(entry)
	return resp, nil
}

// CloseIdleConnections 把空闲连接清理透传给底层 Transport，
// 避免装上请求记录器后 http.Client.CloseIdleConnections 退化成空操作。
func (t *recordingTransport) CloseIdleConnections() {
	if ic, ok := t.next.(interface{ CloseIdleConnections() }); ok {
		ic.CloseIdleConnections()
	}
}

// formatHeaders 把标头拍平成可读文本，原样保留值。
//
// 顺序固定（键名排序）：同一份请求每次记录的排版一致，方便对比两次日志的差异。
func formatHeaders(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\n", k, strings.Join(h.Values(k), "; "))
	}
	return strings.TrimRight(b.String(), "\n")
}
