package md2html

import (
	"bytes"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/russross/blackfriday/v2"
	"golang.org/x/net/html"
)

// The on-site article renderer. It mirrors what editor.md (marked with
// gfm + breaks + smartypants) used to produce in the browser closely enough
// that existing posts look the same, but runs on the server so the article is
// in the HTML response: no jQuery / editor.md download, no blank page while
// scripts load, readable without JavaScript and by crawlers.

// No SpaceHeadings: marked accepts "####3. title" as a heading and posts rely on it.
const articleExtensions = blackfriday.NoIntraEmphasis |
	blackfriday.Tables |
	blackfriday.FencedCode |
	blackfriday.Autolink |
	blackfriday.Strikethrough |
	blackfriday.HardLineBreak |
	blackfriday.NoEmptyLineBeforeBlock |
	blackfriday.BackslashLineBreak |
	blackfriday.DefinitionLists |
	blackfriday.Footnotes

// No blackfriday.Smartypants: it also rewrites "->" (en dash), "1/2", "(c)"…
// which marked never did. smartypants below applies marked's own rules.
const articleHTMLFlags = blackfriday.FootnoteReturnLinks

var (
	// features only the client-side renderer supports
	clientOnlyFence = regexp.MustCompile("(?m)^\\s*```\\s*(flow|seq|sequence|math|latex|katex)\\b")
	texBlock        = regexp.MustCompile(`\$\$[^$]+\$\$`)
	tocMarker       = regexp.MustCompile(`(?mi)^\s*\[TOCM?\]\s*$`)
	shortcode       = regexp.MustCompile(`:(fa|tw)-([a-z0-9-]+):`)
	listHeading     = regexp.MustCompile(`^\s*(#{1,6})\s+`)
	bareEmail       = regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}`)
	openSingleQuote = regexp.MustCompile(`(^|[-\x{2014}/(\[{"\s])'`)
	openDoubleQuote = regexp.MustCompile(`(^|[-\x{2014}/(\[{\x{2018}\s])"`)
	slugStrip       = regexp.MustCompile(`[^\p{L}\p{N}_-]+`)
	slugDashes      = regexp.MustCompile(`-{2,}`)
)

// faEmoji maps the FontAwesome shortcodes editor.md understands to emoji, so
// articles do not need the FontAwesome web font any more.
var faEmoji = map[string]string{
	"hand-o-right": "👉", "hand-o-left": "👈", "hand-o-up": "👆", "hand-o-down": "👇",
	"check": "✅", "check-circle": "✅", "check-square": "✅", "times": "❌", "times-circle": "❌", "close": "❌",
	"warning": "⚠️", "exclamation-triangle": "⚠️", "exclamation-circle": "❗", "info-circle": "ℹ️",
	"question-circle": "❓", "star": "⭐", "heart": "❤️", "thumbs-up": "👍", "thumbs-o-up": "👍",
	"thumbs-down": "👎", "lightbulb-o": "💡", "bolt": "⚡", "fire": "🔥", "rocket": "🚀", "bug": "🐛",
	"link": "🔗", "lock": "🔒", "key": "🔑", "book": "📖", "pencil": "✏️", "flag": "🚩", "bell": "🔔",
	"smile-o": "🙂", "frown-o": "🙁", "coffee": "☕", "github": "🐙", "arrow-right": "➡️", "arrow-left": "⬅️",
	"arrow-up": "⬆️", "arrow-down": "⬇️", "chevron-circle-right": "▶️", "chevron-right": "▶️", "angle-right": "▶️",
	"envelope": "✉️", "envelope-o": "✉️", "calendar": "📅", "clock-o": "🕒", "comment": "💬", "comments": "💬",
	"home": "🏠", "search": "🔍", "cog": "⚙️", "gear": "⚙️", "download": "⬇️", "upload": "⬆️", "tag": "🏷️", "tags": "🏷️",
}

// NeedsClientRenderer reports whether the Markdown uses editor.md features
// that have no server-side counterpart (flow charts, sequence diagrams, TeX).
// Such posts keep being rendered in the browser.
func NeedsClientRenderer(markdown string) bool {
	return clientOnlyFence.MatchString(markdown) || texBlock.MatchString(markdown)
}

// RenderArticle converts Markdown to the HTML shown on post and page views.
func RenderArticle(markdown string) (string, error) {
	// the same clean-up marked does before lexing (tabs are kept: code blocks
	// show them four columns wide via CSS, and copying Go code yields real tabs)
	src := strings.NewReplacer("\r\n", "\n", "\r", "\n", "\u00a0", " ", "\u2424", "\n").Replace(markdown)
	src = tocMarker.ReplaceAllString(src, "")
	src, fenced := normalizeMarkdown(src)

	renderer := blackfriday.NewHTMLRenderer(blackfriday.HTMLRendererParameters{Flags: articleHTMLFlags})
	out := blackfriday.Run([]byte(src), blackfriday.WithExtensions(articleExtensions), blackfriday.WithRenderer(renderer))

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(out))
	if err != nil {
		return "", fmt.Errorf("parse rendered article: %w", err)
	}
	body := doc.Find("body")

	// Same policy the browser renderer applied (htmlDecode filter): authors may
	// use inline HTML, but never active content.
	body.Find("script, style, iframe, object, embed, link, meta, base, form").Remove()
	body.Find("*").Each(func(_ int, s *goquery.Selection) {
		node := s.Get(0)
		kept := node.Attr[:0]
		for _, attr := range node.Attr {
			name := strings.ToLower(attr.Key)
			if strings.HasPrefix(name, "on") {
				continue
			}
			if (name == "href" || name == "src") && isScriptURL(attr.Val) {
				continue
			}
			kept = append(kept, attr)
		}
		node.Attr = kept
	})

	restoreFencedBlocks(body, fenced)
	smartypants(body)
	replaceShortcodes(body)
	headingsInListItems(body)
	addHeadingIDs(body)
	renderTaskLists(body)
	linkBareEmails(body)
	trimTrailingBreaks(body)

	body.Find("pre").Each(func(_ int, s *goquery.Selection) {
		s.AddClass("prettyprint", "linenums")
	})
	body.Find("img").Each(func(i int, s *goquery.Selection) {
		if i > 0 { // the first image is likely above the fold
			s.SetAttr("loading", "lazy")
		}
		s.SetAttr("decoding", "async")
	})
	body.Find("a[href]").Each(func(_ int, s *goquery.Selection) {
		href, _ := s.Attr("href")
		if u, err := url.Parse(href); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
			s.SetAttr("target", "_blank")
			s.SetAttr("rel", "noopener noreferrer")
		}
	})

	htmlOut, err := body.Html()
	if err != nil {
		return "", fmt.Errorf("serialize rendered article: %w", err)
	}
	return htmlOut, nil
}

func isScriptURL(v string) bool {
	clean := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return -1
		}
		return unicode.ToLower(r)
	}, v)
	return strings.HasPrefix(clean, "javascript:") || strings.HasPrefix(clean, "vbscript:") || strings.HasPrefix(clean, "data:text/html")
}

