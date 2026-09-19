package md2html

import (
	"math/rand"
	"strings"
	"testing"
)

func render(t *testing.T, md string) string {
	t.Helper()
	out, err := RenderArticle(md)
	if err != nil {
		t.Fatalf("RenderArticle: %v", err)
	}
	return out
}

func TestRenderArticleBasics(t *testing.T) {
	out := render(t, "## 安装 rclone\n\n第一行\n第二行\n\n它包含:\n1. 一\n2. 二\n\n| a | b |\n| --- | --- |\n| 1 | 2 |\n\n~~删除~~ https://example.com/x\n")

	for _, want := range []string{
		`<h2 id="安装-rclone">安装 rclone</h2>`,
		"第一行<br/>", // hard line breaks like marked's breaks:true
		"<ol>", "<table>", "<del>删除</del>",
		`<a href="https://example.com/x" target="_blank" rel="noopener noreferrer">`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
}

func TestRenderArticleCode(t *testing.T) {
	out := render(t, "```go\nfunc main() { fmt.Println(\"<hi>\") }\n```\n\n行内 `a_b*c` 代码 :fa-check:\n")

	if !strings.Contains(out, `<pre class="prettyprint linenums"><code class="language-go">`) {
		t.Errorf("fenced code must carry prettify classes and the language: %s", out)
	}
	if !strings.Contains(out, "&lt;hi&gt;") || strings.Contains(out, "<hi>") {
		t.Errorf("code must stay escaped: %s", out)
	}
	if !strings.Contains(out, "<code>a_b*c</code>") {
		t.Errorf("inline code must be left untouched: %s", out)
	}
	if !strings.Contains(out, "✅") || strings.Contains(out, ":fa-check:") {
		t.Errorf("FontAwesome shortcode must become an emoji: %s", out)
	}
}

func TestRenderArticleCleansInvisibleCharacters(t *testing.T) {
	out := render(t, "从网页复制来的\u00a0文字\r\n\r\n```\r\nthe\u00a0timer\r\n\tindented\r\n```\r\n")
	if strings.Contains(out, "\u00a0") || strings.Contains(out, "\r") {
		t.Errorf("non-breaking spaces and carriage returns must be normalised like marked does: %q", out)
	}
	if !strings.Contains(out, "the timer\n\tindented\n</code>") {
		t.Errorf("tabs inside code must survive: %q", out)
	}
}

func TestRenderArticleShortcodesInsideCodeStay(t *testing.T) {
	out := render(t, "`:fa-check:`\n\n```\n:fa-rocket:\n```\n\n:fa-unknown-icon: 文本 :fa-hand-o-right:")
	if strings.Count(out, ":fa-check:") != 1 || strings.Count(out, ":fa-rocket:") != 1 {
		t.Errorf("shortcodes inside code must be preserved: %s", out)
	}
	if strings.Contains(out, ":fa-unknown-icon:") || !strings.Contains(out, "👉") {
		t.Errorf("unknown shortcodes are dropped, known ones replaced: %s", out)
	}
}

func TestRenderArticleTwemojiShortcodes(t *testing.T) {
	out := render(t, "充实:tw-1f606: 加油:tw-270a: 国旗:tw-1f1e8-1f1f3: 坏的:tw-zz: 时间 10:18:03\n")
	for _, want := range []string{"充实😆", "加油✊\ufe0f", "国旗🇨🇳", "坏的:tw-zz:", "10:18:03"} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q: %s", want, out)
		}
	}
}

func TestRenderArticleTypography(t *testing.T) {
	out := render(t, "头像 -> settings -> 左侧，范围 10.0.0.0 - 10.255.255.255，1/2 和 (c) 保持原样。\n\n"+
		"他说 \"你好\" -- 然后 'ok'... it's `\"code\" -- ...` https://example.com/a--b\n")
	for _, want := range []string{
		"头像 -&gt; settings -&gt; 左侧", "10.0.0.0 - 10.255.255.255", "1/2 和 (c) 保持原样",
		"他说 “你好” — 然后 ‘ok’… it’s", "<code>&#34;code&#34; -- ...</code>", ">https://example.com/a--b</a>",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q: %s", want, out)
		}
	}
}

