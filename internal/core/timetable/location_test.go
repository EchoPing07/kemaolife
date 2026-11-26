package timetable

import "testing"

func TestShortenLocation(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "实训室加中文括号挂牌名",
			raw:  "6号楼-219植保实训室（华南园艺植物协同创新中心）",
			want: "6号楼-219",
		},
		{
			name: "英文括号挂牌名",
			raw:  "6号楼-219植保实训室(华南园艺植物协同创新中心)",
			want: "6号楼-219",
		},
		{
			name: "已精简的标准楼栋房号",
			raw:  "6号楼-219",
			want: "6号楼-219",
		},
		{
			name: "带字母前缀房号",
			raw:  "实训楼-A305计算机机房",
			want: "实训楼-A305",
		},
		{
			name: "带字母后缀房号",
			raw:  "3号教学楼-101A多媒体教室",
			want: "3号教学楼-101A",
		},
		{
			name: "双房号斜杠拼接",
			raw:  "6号楼-219/220植物组织培养室",
			want: "6号楼-219/220",
		},
		{
			name: "子房号连字符拼接",
			raw:  "6号楼-219-1植保实训室",
			want: "6号楼-219-1",
		},
		{
			name: "房号与实训室名之间带横杠",
			raw:  "8号楼-221-显微互动实训室",
			want: "8号楼-221",
		},
		{
			name: "加号双房号加横杠实训室名",
			raw:  "8号楼-502+505-水生生物基础实训室",
			want: "8号楼-502+505",
		},
		{
			name: "加号双房号直接连中文描述",
			raw:  "8号楼-507+508动物营养与饲料分析实训室",
			want: "8号楼-507+508",
		},
		{
			name: "子房号加横杠实训室名",
			raw:  "6号楼-219-1-植保实训室",
			want: "6号楼-219-1",
		},
		{
			name: "简称前缀加多媒体室",
			raw:  "综-212多媒体室",
			want: "综-212",
		},
		{
			name: "课室后缀与斜杠功能名",
			raw:  "8号楼-224中兽医实训室/模型标本室",
			want: "8号楼-224",
		},
		{
			name: "单字楼栋简称不改动",
			raw:  "教103",
			want: "教103",
		},
		{
			name: "实训简称房号不改动",
			raw:  "实212",
			want: "实212",
		},
		{
			name: "纯中文段前非房号不误裁",
			raw:  "白云校区-实训楼-多媒体大厅",
			want: "白云校区-实训楼-多媒体大厅",
		},
		{
			name: "带校区前缀与多段横杠",
			raw:  "白云校区-10号楼-219植保实训室（协同中心）",
			want: "白云校区-10号楼-219",
		},
		{
			name: "无横杠楼栋房号",
			raw:  "6号楼219植保实训室（西）",
			want: "6号楼219",
		},
		{
			name: "楼层描述不误裁为数字",
			raw:  "实训楼-3楼机房",
			want: "实训楼-3楼机房",
		},
		{
			name: "无房号场馆只去括号",
			raw:  "图书馆报告厅（一楼）",
			want: "图书馆报告厅",
		},
		{
			name: "纯中文场地原样保留",
			raw:  "田径场",
			want: "田径场",
		},
		{
			name: "纯括号不清空",
			raw:  "（待定）",
			want: "（待定）",
		},
		{
			name: "空串与纯空白",
			raw:  "   ",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ShortenLocation(tc.raw); got != tc.want {
				t.Errorf("ShortenLocation(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