// replaceShortcodes handles editor.md's emoji shortcodes in text nodes outside
// code: :fa-name: (FontAwesome) becomes the closest emoji, :tw-1f606: (Twemoji,
// addressed by code point) becomes the character itself. editor.md rendered
// both through web fonts / image CDNs that the site no longer has to load.
func replaceShortcodes(body *goquery.Selection) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "code" || n.Data == "pre") {
			return
		}
		if n.Type == html.TextNode && (strings.Contains(n.Data, ":fa-") || strings.Contains(n.Data, ":tw-")) {
			n.Data = shortcode.ReplaceAllStringFunc(n.Data, func(m string) string {
				parts := shortcode.FindStringSubmatch(m)
				if parts[1] == "fa" {
					return faEmoji[parts[2]] // unknown icons are dropped, as the missing font did
				}
				if emoji, ok := twemoji(parts[2]); ok {
					return emoji
				}
				return m
			})
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range body.Nodes {
		walk(n)
	}
}

// smartypants applies the typographic replacements of marked's smartypants
// option (curly quotes, "--" to an em dash, "..." to an ellipsis) to prose.
// Code, autolinked URLs and raw HTML attributes are left alone.
func smartypants(body *goquery.Selection) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "code", "pre", "kbd", "samp", "var", "script", "style", "textarea":
				return
			case "a":
				if isAutolink(n) {
					return
				}
			}
		}
		if n.Type == html.TextNode && strings.ContainsAny(n.Data, `-'".`) {
			t := strings.ReplaceAll(n.Data, "--", "\u2014")
			t = openSingleQuote.ReplaceAllString(t, "${1}\u2018")
			t = strings.ReplaceAll(t, "'", "\u2019")
			t = openDoubleQuote.ReplaceAllString(t, "${1}\u201c")
			t = strings.ReplaceAll(t, `"`, "\u201d")
			n.Data = strings.ReplaceAll(t, "...", "\u2026")
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	for _, n := range body.Nodes {
		walk(n)
	}
}