func TestRenderArticleSanitizes(t *testing.T) {
	out := render(t, "<script>alert(1)</script>\n\n<iframe src=\"https://evil.example\"></iframe>\n\n<style>body{display:none}</style>\n\n"+
		"<img src=\"a.png\" onerror=\"alert(1)\">\n\n<a href=\"java\tscript:alert(1)\" onclick=\"x()\">x</a>\n\n<div class=\"note\">保留 <b>内联 HTML</b></div>\n\n[md](javascript:alert(2))")

	for _, banned := range []string{"<script", "<iframe", "<style", "onerror", "onclick", "javascript:", "alert("} {
		if strings.Contains(strings.ToLower(out), banned) {
			t.Errorf("output must not contain %q:\n%s", banned, out)
		}
	}
	if !strings.Contains(out, `<div class="note">`) || !strings.Contains(out, "<b>内联 HTML</b>") {
		t.Errorf("harmless inline HTML must survive: %s", out)
	}
}

func TestRenderArticleImagesAndTOCMarker(t *testing.T) {
	out := render(t, "[TOC]\n\n![one](https://img.example/1.png)\n\n![two](https://img.example/2.png)\n")

	if strings.Contains(out, "[TOC]") {
		t.Errorf("[TOC] marker must be removed: %s", out)
	}
	if strings.Count(out, `loading="lazy"`) != 1 || strings.Count(out, `decoding="async"`) != 2 {
		t.Errorf("all images decode async, all but the first load lazily: %s", out)
	}
}

func TestRenderArticleTaskLists(t *testing.T) {
	out := render(t, "- [ ] todo\n- [x] done\n- plain [ ] item\n")
	if strings.Count(out, `type="checkbox"`) != 2 || strings.Count(out, "checked") != 1 {
		t.Errorf("two checkboxes, one checked expected: %s", out)
	}
	if strings.Contains(out, "[x] done") || !strings.Contains(out, "plain [ ] item") {
		t.Errorf("markers are consumed only at the start of an item: %s", out)
	}
}

