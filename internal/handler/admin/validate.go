package admin

import (
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"goblog/internal/filestore"
)

// slugPattern is what a new address may look like: it ends up in URLs
// (/posts/<slug>, /pages/<id>) and as a file name in the content repository.
var slugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]*$`)

const maxSlugLength = 100

// checkSlug validates a new or changed slug and returns a message for the
// author ("" when it is fine).
func checkSlug(slug string) string {
	switch {
	case slug == "":
		return "请填写地址（英文标题），它会成为文章地址的最后一段"
	case utf8.RuneCountInString(slug) > maxSlugLength:
		return "地址太长了，最多 100 个字符"
	case !slugPattern.MatchString(slug) || strings.Contains(slug, ".."):
		return "地址只能用小写字母、数字和 - _ . ，并以字母或数字开头"
	}
	return ""
}

const maxLabelLength = 40

// checkLabel validates the name of a tag or category (what: "标签名" / "分类名") and returns a message
// for the author ("" when it is fine).
func checkLabel(name, what string) string {
	switch {
	case name == "":
		return what + "不能为空"
	case utf8.RuneCountInString(name) > maxLabelLength:
		return what + "太长了，最多 40 个字"
	case strings.ContainsAny(name, ",，"):
		return what + "里不能有逗号：写文章时逗号用来分隔多个标签"
	case strings.ContainsFunc(name, unicode.IsControl):
		return what + "里不能有换行或控制字符"
	}
	return ""
}

// saveErrorMessage turns a repository error into something the author can act on.
func saveErrorMessage(err error) string {
	switch {
	case errors.Is(err, filestore.ErrSlugTaken):
		return "这个地址已经被另一篇文章占用了，换一个吧"
	case errors.Is(err, filestore.ErrInvalidSlug):
		return "这个地址不能用作文件名，请换一个"
	}
	return "写入失败，请稍后重试（内容仍在下面的编辑器里，也已在本机留了草稿）"
}

// adminZone is the time zone dates are shown in (the same one the site uses).
var adminZone = func() *time.Location {
	if loc, err := time.LoadLocation("Asia/Shanghai"); err == nil {
		return loc
	}
	return time.FixedZone("CST", 8*3600)
}()

// toJSON serialises a small list for a data- attribute; html/template escapes it.
func toJSON(list []string) string {
	if list == nil {
		list = []string{}
	}
	raw, err := json.Marshal(list)
	if err != nil {
		return "[]"
	}
	return string(raw)
}