// isAutolink reports whether a link shows its own address.
func isAutolink(a *html.Node) bool {
	if a.FirstChild == nil || a.FirstChild != a.LastChild || a.FirstChild.Type != html.TextNode {
		return false
	}
	text := a.FirstChild.Data
	for _, attr := range a.Attr {
		if attr.Key == "href" {
			return attr.Val == text || attr.Val == "mailto:"+text || attr.Val == "http://"+text
		}
	}
	return false
}

// twemoji converts "1f1e8-1f1f3" (hex code points) to the emoji it names.
func twemoji(code string) (string, bool) {
	var b strings.Builder
	fields := strings.Split(code, "-")
	for _, f := range fields {
		cp, err := strconv.ParseUint(f, 16, 32)
		if err != nil || cp < 0x20 || cp > unicode.MaxRune || !utf8.ValidRune(rune(cp)) {
			return "", false
		}
		b.WriteRune(rune(cp))
	}
	if r := []rune(b.String()); len(r) == 1 && r[0] < 0x1F000 {
		b.WriteRune(0xFE0F) // ask for the emoji glyph of characters that default to text
	}
	return b.String(), true
}

// headingsInListItems handles "- ###### title": marked renders a heading inside
// the list item, blackfriday leaves the hashes in the text. The heading is the
// first line only; lazy continuation lines stay ordinary item text.
func headingsInListItems(body *goquery.Selection) {
	body.Find("li").Each(func(_ int, li *goquery.Selection) {
		item := li.Get(0)
		container := item
		if first := item.FirstChild; first != nil && first.Type == html.ElementNode && first.Data == "p" {
			container = first
		}
		first := container.FirstChild
		if first == nil || first.Type != html.TextNode {
			return
		}
		m := listHeading.FindStringSubmatch(first.Data)
		if m == nil {
			return
		}
		first.Data = first.Data[len(m[0]):]
		heading := &html.Node{Type: html.ElementNode, Data: fmt.Sprintf("h%d", len(m[1]))}
		for n := first; n != nil; {
			next := n.NextSibling
			if n.Type == html.ElementNode && blockElements[n.Data] {
				break
			}
			container.RemoveChild(n)
			if n.Type == html.ElementNode && n.Data == "br" {
				if next != nil && next.Type == html.TextNode {
					next.Data = strings.TrimLeft(next.Data, "\n")
				}
				break // end of the heading line
			}
			heading.AppendChild(n)
			n = next
		}
		item.InsertBefore(heading, item.FirstChild)
		if container != item && isBlankInline(container) {
			item.RemoveChild(container)
		}
	})
}

