package admin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSplitLines(t *testing.T) {
	assert.Equal(t, []string{"一条命令切换", "支持规则", "*nix 也能用", "-1 的延迟", "粘贴来的列表"},
		splitLines("- 一条命令切换\r\n\n  • 支持规则 \n*nix 也能用\n-1 的延迟\n·粘贴来的列表\n   \n"))
	assert.Nil(t, splitLines(" \n\n"))
}

func TestRepoKey(t *testing.T) {
	assert.Equal(t, repoKey("https://github.com/whrss9527/GoBlog.git"), repoKey("whrss9527/goblog"))
	assert.Equal(t, repoKey("https://gitee.com/a/b/"), repoKey("https://GITEE.com/a/b"))
	assert.NotEqual(t, repoKey("https://gitee.com/a/b"), repoKey("https://github.com/a/b"))
}

func TestIsWebAddress(t *testing.T) {
	assert.True(t, isWebAddress("https://whrss.com"))
	assert.True(t, isWebAddress("http://localhost:9091/x"))
	assert.False(t, isWebAddress("whrss.com"))
	assert.False(t, isWebAddress("javascript:alert(1)"))
	assert.False(t, isWebAddress("https://"))
}
