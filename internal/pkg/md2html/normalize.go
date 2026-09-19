package md2html

import (
	"fmt"
	"regexp"
	"strings"
)

// fencedBlock is a fenced code block lifted out of the Markdown source. It is
// put back as <pre><code> once blackfriday has rendered everything around it.
type fencedBlock struct {
	lang string
	code string
}

// listLevel is one open nesting level of the list the normalizer is inside of.
// marked lexes the text of a list item recursively after stripping the item's
// indentation, so every level has its own coordinate system.
type listLevel struct {
	bulletCol int // column of the level's bullets, in the level's own coordinates
	space     int // columns marked strips from the lines of the current item
}

const (
	fencePlaceholderPrefix = "GOBLOGFENCEDBLOCK"
	fencePlaceholderSuffix = "END"
)

var (
	fencePlaceholder = regexp.MustCompile(fencePlaceholderPrefix + `(\d+)` + fencePlaceholderSuffix)
	listItem         = regexp.MustCompile(`^([-*+]|\d{1,9}\.)([ \t]+)(.*)$`)
	looseHeading     = regexp.MustCompile(`^ {1,3}#{1,6}`)
	horizontalRule   = regexp.MustCompile(`^([-*_] *){3,}$`)
	quotePrefix      = regexp.MustCompile(`^( {0,3}> ?)+`)
)

