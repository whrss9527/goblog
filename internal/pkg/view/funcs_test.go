package view

import "testing"

func TestExcerpt(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		limit int
		want  string
	}{
		{"plain", "效能，效就是结果。", 0, "效能，效就是结果。"},
		{"multi-line CJK is concatenated", "Tunnel 可以做什么?\n\n将本地网络的服务暴露到公网。\n自动提供 HTTPS。", 0, "Tunnel 可以做什么?将本地网络的服务暴露到公网。自动提供 HTTPS。"},
		{"latin lines get a space", "hello world\nsecond line", 0, "hello world second line"},
		{"markdown image and link", "图床\n![找不到了.png](https://pic.example.com/a.png)\n看[这个项目](https://github.com/x/y)吧", 0, "图床看这个项目吧"},
		{"headings, quotes, lists, emphasis", "## 标题\n> 引用 **加粗**\n- 列表 `code`", 0, "标题引用 加粗列表 code"},
		{"identifiers keep their underscores", "每张表通过 game_id 字段进行隔离，__强调__", 0, "每张表通过 game_id 字段进行隔离，强调"},
		{"code fences and html dropped", "前文\n```go\nfmt.Println(1)\n```\n<b>后文</b>", 0, "前文后文"},
		{"table rules dropped", "| a | b |\n| --- | --- |\n正文", 0, "| a | b |正文"},
		{"truncated with ellipsis", "一二三四五六七八九十", 5, "一二三四五…"},
		{"truncation trims dangling punctuation", "一二三四，五六七八", 5, "一二三四…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Excerpt(tt.in, tt.limit); got != tt.want {
				t.Errorf("Excerpt() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReadingMinutes(t *testing.T) {
	for wc, want := range map[int]int{0: 1, -5: 1, 54: 1, 400: 1, 401: 2, 2118: 6, 5885: 15} {
		if got := ReadingMinutes(wc); got != want {
			t.Errorf("ReadingMinutes(%d) = %d, want %d", wc, got, want)
		}
	}
}

func TestHumanCount(t *testing.T) {
	for n, want := range map[int]string{0: "0", 9999: "9999", 10000: "1万", 16260: "1.6万", 123456: "12.3万"} {
		if got := HumanCount(n); got != want {
			t.Errorf("HumanCount(%d) = %q, want %q", n, got, want)
		}
	}
}