func TestHeadingIDsAreUniqueAndStable(t *testing.T) {
	out := render(t, "# 总结\n\n## 总结\n\n### Q&A: what?\n\n#### ！！\n")
	for _, want := range []string{`id="总结"`, `id="总结-2"`, `id="qa-what"`, `id="section-4"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %s in %s", want, out)
		}
	}
}

func TestSlugify(t *testing.T) {
	tests := map[string]string{
		"安装 rclone":         "安装-rclone",
		"通过 CLI 参数控制执行行为":   "通过-cli-参数控制执行行为",
		"结构化统计信息 + 可追溯日志":   "结构化统计信息-可追溯日志",
		"  Hello,  World! ": "hello-world",
		"！！":                "",
	}
	for in, want := range tests {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNeedsClientRenderer(t *testing.T) {
	tests := map[string]bool{
		"plain **markdown**":             false,
		"```go\ncode\n```":               false,
		"```flow\nst=>start: Start\n```": true,
		"  ```seq\nA->B: hi\n```":        true,
		"公式 $$E=mc^2$$ 在这里":              true,
		"价格是 $5 和 $$ 不成对":                false,
		"```math\nx\n```":                true,
		"text mentioning flow and seq.":  false,
	}
	for in, want := range tests {
		if got := NeedsClientRenderer(in); got != want {
			t.Errorf("NeedsClientRenderer(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestNormalizeMarkdown(t *testing.T) {
	tests := []struct {
		name, in, want string
	}{
		{"lazy continuation after a paragraph", "para\n\tcontinued\n    more", "para\ncontinued\nmore"},
		{"indented list-looking line after a paragraph stays code (as in marked)", "steps:\n\t1. one\n\t2. two", "steps:\n\t1. one\n\t2. two"},
		{"indented fence after a paragraph stays code (as in marked)", "steps:\n    ```\n    x\n    ```", "steps:\n    ```\n    x\n    ```"},
		{"real indented code after a blank line stays", "para\n\n    code", "para\n\n    code"},
		{"indented line after a heading is still code", "# h\n    code", "# h\n    code"},
		{"heading with leading spaces", "  ## title", "## title"},
		{"lazy continuation inside a quote", "> quoted\n> \t\t*- author*", "> quoted\n> *- author*"},
		{"quoted paragraph continued without the marker", "> quoted\n      lazy\n\nnext", "> quoted\nlazy\n\nnext"},
		{"lazy list lines get the canonical indent", "1. item\n    ![img](x.png)\n  more", "1. item\n    ![img](x.png)\n    more"},
		{"item continuation after a blank line is re-indented", "1. one\n   \n   *note*\n\n2. two", "1. one\n\n    *note*\n\n2. two"},
		{"unindented text after a blank line ends the list", "- a\n\ntext\n   indented", "- a\n\ntext\n   indented"},
		{"nested items move to four spaces per level", "1. a\n\n   - sub\n     - deep\n   - sub2\n2. b", "1. a\n\n    - sub\n        - deep\n    - sub2\n2. b"},
		{"continuation goes to the level it is indented for", "- a\n  - b\n\n    of b\n\n  of a\n- c", "- a\n    - b\n\n        of b\n\n    of a\n- c"},
		{"indented list is moved to the margin", "  * a\n  * b", "* a\n* b"},
		{"bullets left of the list belong to the item (as in marked)", " 1. 一\n- x\n- y\n\n 2. 二\n\n- z", "1. 一\n    - x\n    - y\n\n2. 二\n\n- z"},
		{"a rule ends the list", "- a\n***\ntext", "- a\n\n***\ntext"},
		{"indented code inside a list item", "1. run\n\n        go build\n          -v\n2. done", "1. run\n\n         go build\n           -v\n2. done"},
		{"over-indented lazy line in an item is code (as in marked)", "- run\n      go build\n- done", "- run\n\n        go build\n- done"},
		{"quote inside a list item", "- a\n\n  > quoted\n- b", "- a\n\n    > quoted\n- b"},
		{"list inside a quote", "> - a\n>\n>   more\n> - b", "> - a\n>\n>     more\n> - b"},
		{"quote after a list item ends the list", "- a\n> quoted", "- a\n\n> quoted"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, blocks := normalizeMarkdown(tt.in)
			if got != tt.want || len(blocks) != 0 {
				t.Errorf("normalizeMarkdown(%q)\n got %q (%d blocks)\nwant %q", tt.in, got, len(blocks), tt.want)
			}
		})
	}
}

func TestNormalizeMarkdownLiftsFencedCode(t *testing.T) {
	in := "1. one\n   ```go\n   x := 1\n\n     y\n   ```  \n2. two\n\n~~~\n```\n~~~\ntext\n> ```sh\n> ls\n> ```\n\n```not closed\n"
	got, blocks := normalizeMarkdown(in)

	want := "1. one\n\n    GOBLOGFENCEDBLOCK0END\n\n2. two\n\nGOBLOGFENCEDBLOCK1END\n\ntext\n>\n> GOBLOGFENCEDBLOCK2END\n>\n\n```not closed\n"
	if got != want {
		t.Errorf("normalizeMarkdown\n got %q\nwant %q", got, want)
	}
	wantBlocks := []fencedBlock{{"go", "x := 1\n\n  y\n"}, {"", "```\n"}, {"sh", "ls\n"}}
	if len(blocks) != len(wantBlocks) {
		t.Fatalf("got %d blocks, want %d: %#v", len(blocks), len(wantBlocks), blocks)
	}
	for i, b := range blocks {
		if b != wantBlocks[i] {
			t.Errorf("block %d = %#v, want %#v", i, b, wantBlocks[i])
		}
	}
}

func TestRenderArticleKeepsListNumbering(t *testing.T) {
	out := render(t, "1. **一**：说明\n   \n   *补充段落*\n\n2. **二**：说明\n\n   ```go\n   x := 1\n\n   y := 2\n   ```\n\n3. **三**\n   - 子项\n     - 孙项\n4. 四\n")
	if strings.Count(out, "<ol>") != 1 {
		t.Errorf("continuation paragraphs and fences must not split the list (numbering would restart): %s", out)
	}
	if !strings.Contains(out, "<em>补充段落</em>") || !strings.Contains(out, "x := 1\n\ny := 2\n</code></pre>") {
		t.Errorf("continuation content must survive, code keeps its blank lines: %s", out)
	}
	if strings.Count(out, "<ul>") != 2 || strings.Contains(out, "<br/>") {
		t.Errorf("two nested levels expected, without <br> in front of the sub-lists: %s", out)
	}
}

func TestRenderArticleLooselyIndentedLists(t *testing.T) {
	out := strings.ReplaceAll(render(t, " 1. 全局 recover\n- 入口包装\n- 输出日志\n\n 2. 信号监听\n\n- 支持 SIGINT\n- 检测 context\n"), "\n", "")
	if strings.Count(out, "<ol>") != 1 || strings.Count(out, "<ul>") != 2 {
		t.Fatalf("one numbered list with a nested and a following bullet list expected: %s", out)
	}
	if !strings.Contains(out, "全局 recover</p><ul><li>入口包装</li>") || !strings.Contains(out, "</ol><ul><li>支持 SIGINT</li>") {
		t.Errorf("first bullets nest under item 1, the last ones follow the numbered list: %s", out)
	}
}

func TestRenderArticleFencedCodePlacement(t *testing.T) {
	out := render(t, "前文\n```js\nlet a = '<b>';\n```\n后文\n\n> 引用\n> ```\n> quoted code\n> ```\n\n<div>\n\n```\nin html\n```\n\n</div>\n\n文字 GOBLOGFENCEDBLOCK99END 保留\n")
	for _, want := range []string{
		"<p>前文</p>", "<p>后文</p>",
		`<pre class="prettyprint linenums"><code class="language-js">let a = &#39;&lt;b&gt;&#39;;` + "\n</code></pre>",
		"<blockquote>", "quoted code\n</code></pre>",
		"in html\n</code></pre>",
		"GOBLOGFENCEDBLOCK99END",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output does not contain %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "<p><pre") || strings.Contains(out, "<p></p>") {
		t.Errorf("code blocks must not end up inside paragraphs: %s", out)
	}
	if i, j := strings.Index(out, "<blockquote>"), strings.Index(out, "quoted code"); i < 0 || j < i || j > strings.Index(out, "</blockquote>") {
		t.Errorf("quoted code must stay inside the quote: %s", out)
	}
}

func TestRenderArticleMarkedCompatibility(t *testing.T) {
	out := render(t, "```go\nx := 1\n```  \n\n正文在围栏之后\n\n####3. 没有空格的标题\n\n- ###### 列表里的标题\n- 普通项  \n\n联系 me@example.com 或 [写信](mailto:a@b.cc)\n")

	if !strings.Contains(out, "<p>正文在围栏之后</p>") {
		t.Errorf("text after a fence closed with trailing spaces must not be swallowed: %s", out)
	}
	if !strings.Contains(out, ">3. 没有空格的标题</h4>") {
		t.Errorf("headings without a space after the hashes must work: %s", out)
	}
	if !strings.Contains(out, "<li><h6") || strings.Contains(out, "######") {
		t.Errorf("a heading inside a list item must be rendered as one: %s", out)
	}
	if tight := render(t, "- ###### 标题\n  说明文字\n- 二\n"); !strings.Contains(tight, "标题</h6>说明文字</li>") {
		t.Errorf("only the first line of the item is the heading: %s", tight)
	}
	if loose := strings.ReplaceAll(render(t, "- ###### 标题\n  说明文字\n\n- 二\n"), "\n", ""); !strings.Contains(loose, "标题</h6><p>说明文字</p></li>") {
		t.Errorf("the rest of a loose item stays a paragraph: %s", loose)
	}
	if strings.Contains(out, "<br/></li>") || strings.Contains(out, "<br/>\n</li>") {
		t.Errorf("no dangling <br> at the end of list items: %s", out)
	}
	if !strings.Contains(out, `<a href="mailto:me@example.com">me@example.com</a>`) || strings.Count(out, "mailto:a@b.cc") != 1 {
		t.Errorf("bare e-mail addresses become links, existing links stay untouched: %s", out)
	}
}

// TestRenderArticleNeverPanics feeds the renderer random mixtures of the
// constructs the normalizer rewrites (lists, fences, quotes, indentation).
func TestRenderArticleNeverPanics(t *testing.T) {
	fragments := []string{
		"- a", "* b", "+ c", "1. one", "12. twelve", " 2. two", "    - deep", "\t- tab", "      - deeper", "-", "1.", "- [ ] todo", "- ###### h",
		"```", "```go", "~~~", "````", "   ```sh", "\t```", "> quote", "> - quoted item", ">", "> ```", ">> nested",
		"", "", " ", "   ", "text", "    code", "\tcode", "  two", "       seven", "# h1", " ## h2", "####3. tight", "---", "* * *", "***",
		"| a | b |", "| --- | --- |", "<div>", "</div>", "<script>alert(1)</script>", "[^1]: note", "term\n: definition", ":fa-check: :tw-1f606: :tw-zz:",
		"GOBLOGFENCEDBLOCK0END", "GOBLOGFENCEDBLOCK99END", "**bold*", "a \"q\" -- 'x'...", "me@example.com", "\u00a0nbsp", "[TOC]", "![i](https://e.x/i.png)",
	}
	rng := rand.New(rand.NewSource(20260919))
	for i := 0; i < 5000; i++ {
		var b strings.Builder
		for n := rng.Intn(14) + 1; n > 0; n-- {
			b.WriteString(fragments[rng.Intn(len(fragments))])
			b.WriteByte('\n')
		}
		src := b.String()
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("panic %v for input %q", r, src)
				}
			}()
			out, err := RenderArticle(src)
			if err != nil {
				t.Fatalf("error %v for input %q", err, src)
			}
			if strings.Contains(out, "<script") {
				t.Fatalf("script survived for input %q: %s", src, out)
			}
			for _, placeholder := range fencePlaceholder.FindAllString(out, -1) {
				if !strings.Contains(src, placeholder) {
					t.Fatalf("placeholder %s leaked for input %q: %s", placeholder, src, out)
				}
			}
		}()
	}
}