// normalizeMarkdown rewrites the loosely written constructs that marked (the
// parser inside editor.md, which every existing post was written against and
// which still powers the admin preview) and blackfriday read differently, so
// that blackfriday produces what the author saw:
//
//   - Fenced code blocks are lifted out and replaced by a placeholder paragraph.
//     blackfriday mishandles them inside list items (a fence with a language
//     swallows the following items, blank lines inside the code get lost) and
//     does not accept a closing fence that is followed by spaces.
//   - Lists are re-indented to the four-spaces-per-level form blackfriday
//     understands. marked continues an item with any block indented by one
//     space or more and nests by relative indentation; blackfriday ends the
//     list instead (the next item then restarts at "1.") or flattens the levels.
//   - An indented line right after a paragraph line is a lazy continuation in
//     marked; blackfriday (with NoEmptyLineBeforeBlock) would start a code block.
//   - " ## title" with up to three leading spaces is a heading.
//
// The returned blocks are referenced by the placeholders in the returned text.
func normalizeMarkdown(src string) (string, []fencedBlock) {
	const (
		kindBlank = iota
		kindPara
		kindList
		kindListCode
		kindOther
	)
	var (
		lines     = strings.Split(src, "\n")
		out       = make([]string, 0, len(lines)+8)
		blocks    []fencedBlock
		stack     []listLevel
		prevKind  = kindBlank
		prevQuote string // quote markers of the previous line, without spaces
		sawBlank  bool
		lastBlank = true
	)
	emit := func(s string) {
		out = append(out, s)
		lastBlank = false
	}
	blankLine := func(prefix string) {
		if !lastBlank {
			out = append(out, strings.TrimRight(prefix, " "))
			lastBlank = true
		}
	}
	pad := func(levels int) string { return strings.Repeat("    ", levels) }
	// colAt translates a source indentation into the coordinates of nesting
	// level d: marked strips up to "space" columns for every enclosing item.
	colAt := func(d, indent int) int {
		for j := 0; j < d; j++ {
			if indent -= stack[j].space; indent < 0 {
				indent = 0
			}
		}
		return indent
	}
	// placeBullet finds the level a list item line belongs to the way marked
	// does: a bullet in the column of a level's bullets is a sibling there, any
	// other bullet belongs to the current item of that level and is looked at
	// again one level deeper. width is the size of the bullet and the spaces
	// after it. It reports false when the line is indented so deeply that it
	// is code inside the deepest item.
	placeBullet := func(indent, width int, afterBlank bool) bool {
		for d := 0; d < len(stack); d++ {
			col := colAt(d, indent)
			if col == stack[d].bulletCol {
				stack = stack[:d+1]
				stack[d].space = col + width
				return true
			}
			if afterBlank && col == 0 {
				// the list of this level is over, the bullet opens a new one in its place
				stack = append(stack[:d], listLevel{bulletCol: 0, space: width})
				return true
			}
		}
		col := colAt(len(stack), indent)
		if col >= 4 {
			return false
		}
		stack = append(stack, listLevel{bulletCol: col, space: col + width})
		return true
	}
	// settle closes the nested levels a block that follows a blank line is not
	// indented for. (Without a blank line the line continues the deepest item.)
	settle := func(indent int) {
		d := 0
		for d+1 < len(stack) && colAt(d+1, indent) >= 1 {
			d++
		}
		stack = stack[:d+1]
	}

	for i := 0; i < len(lines); i++ {
		line := lines[i]
		prefix, content := "", line
		if p, c := splitQuotePrefix(line); p != "" && (len(stack) == 0 || prevQuote != "" || leadingIndent(line) == 0) {
			prefix, content = p, c
		}
		quote := strings.ReplaceAll(prefix, " ", "")
		trimmed := strings.TrimSpace(content)
		indent := leadingIndent(content)

		if trimmed == "" {
			if prefix != "" {
				out = append(out, strings.TrimRight(line, " \t"))
			} else {
				out = append(out, "")
			}
			lastBlank, sawBlank, prevKind, prevQuote = true, true, kindBlank, quote
			continue
		}

		// a quoted paragraph continued without the ">" marker
		if quote == "" && prevQuote != "" && prevKind == kindPara && !startsBlock(trimmed) {
			emit(trimmed)
			continue
		}
		if quote != prevQuote {
			// entering or leaving a quote ends whatever block was open
			if len(stack) > 0 {
				stack = stack[:0]
				blankLine("")
				sawBlank = true
			}
			if prevKind != kindBlank {
				prevKind = kindOther
			}
		}
		prevQuote = quote

		m := listItem.FindStringSubmatch(strings.TrimLeft(content, " \t"))
		isItem := m != nil && !horizontalRule.MatchString(trimmed)

		width := 0
		if isItem {
			width = len(m[1]) + leadingIndent(m[2])
		}
		itemCode := false // indented code inside a list item
		if len(stack) > 0 {
			switch {
			case horizontalRule.MatchString(trimmed):
				stack = stack[:0] // a rule always ends the list
				blankLine(prefix)
			case sawBlank && indent == 0 && !isItem:
				stack = stack[:0] // marked: a blank line followed by unindented text ends the list
			case isItem:
				if !placeBullet(indent, width, sawBlank) {
					isItem, itemCode = false, true
				}
			default:
				if sawBlank {
					settle(indent)
				}
				itemCode = colAt(len(stack), indent) >= 4
			}
		}

		// fenced code
		if n, ch := fenceRun(trimmed); n >= 3 && !itemCode && !isItem && (len(stack) > 0 || indent <= 3) {
			if lang, ok := fenceInfo(trimmed[n:], ch); ok {
				if end := closingFence(lines, i+1, prefix != "", ch, n); end >= 0 {
					body := make([]string, 0, end-i-1)
					for _, l := range lines[i+1 : end] {
						if prefix != "" {
							if p, c := splitQuotePrefix(l); p != "" {
								l = c
							}
						}
						body = append(body, trimIndent(l, indent))
					}
					code := strings.TrimRight(strings.Join(body, "\n"), " \t\n")
					if code != "" {
						code += "\n"
					}
					placeholder := fmt.Sprintf("%s%d%s", fencePlaceholderPrefix, len(blocks), fencePlaceholderSuffix)
					blocks = append(blocks, fencedBlock{lang: lang, code: code})

					blankLine(prefix)
					emit(prefix + pad(len(stack)) + placeholder)
					blankLine(prefix)
					i = end
					prevKind, sawBlank = kindOther, false
					if len(stack) > 0 {
						prevKind = kindList
					}
					continue
				}
			}
		}

		if len(stack) > 0 {
			switch {
			case itemCode:
				// indented code inside the item: keep it relative to the item text
				if prevKind != kindListCode {
					blankLine(prefix)
				}
				emit(prefix + pad(len(stack)+1) + strings.Repeat(" ", colAt(len(stack), indent)-4) + strings.TrimLeft(content, " \t"))
				prevKind = kindListCode
			case isItem:
				emit(prefix + pad(len(stack)-1) + m[1] + " " + m[3])
				prevKind = kindList
			default:
				// a continuation block or a lazy continuation line of the item
				emit(prefix + pad(len(stack)) + strings.TrimLeft(content, " \t"))
				prevKind = kindList
			}
			sawBlank = false
			continue
		}

		switch {
		case isItem && indent <= 3:
			stack = append(stack, listLevel{bulletCol: indent, space: indent + width})
			emit(prefix + m[1] + " " + m[3])
			prevKind = kindList
		case indent >= 4 && prevKind == kindPara && !startsBlock(trimmed):
			// lazy continuation of the paragraph above. (An indented line that
			// looks like a list item, quote, heading… ends the paragraph in
			// marked and is then lexed as indented code, so it is left alone.)
			emit(prefix + trimmed)
		case indent >= 4:
			emit(line) // a real indented code block
			prevKind = kindOther
		case looseHeading.MatchString(content):
			emit(prefix + strings.TrimLeft(content, " "))
			prevKind = kindOther
		case strings.HasPrefix(trimmed, "#"), strings.HasPrefix(trimmed, "|"), strings.HasPrefix(trimmed, "<"), horizontalRule.MatchString(trimmed):
			emit(line)
			prevKind = kindOther
		default:
			emit(line)
			prevKind = kindPara
		}
		sawBlank = false
	}
	return strings.Join(out, "\n"), blocks
}

