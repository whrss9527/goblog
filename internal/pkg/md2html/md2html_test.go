package md2html

import (
	"strings"
	"testing"
)

func TestFeedHTML(t *testing.T) {
	out := FeedHTML("1. 一\n\n   ```go\n\tif x {\n\t\treturn\n\t}\n   ```\n2. 二 `<T>` :fa-check:\n\n![图](https://img.example/a.png)\n"+
		"[上一篇](/posts/older) ![本站](/covers/a.png) [外站](https://example.com/x) [协议相对](//cdn.example/y)\n", "https://blog.example.com/")

	if strings.Count(out, "<ol") != 1 {
		t.Errorf("the code block must not split the numbered list: %s", out)
	}
	if !strings.Contains(out, "\tif x {\n\t\treturn\n\t}") {
		t.Errorf("code must keep its tab indentation: %s", out)
	}
	if !strings.Contains(out, "<code>&lt;T&gt;</code>") || strings.Contains(out, "<t>") {
		t.Errorf("inline code stays code and stays escaped: %s", out)
	}
	if !strings.Contains(out, "✅") || strings.Contains(out, ":fa-check:") {
		t.Errorf("shortcodes are rendered like on the site: %s", out)
	}
	if strings.Contains(out, "<html") || strings.Contains(out, "<body") {
		t.Errorf("feed content is a fragment, not a document: %s", out)
	}
	if strings.Contains(out, "style=") || strings.Contains(out, "/>") {
		t.Errorf("no inline styles, void tags are written HTML style: %s", out)
	}
	for _, want := range []string{`href="https://blog.example.com/posts/older"`, `src="https://blog.example.com/covers/a.png"`,
		`href="https://example.com/x"`, `href="//cdn.example/y"`, `src="https://img.example/a.png"`} {
		if !strings.Contains(out, want) {
			t.Errorf("feed content does not contain %s: %s", want, out)
		}
	}
}