// linkBareEmails keeps editor.md's behaviour of turning a plain e-mail address
// into a mailto: link.
func linkBareEmails(body *goquery.Selection) {
	var walk func(n *html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "a" || n.Data == "code" || n.Data == "pre") {
			return
		}
		if n.Type == html.TextNode && strings.Contains(n.Data, "@") {
			loc := bareEmail.FindStringIndex(n.Data)
			if loc == nil {
				return
			}
			before, addr, after := n.Data[:loc[0]], n.Data[loc[0]:loc[1]], n.Data[loc[1]:]
			link := &html.Node{Type: html.ElementNode, Data: "a", Attr: []html.Attribute{{Key: "href", Val: "mailto:" + addr}}}
			link.AppendChild(&html.Node{Type: html.TextNode, Data: addr})
			rest := &html.Node{Type: html.TextNode, Data: after}
			n.Data = before
			n.Parent.InsertBefore(link, n.NextSibling)
			n.Parent.InsertBefore(rest, link.NextSibling)
			walk(rest) // more addresses in the remainder
			return
		}
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			walk(c)
			c = next
		}
	}
	for _, n := range body.Nodes {
		walk(n)
	}
}

// restoreFencedBlocks puts the code blocks normalizeMarkdown lifted out of the
// source back in place of their placeholders.
func restoreFencedBlocks(body *goquery.Selection, blocks []fencedBlock) {
	if len(blocks) == 0 {
		return
	}
	var targets []*html.Node
	var walk func(n *html.Node, inCode bool)
	walk = func(n *html.Node, inCode bool) {
		if n.Type == html.ElementNode && (n.Data == "pre" || n.Data == "code") {
			inCode = true
		}
		if n.Type == html.TextNode && strings.Contains(n.Data, fencePlaceholderPrefix) {
			if inCode {
				// blackfriday read the surroundings as code (deeply indented or
				// inside an unclosed fence): the block is just more of that code
				n.Data = fencePlaceholder.ReplaceAllStringFunc(n.Data, func(m string) string {
					idx, err := strconv.Atoi(fencePlaceholder.FindStringSubmatch(m)[1])
					if err != nil || idx >= len(blocks) {
						return m
					}
					return strings.TrimRight(blocks[idx].code, "\n")
				})
				return
			}
			targets = append(targets, n)
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c, inCode)
		}
	}
	for _, n := range body.Nodes {
		walk(n, false)
	}

	for _, n := range targets {
		for n != nil {
			loc := fencePlaceholder.FindStringSubmatchIndex(n.Data)
			if loc == nil {
				break
			}
			idx, err := strconv.Atoi(n.Data[loc[2]:loc[3]])
			if err != nil || idx >= len(blocks) {
				break // not one of ours
			}
			n = replacePlaceholder(n, loc[0], loc[1], blocks[idx])
		}
	}
}

// replacePlaceholder swaps n.Data[from:to] for the code block and returns the
// text node holding what followed the placeholder.
func replacePlaceholder(n *html.Node, from, to int, block fencedBlock) *html.Node {
	code := &html.Node{Type: html.ElementNode, Data: "code"}
	if block.lang != "" {
		code.Attr = []html.Attribute{{Key: "class", Val: "language-" + block.lang}}
	}
	code.AppendChild(&html.Node{Type: html.TextNode, Data: block.code})
	pre := &html.Node{Type: html.ElementNode, Data: "pre"}
	pre.AppendChild(code)

	rest := &html.Node{Type: html.TextNode, Data: n.Data[to:]}
	n.Data = n.Data[:from]
	parent := n.Parent

	if parent.Type != html.ElementNode || parent.Data != "p" || parent.Parent == nil {
		parent.InsertBefore(pre, n.NextSibling)
		parent.InsertBefore(rest, pre.NextSibling)
		return rest
	}

	// <pre> cannot live inside <p>: split the paragraph around it
	tail := &html.Node{Type: html.ElementNode, Data: "p"}
	tail.AppendChild(rest)
	for next := n.NextSibling; next != nil; next = n.NextSibling {
		parent.RemoveChild(next)
		tail.AppendChild(next)
	}
	grand := parent.Parent
	grand.InsertBefore(pre, parent.NextSibling)
	grand.InsertBefore(tail, pre.NextSibling)
	for _, p := range []*html.Node{parent, tail} {
		if isBlankInline(p) {
			if p == tail {
				rest = nil
			}
			grand.RemoveChild(p)
		}
	}
	return rest
}

// isBlankInline reports whether an element holds nothing but whitespace and <br>.
func isBlankInline(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		switch {
		case c.Type == html.TextNode && strings.TrimSpace(c.Data) == "":
		case c.Type == html.ElementNode && c.Data == "br":
		default:
			return false
		}
	}
	return true
}

