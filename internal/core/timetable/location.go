package timetable

import (
	"regexp"
	"strings"
)

const roomCodePattern = `[A-Za-z]?\d{2,4}[A-Za-z]?(?:[+/、][A-Za-z]?\d{2,4}[A-Za-z]?)*`

var (
	// 成对括号（中英文）里的挂牌 / 备注信息，如「（华南园艺植物协同创新中心）」。
	reLocationParens = regexp.MustCompile(`[（(][^（）()]*[）)]`)
	// 多余空白折叠为单个空格。
	reLocationSpaces = regexp.MustCompile(`\s+`)
	// 分隔符后的「房号 + 中文描述」，如 `219植保实训室`、`A305计算机机房`、`219/220实训室`、`502+505实训室`。
	// 房号要求至少 2 位数字，避免把 `3楼机房` / `2层大厅` 误裁成 `3` / `2`。
	reTrailingRoom = regexp.MustCompile(`^(` + roomCodePattern + `)\s*[\p{Han}].*$`)
	// `6号楼-219-1植保实训室` 这类子房号尾段：最后一段是 1~2 位数字+中文，且前一段是纯房号。
	reSubRoomTail   = regexp.MustCompile(`^(\d{1,2}[A-Za-z]?)\s*[\p{Han}].*$`)
	rePlainSubRoom  = regexp.MustCompile(`^\d{1,2}[A-Za-z]?$`)
	rePlainRoomCode = regexp.MustCompile(`^` + roomCodePattern + `$`)
	// `8号楼-221-显微互动实训室` / `8号楼-502+505-水生生物基础实训室` 这类「房号-中文功能名」尾段。
	reChineseDescTail = regexp.MustCompile(`^[\p{Han}].*$`)
	// 无横杠写法，如 `6号楼219植保实训室`、`实训楼A305计算机机房`。
	reInlineBuildingRoom = regexp.MustCompile(`^(.+?(?:号楼|教学楼|实训楼|综合楼|实验楼|行政楼|楼|栋|幢|座|馆|中心))\s*(` + roomCodePattern + `)\s*[\p{Han}].*$`)
)

// ShortenLocation 把教务系统返回的冗长课室名精简为适合窄屏网格展示的短地点。
//
// 纯函数、无副作用（AGENTS §6）；只用于派生 DTO 的 `shortLocation` 与可选的
// CSV 导出优化，绝不修改快照里的原始 `Location`。
//
// 首版规则（保守裁剪，宁可保留也不误伤）：
//  1. 去掉成对中英文括号里的补充挂牌说明；若去完为空则回退原文本。
//  2. 若存在「楼栋-房号+中文描述」结构（如 `6号楼-219植保实训室`），保留到房号为止。
//  3. 若无分隔符但满足「<楼栋字样><房号><中文描述>」，同样保留到房号为止。
//  4. 其余非标准地点（如 `田径场`、`图书馆报告厅`）返回去括号后的规范文本。
func ShortenLocation(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	s := stripLocationParens(trimmed)
	if s == "" {
		s = trimmed
	}

	if shortened, ok := shortenDelimitedLocation(s); ok {
		return shortened
	}
	if m := reInlineBuildingRoom.FindStringSubmatch(s); len(m) == 3 {
		return m[1] + m[2]
	}
	return s
}

func stripLocationParens(s string) string {
	cur := s
	// 最多迭代几轮以剥离嵌套括号，同时避免异常输入死循环。
	for i := 0; i < 4; i++ {
		next := reLocationParens.ReplaceAllString(cur, "")
		if next == cur {
			break
		}
		cur = next
	}
	cur = reLocationSpaces.ReplaceAllString(strings.TrimSpace(cur), " ")
	return strings.TrimRight(cur, "-—–/ ")
}

func shortenDelimitedLocation(s string) (string, bool) {
	idx := strings.LastIndexAny(s, "-—–")
	if idx <= 0 || idx >= len(s)-1 {
		return "", false
	}

	prefix := strings.TrimRight(s[:idx], " ")
	sep := s[idx : idx+1]
	// UTF-8 下 `—` / `–` 占 3 字节，按字符宽度切片，避免把多字节分隔符截断。
	for _, r := range []string{"—", "–", "-"} {
		if strings.HasPrefix(s[idx:], r) {
			sep = r
			break
		}
	}
	tail := strings.TrimSpace(s[idx+len(sep):])
	if prefix == "" || tail == "" {
		return "", false
	}

	if m := reTrailingRoom.FindStringSubmatch(tail); len(m) == 2 {
		// 避免把 `10号楼` 这类以数字开头的楼栋段当成房号。
		if !isBuildingUnitPrefix(tail, m[1]) {
			return prefix + sep + m[1], true
		}
	}

	// 处理 `6号楼-219-1植保实训室`：最后一段是子房号，且紧邻的前一段本身是纯房号。
	if m := reSubRoomTail.FindStringSubmatch(tail); len(m) == 2 && !isBuildingUnitPrefix(tail, m[1]) {
		if _, prevSegment := splitLastDelimited(prefix); rePlainRoomCode.MatchString(prevSegment) {
			return prefix + sep + m[1], true
		}
	}

	// 处理 `8号楼-221-显微互动实训室`、`8号楼-502+505-水生生物基础实训室`、`6号楼-219-1-植保实训室`：
	// 最后一段是以中文开头的功能描述，且紧邻的前一段是房号（或房号+子房号）。
	if reChineseDescTail.MatchString(tail) && endsWithRoomCode(prefix) {
		return prefix, true
	}

	return "", false
}

// endsWithRoomCode 判断前缀是否以合法房号（含组合房号或「房号-子房号」）结尾。
func endsWithRoomCode(prefix string) bool {
	head, last := splitLastDelimited(prefix)
	if rePlainRoomCode.MatchString(last) {
		return true
	}
	if rePlainSubRoom.MatchString(last) && head != "" {
		_, prev := splitLastDelimited(head)
		return rePlainRoomCode.MatchString(prev)
	}
	return false
}

// splitLastDelimited 按最后一个 `-` / `—` / `–` 拆成前半段与末段；无分隔符时前半段为空。
func splitLastDelimited(s string) (head, last string) {
	idx := strings.LastIndexAny(s, "-—–")
	if idx < 0 {
		return "", strings.TrimSpace(s)
	}
	sepLen := 1
	for _, r := range []string{"—", "–", "-"} {
		if strings.HasPrefix(s[idx:], r) {
			sepLen = len(r)
			break
		}
	}
	return strings.TrimRight(s[:idx], " "), strings.TrimSpace(s[idx+sepLen:])
}

// isBuildingUnitPrefix 防止把 `10号楼` / `2栋` / `3楼` 等量词前缀误判成房号结尾。
func isBuildingUnitPrefix(tail, code string) bool {
	rest := strings.TrimSpace(strings.TrimPrefix(tail, code))
	for _, unit := range []string{"号楼", "栋", "幢", "座", "楼", "层", "期", "区"} {
		if strings.HasPrefix(rest, unit) {
			return true
		}
	}
	return false
}
