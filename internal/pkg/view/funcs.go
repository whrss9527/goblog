package view

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	mdFencedCode = regexp.MustCompile("(?s)```.*?(```|$)")
	mdImage      = regexp.MustCompile(`!\[[^\]]*\]\([^)]*\)`)
	mdLink       = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	htmlTag      = regexp.MustCompile(`<[^>]+>`)
	mdLinePrefix = regexp.MustCompile(`^\s*(#{1,6}\s+|>\s?|[-*+]\s+|\d+\.\s+)`)
	mdEmphasis   = regexp.MustCompile("[*`~]+|__")
)

// Excerpt turns a (possibly multi-line, possibly Markdown) text into a single
// line of plain text no longer than limit runes. It is used for list
// summaries and <meta name="description">.
func Excerpt(s string, limit int) string {
	s = mdFencedCode.ReplaceAllString(s, " ")
	s = mdImage.ReplaceAllString(s, "")
	s = mdLink.ReplaceAllString(s, "$1")
	s = htmlTag.ReplaceAllString(s, "")

	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		line = mdLinePrefix.ReplaceAllString(line, "")
		line = mdEmphasis.ReplaceAllString(line, "")
		line = strings.Join(strings.Fields(line), " ")
		if line == "" || strings.Trim(line, "-=| :") == "" {
			continue
		}
		if b.Len() > 0 {
			// Only Latin words need a separating space; CJK sentences read
			// better when simply concatenated.
			prev, _ := utf8.DecodeLastRuneInString(b.String())
			next, _ := utf8.DecodeRuneInString(line)
			if isLatinWordRune(prev) && isLatinWordRune(next) {
				b.WriteByte(' ')
			}
		}
		b.WriteString(line)
	}

	out := b.String()
	if limit > 0 && utf8.RuneCountInString(out) > limit {
		runes := []rune(out)
		out = strings.TrimRightFunc(string(runes[:limit]), func(r rune) bool {
			return unicode.IsSpace(r) || unicode.IsPunct(r)
		}) + "…"
	}
	return out
}

func isLatinWordRune(r rune) bool {
	return r < utf8.RuneSelf && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(".,;:!?)", r))
}

// ReadingMinutes estimates the reading time for a post from its word count
// (roughly 400 CJK characters per minute), never less than one minute.
func ReadingMinutes(wordCount int) int {
	if wordCount <= 0 {
		return 1
	}
	return int(math.Max(1, math.Ceil(float64(wordCount)/400)))
}

// HumanCount formats counters the way Chinese readers expect: 9527 -> "9527",
// 16260 -> "1.6万".
func HumanCount(n int) string {
	if n < 10000 {
		return strconv.Itoa(n)
	}
	s := fmt.Sprintf("%.1f", float64(n)/10000)
	return strings.TrimSuffix(s, ".0") + "万"
}

// AbsoluteURL resolves a reference found in content or config against the site
// address. Social previews and structured data need absolute URLs; "" is
// returned for anything that cannot be made absolute.
func AbsoluteURL(host, ref string) string {
	ref = strings.TrimSpace(ref)
	switch {
	case ref == "":
		return ""
	case strings.HasPrefix(ref, "https://"), strings.HasPrefix(ref, "http://"):
		return ref
	case strings.HasPrefix(ref, "//"):
		return "https:" + ref
	case strings.HasPrefix(ref, "/"):
		return strings.TrimRight(host, "/") + ref
	}
	return ""
}

// SiteLogo is the absolute address of the site logo, the fallback preview image.
func SiteLogo(host, cdn string) string {
	return AbsoluteURL(host, strings.TrimRight(cdn, "/")+"/logo.png")
}