var blockElements = map[string]bool{
	"ul": true, "ol": true, "pre": true, "blockquote": true, "table": true, "p": true, "div": true, "hr": true, "dl": true,
	"h1": true, "h2": true, "h3": true, "h4": true, "h5": true, "h6": true,
}

// trimTrailingBreaks drops the <br> that HardLineBreak leaves where marked has
// none: at the very end of list items, paragraphs and cells (source lines
// ending in two spaces) and right before a nested block such as a sub-list.
func trimTrailingBreaks(body *goquery.Selection) {
	body.Find("li, p, td, th, dd").Each(func(_ int, s *goquery.Selection) {
		node := s.Get(0)
		for last := node.LastChild; last != nil; last = node.LastChild {
			switch {
			case last.Type == html.TextNode && strings.TrimSpace(last.Data) == "":
				node.RemoveChild(last)
			case last.Type == html.ElementNode && last.Data == "br":
				node.RemoveChild(last)
			default:
				return
			}
		}
	})
	// blackfriday writes "<br />\n": without the newline the DOM equals marked's
	// (the newline shows up as an empty line in front of block-level images)
	body.Find("br").Each(func(_ int, s *goquery.Selection) {
		if next := s.Get(0).NextSibling; next != nil && next.Type == html.TextNode {
			next.Data = strings.TrimLeft(next.Data, "\n")
		}
	})
	body.Find("li > br, dd > br").Each(func(_ int, s *goquery.Selection) {
		br := s.Get(0)
		next := br.NextSibling
		for next != nil && next.Type == html.TextNode && strings.TrimSpace(next.Data) == "" {
			next = next.NextSibling
		}
		if next != nil && next.Type == html.ElementNode && blockElements[next.Data] {
			br.Parent.RemoveChild(br)
		}
	})
}

// renderTaskLists turns "- [ ] todo" / "- [x] done" items into disabled checkboxes
// (GitHub style task lists, which blackfriday does not know about).
func renderTaskLists(body *goquery.Selection) {
	body.Find("li").Each(func(_ int, li *goquery.Selection) {
		node := li.Get(0).FirstChild
		if node != nil && node.Type == html.ElementNode && node.Data == "p" {
			node = node.FirstChild
		}
		if node == nil || node.Type != html.TextNode {
			return
		}
		var checked bool
		switch {
		case strings.HasPrefix(node.Data, "[ ] "):
		case strings.HasPrefix(node.Data, "[x] "), strings.HasPrefix(node.Data, "[X] "):
			checked = true
		default:
			return
		}
		node.Data = node.Data[4:]
		box := &html.Node{Type: html.ElementNode, Data: "input", Attr: []html.Attribute{
			{Key: "type", Val: "checkbox"}, {Key: "disabled", Val: ""}, {Key: "class", Val: "task-list-item-checkbox"},
		}}
		if checked {
			box.Attr = append(box.Attr, html.Attribute{Key: "checked", Val: ""})
		}
		node.Parent.InsertBefore(box, node)
		li.AddClass("task-list-item")
	})
}

// Slugify builds a readable fragment id from a heading; static/js/article.js
// implements the same rules for client-rendered posts.
func Slugify(text string) string {
	s := strings.ToLower(strings.TrimSpace(text))
	s = strings.Join(strings.Fields(s), "-")
	s = slugStrip.ReplaceAllString(s, "")
	s = slugDashes.ReplaceAllString(s, "-")
	return strings.Trim(s, "-")
}

func addHeadingIDs(body *goquery.Selection) {
	seen := make(map[string]bool)
	body.Find("h1, h2, h3, h4, h5, h6").Each(func(i int, s *goquery.Selection) {
		base := Slugify(s.Text())
		if base == "" {
			base = fmt.Sprintf("section-%d", i+1)
		}
		id := base
		for n := 2; seen[id]; n++ {
			id = fmt.Sprintf("%s-%d", base, n)
		}
		seen[id] = true
		s.SetAttr("id", id)
	})
}
