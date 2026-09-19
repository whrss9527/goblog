package md2html

import (
	"strings"
	"testing"
)

func TestMd2HtmlForFeed(t *testing.T) {
	out := Md2Html([]byte("1. 一\n\n   ```go\n\tif x {\n\t\treturn\n\t}\n   ```\n2. 二 `<T>` :fa-check:\n\n![图](https://img.example/a.png)\n"))

	if strings.Count(out, "<ol") != 1 {
		t.Errorf("the code block must not split the numbered list: %s", out)
	}
	if !strings.Contains(out, "\tif x {\n\t\treturn\n\t}") {
		t.Errorf("code must keep its tab indentation: %s", out)
	}
	if !strings.Contains(out, "<b>&lt;T&gt;</b>") || strings.Contains(out, "<t>") {
		t.Errorf("inline code becomes bold text and stays escaped: %s", out)
	}
	if !strings.Contains(out, "✅") || strings.Contains(out, ":fa-check:") {
		t.Errorf("shortcodes are rendered like on the site: %s", out)
	}
	if strings.Contains(out, "<html") || strings.Contains(out, "<body") {
		t.Errorf("feed content is a fragment, not a document: %s", out)
	}
	if !strings.Contains(out, `style="max-width: 500px;`) || strings.Contains(out, "/>") {
		t.Errorf("images get inline styles, void tags are written HTML style: %s", out)
	}
	if li := out[strings.Index(out, "<li"):]; strings.HasPrefix(li, `<li style="max-width: 1300px; display: block`) {
		t.Errorf("list items must not be display:block (readers drop the numbers): %s", out)
	}
}