// startsBlock reports whether a de-indented line would open a new block.
func startsBlock(trimmed string) bool {
	if n, _ := fenceRun(trimmed); n >= 3 {
		return true
	}
	return listItem.MatchString(trimmed) || strings.HasPrefix(trimmed, ">") || strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "<") || horizontalRule.MatchString(trimmed)
}

// leadingIndent measures indentation in columns; a tab counts as four.
func leadingIndent(s string) int {
	n := 0
	for _, r := range s {
		switch r {
		case ' ':
			n++
		case '\t':
			n += 4
		default:
			return n
		}
	}
	return n
}

// trimIndent removes up to n columns of leading whitespace.
func trimIndent(s string, n int) string {
	for n > 0 && s != "" {
		switch {
		case s[0] == ' ':
			n--
		case s[0] == '\t' && n >= 4:
			n -= 4
		default:
			return s
		}
		s = s[1:]
	}
	return s
}

// splitQuotePrefix separates leading blockquote markers ("> > ") from the rest.
func splitQuotePrefix(line string) (prefix, content string) {
	prefix = quotePrefix.FindString(line)
	return prefix, line[len(prefix):]
}

// fenceRun returns the length and character of a leading ``` / ~~~ run.
func fenceRun(trimmed string) (int, byte) {
	if trimmed == "" || (trimmed[0] != '`' && trimmed[0] != '~') {
		return 0, 0
	}
	ch := trimmed[0]
	n := 0
	for n < len(trimmed) && trimmed[n] == ch {
		n++
	}
	return n, ch
}

// fenceInfo validates what follows the opening fence and returns the language.
func fenceInfo(rest string, ch byte) (lang string, ok bool) {
	info := strings.TrimLeft(strings.TrimSpace(rest), " .")
	if ch == '`' && strings.Contains(info, "`") {
		return "", false // an inline code span such as ```x``` at the start of a line
	}
	if fields := strings.Fields(info); len(fields) > 0 {
		lang = fields[0]
	}
	return lang, true
}

// closingFence finds the line that closes a fence opened with n times ch.
// Like marked (and blackfriday) an unclosed fence is not a code block.
func closingFence(lines []string, from int, quoted bool, ch byte, n int) int {
	for j := from; j < len(lines); j++ {
		l := lines[j]
		if quoted {
			if p, c := splitQuotePrefix(l); p != "" {
				l = c
			}
		}
		t := strings.TrimSpace(l)
		if run, c := fenceRun(t); run >= n && c == ch && strings.Trim(t, string(ch)) == "" {
			return j
		}
	}
	return -1
}
