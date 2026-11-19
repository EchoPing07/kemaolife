package desktop

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/EchoPing07/kemaolife/internal/core"
	"github.com/EchoPing07/kemaolife/internal/version"
)

// latestReleaseURL 是官方仓库最新正式发布的公开查询端点（非 draft、非 prerelease）。
// 与三个功能域无关：不带凭据、Cookie、学号或设备标识，也不读本机快照。
const latestReleaseURL = "https://api.github.com/repos/EchoPing07/kemaolife/releases/latest"

// UpdateInfo 是一次更新检查的结果。Latest 是按数值重排后的三段版本号，
// 不把上游 tag 字符串原样带进 UI（P2）。
type UpdateInfo struct {
	Latest    string
	HasUpdate bool
}

// CheckUpdate 取最新发布的 tag_name 并与本机版本比较。
//
// 错误文案一律本地固定措辞（最多带上游状态码）：网络错误可能含代理地址，
// 上游正文可能含任意内容，都不得原样进入错误信息（P2）。
func CheckUpdate(ctx context.Context, doer core.HTTPDoer, current string) (UpdateInfo, error) {
	cur, ok := version.Parse(current)
	if !ok {
		return UpdateInfo{}, errors.New("检查更新失败：本机版本号无法解析")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseURL, nil)
	if err != nil {
		return UpdateInfo{}, errors.New("检查更新失败：请求构造异常")
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "kemaolife-desktop")
	resp, err := doer.Do(req)
	if err != nil {
		return UpdateInfo{}, errors.New("检查更新失败：网络请求未成功")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return UpdateInfo{}, fmt.Errorf("检查更新失败：上游状态码 %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return UpdateInfo{}, errors.New("检查更新失败：响应读取中断")
	}
	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UpdateInfo{}, errors.New("检查更新失败：响应格式异常")
	}
	latest, ok := version.Parse(payload.TagName)
	if !ok {
		return UpdateInfo{}, errors.New("检查更新失败：版本标签格式异常")
	}
	return UpdateInfo{Latest: latest.String(), HasUpdate: version.Compare(latest, cur) > 0}, nil
}
