package filestore

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

// parseFrontmatter splits a document into its "key: value" header and the
// Markdown body. Values may be double-quoted; a quoted value may span several
// lines (descriptions written in the admin textarea do), in which case lines
// are collected until the closing quote. Inside a quoted value neither a "---"
// line nor a "key: value" looking line is treated as structure.
func parseFrontmatter(raw string) (meta map[string]string, content string) {
	meta = make(map[string]string)
	raw = strings.TrimSpace(raw)
	if !strings.HasPrefix(raw, "---") {
		return meta, raw
	}

	lines := strings.Split(raw[3:], "\n")
	parsed := make(map[string]string)
	closed := false
	bodyStart := 0

	var quotedKey string
	var quoted []string

	for i := 1; i < len(lines); i++ { // lines[0] is the remainder of the opening "---" line
		line := strings.TrimRight(lines[i], "\r")

		if quotedKey != "" {
			if endsWithUnescapedQuote(line) {
				trimmed := strings.TrimRight(line, " \t")
				quoted = append(quoted, trimmed[:len(trimmed)-1])
				parsed[quotedKey] = unescapeQuoted(strings.Join(quoted, "\n"))
				quotedKey, quoted = "", nil
			} else {
				quoted = append(quoted, line)
			}
			continue
		}

		if strings.HasPrefix(line, "---") {
			closed = true
			bodyStart = i + 1
			break
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		colonIdx := strings.Index(trimmed, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:colonIdx])
		val := strings.TrimSpace(trimmed[colonIdx+1:])
		switch {
		case len(val) >= 2 && val[0] == '"' && endsWithUnescapedQuote(val):
			parsed[key] = unescapeQuoted(val[1 : len(val)-1])
		case len(val) >= 1 && val[0] == '"':
			// opening quote without a closing one: multi-line value
			quotedKey = key
			quoted = []string{val[1:]}
		default:
			parsed[key] = val
		}
	}

	if !closed {
		if quotedKey != "" {
			// A quote was opened but never closed, so the multi-line scan ran past
			// the real end of the header. Fall back to the line-based parser.
			return parseFrontmatterLegacy(raw)
		}
		return meta, raw
	}
	content = strings.TrimSpace(strings.Join(lines[bodyStart:], "\n"))
	return parsed, content
}

// parseFrontmatterLegacy is the original single-line parser, kept as a safety
// net for headers with unbalanced quotes.
func parseFrontmatterLegacy(raw string) (meta map[string]string, content string) {
	meta = make(map[string]string)
	if !strings.HasPrefix(raw, "---") {
		return meta, raw
	}
	rest := raw[3:]
	idx := strings.Index(rest, "\n---")
	if idx < 0 {
		return meta, raw
	}
	header := rest[:idx]
	content = strings.TrimSpace(rest[idx+4:])

	for _, line := range strings.Split(header, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		colonIdx := strings.Index(line, ":")
		if colonIdx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:colonIdx])
		val := strings.TrimSpace(line[colonIdx+1:])
		if len(val) >= 2 && val[0] == '"' && val[len(val)-1] == '"' {
			val = unescapeQuoted(val[1 : len(val)-1])
		}
		meta[key] = val
	}
	return meta, content
}

// endsWithUnescapedQuote reports whether s (ignoring trailing blanks) ends
// with a double quote that is not escaped by a backslash.
func endsWithUnescapedQuote(s string) bool {
	s = strings.TrimRight(s, " \t")
	if !strings.HasSuffix(s, `"`) {
		return false
	}
	backslashes := 0
	for i := len(s) - 2; i >= 0 && s[i] == '\\'; i-- {
		backslashes++
	}
	return backslashes%2 == 0
}

// quotedEscaper / quotedUnescaper implement the tiny escaping scheme used for
// double-quoted values: backslash and double quote are backslash-escaped.
// Escaping the backslash itself keeps a value that ends in "\" unambiguous.
var (
	quotedEscaper   = strings.NewReplacer(`\`, `\\`, `"`, `\"`)
	quotedUnescaper = strings.NewReplacer(`\\`, `\`, `\"`, `"`)
)

func escapeQuoted(s string) string {
	return quotedEscaper.Replace(s)
}

func unescapeQuoted(s string) string {
	return quotedUnescaper.Replace(s)
}

func (r *FileRepository) parsePost(raw string, slug string) *model.Post {
	meta, content := parseFrontmatter(raw)
	post := &model.Post{
		Id:          slug,
		Title:       meta["title"],
		Identity:    slug,
		Description: meta["description"],
		Content:     content,
	}

	post.Status, _ = strconv.Atoi(meta["status"])
	post.CategoryId, _ = strconv.Atoi(meta["category_id"])
	post.IsTop, _ = strconv.Atoi(meta["is_top"])
	post.WordCount, _ = strconv.Atoi(meta["word_count"])

	if t, err := time.Parse(time.RFC3339, meta["created_at"]); err == nil {
		post.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339, meta["updated_at"]); err == nil {
		post.UpdatedAt = t
	}

	if tagStr := meta["tag_ids"]; tagStr != "" {
		var tagIds []int
		if err := json.Unmarshal([]byte(tagStr), &tagIds); err == nil {
			post.TagIds = tagIds
		}
	}

	tagIds, _ := json.Marshal(post.TagIds)
	post.TagIdString = string(tagIds)

	// only drafts carry tag names: their new tags are not created until publication
	for _, name := range strings.Split(meta["tag_names"], ",") {
		if name = strings.TrimSpace(name); name != "" {
			post.TagNames = append(post.TagNames, name)
		}
	}

	return post
}

// tagIdString is the JSON list the templates and the frontmatter use: "[1, 2]", never "null".
// The ", " separator is what the migrated content files have, so saving a post does not show up as a
// formatting change in the content repository.
func tagIdString(tagIds []int) string {
	parts := make([]string, len(tagIds))
	for i, tagId := range tagIds {
		parts[i] = strconv.Itoa(tagId)
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func postToFrontmatter(post *model.Post) string {
	desc := escapeQuoted(strings.TrimSpace(post.Description))
	title := escapeQuoted(strings.TrimSpace(post.Title))

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "title: \"%s\"\n", title)
	fmt.Fprintf(&b, "status: %d\n", post.Status)
	fmt.Fprintf(&b, "created_at: %s\n", post.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "updated_at: %s\n", post.UpdatedAt.Format(time.RFC3339))
	fmt.Fprintf(&b, "category_id: %d\n", post.CategoryId)
	fmt.Fprintf(&b, "is_top: %d\n", post.IsTop)
	fmt.Fprintf(&b, "tag_ids: %s\n", tagIdString(post.TagIds))
	fmt.Fprintf(&b, "description: \"%s\"\n", desc)
	fmt.Fprintf(&b, "word_count: %d\n", post.WordCount)
	b.WriteString("---\n\n")
	b.WriteString(post.Content)
	b.WriteString("\n") // text files end with a newline; the parser trims the body again

	return b.String()
}

func pageToFrontmatter(page model.Page) string {
	title := escapeQuoted(strings.TrimSpace(page.Title))
	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "id: %s\n", page.Id)
	fmt.Fprintf(&b, "title: \"%s\"\n", title)
	b.WriteString("---\n\n")
	b.WriteString(page.Content)
	b.WriteString("\n")
	return b.String()
}
