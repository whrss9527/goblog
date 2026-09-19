package filestore

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"goblog/internal/pkg/model"
)

func TestParseFrontmatter_Basic(t *testing.T) {
	raw := `---
title: "Hello World"
status: 1
category_id: 2
description: "A test post"
---

This is the content.`

	meta, content := parseFrontmatter(raw)

	assert.Equal(t, "Hello World", meta["title"])
	assert.Equal(t, "1", meta["status"])
	assert.Equal(t, "2", meta["category_id"])
	assert.Equal(t, "A test post", meta["description"])
	assert.Equal(t, "This is the content.", content)
}

func TestParseFrontmatter_NoFrontmatter(t *testing.T) {
	raw := "Just plain content without frontmatter."
	meta, content := parseFrontmatter(raw)

	assert.Empty(t, meta)
	assert.Equal(t, raw, content)
}

func TestParseFrontmatter_EmptyContent(t *testing.T) {
	raw := `---
title: "Empty"
---`

	meta, content := parseFrontmatter(raw)

	assert.Equal(t, "Empty", meta["title"])
	assert.Equal(t, "", content)
}

func TestParseFrontmatter_MissingClosingDelimiter(t *testing.T) {
	raw := `---
title: "Broken"
No closing delimiter here.`

	meta, content := parseFrontmatter(raw)

	assert.Empty(t, meta)
	assert.Equal(t, raw, content)
}

func TestParsePost(t *testing.T) {
	r := &FileRepository{}
	raw := `---
title: "Test Post"
status: 1
created_at: 2024-01-15T10:30:00Z
updated_at: 2024-06-20T14:00:00Z
category_id: 3
is_top: 1
tag_ids: [1,2,3]
description: "A description"
word_count: 150
---

# Hello
This is markdown content.`

	post := r.parsePost(raw, "test-slug")

	assert.Equal(t, "test-slug", post.Id)
	assert.Equal(t, "test-slug", post.Identity)
	assert.Equal(t, "Test Post", post.Title)
	assert.Equal(t, 1, post.Status)
	assert.Equal(t, 3, post.CategoryId)
	assert.Equal(t, 1, post.IsTop)
	assert.Equal(t, 150, post.WordCount)
	assert.Equal(t, "A description", post.Description)
	assert.Equal(t, []int{1, 2, 3}, post.TagIds)
	assert.Equal(t, 2024, post.CreatedAt.Year())
	assert.Equal(t, time.January, post.CreatedAt.Month())
	assert.Contains(t, post.Content, "# Hello")
}

func TestPostToFrontmatterRoundTrip(t *testing.T) {
	original := &model.Post{
		Title:       "Round Trip Test",
		Status:      1,
		CreatedAt:   time.Date(2024, 3, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 6, 20, 14, 0, 0, 0, time.UTC),
		CategoryId:  5,
		IsTop:       0,
		TagIds:      []int{10, 20},
		Description: "Some description",
		WordCount:   300,
		Content:     "# Title\n\nBody text here.",
	}

	serialized := postToFrontmatter(original)
	r := &FileRepository{}
	parsed := r.parsePost(serialized, "round-trip")

	assert.Equal(t, original.Title, parsed.Title)
	assert.Equal(t, original.Status, parsed.Status)
	assert.Equal(t, original.CategoryId, parsed.CategoryId)
	assert.Equal(t, original.IsTop, parsed.IsTop)
	assert.Equal(t, original.TagIds, parsed.TagIds)
	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, original.WordCount, parsed.WordCount)
	assert.Equal(t, original.Content, parsed.Content)
	assert.True(t, original.CreatedAt.Equal(parsed.CreatedAt))
	assert.True(t, original.UpdatedAt.Equal(parsed.UpdatedAt))
}

func TestPostToFrontmatter_EscapesQuotes(t *testing.T) {
	post := &model.Post{
		Title:       `He said "hello"`,
		Description: `A "quoted" description`,
		Content:     "content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	serialized := postToFrontmatter(post)

	r := &FileRepository{}
	parsed := r.parsePost(serialized, "quote-test")
	assert.Equal(t, `He said "hello"`, parsed.Title)
	assert.Equal(t, `A "quoted" description`, parsed.Description)
}

func TestPageToFrontmatterRoundTrip(t *testing.T) {
	page := model.Page{
		Id:      "about",
		Title:   "About Me",
		Content: "This is the about page.\n\nWith multiple paragraphs.",
	}

	serialized := pageToFrontmatter(page)
	meta, content := parseFrontmatter(serialized)

	assert.Equal(t, "about", meta["id"])
	assert.Equal(t, "About Me", meta["title"])
	assert.Equal(t, page.Content, content)
}

func TestParseFrontmatter_MultiLineQuotedValue(t *testing.T) {
	raw := "---\n" +
		"title: \"多行摘要\"\n" +
		"status: 1\n" +
		"description: \"第一行\n" +
		"![img](https://example.com/a.png)\n" +
		"status: 0\n" +
		"---\n" +
		"最后一行 \\\"带引号\\\"\"\n" +
		"word_count: 42\n" +
		"---\n\n" +
		"正文"

	meta, content := parseFrontmatter(raw)

	assert.Equal(t, "多行摘要", meta["title"])
	assert.Equal(t, "1", meta["status"], "a key-looking line inside a quoted value must not override real keys")
	assert.Equal(t, "第一行\n![img](https://example.com/a.png)\nstatus: 0\n---\n最后一行 \"带引号\"", meta["description"])
	assert.Equal(t, "42", meta["word_count"])
	assert.Equal(t, "正文", content)
}

func TestParseFrontmatter_TrailingNewlineInsideQuotes(t *testing.T) {
	// This is what the admin textarea used to produce: the closing quote sits
	// alone on the next line.
	raw := "---\ntitle: \"T\"\ndescription: \"只有一行。\n\"\nword_count: 7\n---\n\nbody"

	meta, content := parseFrontmatter(raw)

	assert.Equal(t, "只有一行。\n", meta["description"])
	assert.Equal(t, "7", meta["word_count"])
	assert.Equal(t, "body", content)
}

func TestPostToFrontmatterRoundTrip_Backslashes(t *testing.T) {
	original := &model.Post{
		Title:       `C:\temp\`,
		Status:      1,
		Description: `regex \d+ and a trailing slash \`,
		Content:     "content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	r := &FileRepository{}
	parsed := r.parsePost(postToFrontmatter(original), "backslash")

	assert.Equal(t, original.Title, parsed.Title)
	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, 1, parsed.Status)
}

func TestParseFrontmatter_UnterminatedQuoteFallsBack(t *testing.T) {
	raw := "---\ntitle: \"Broken\nstatus: 1\n---\n\nbody"

	meta, content := parseFrontmatter(raw)

	assert.Equal(t, "1", meta["status"])
	assert.Equal(t, "body", content)
}

func TestPostToFrontmatterRoundTrip_MultiLineDescription(t *testing.T) {
	original := &model.Post{
		Title:       "Multi",
		Status:      1,
		Description: "line one\nline \"two\"\nkey: value",
		Content:     "content",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	r := &FileRepository{}
	parsed := r.parsePost(postToFrontmatter(original), "multi")

	assert.Equal(t, original.Description, parsed.Description)
	assert.Equal(t, 1, parsed.Status)
	assert.Equal(t, "content", parsed.Content)
}
