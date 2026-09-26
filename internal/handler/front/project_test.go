package front

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"goblog/internal/pkg/model"
)

func TestProjectAnchor(t *testing.T) {
	tests := map[string]string{
		"goblog":              "project-goblog",
		"ProxySwitch for Mac": "project-proxyswitch-for-mac",
		"  A -- B  ":          "project-a-b",
		"软考 模拟器":              "project-软考-模拟器",
		"go-alipay-sdk-v3":    "project-go-alipay-sdk-v3",
		"!!!":                 "project-7",
	}
	for name, want := range tests {
		assert.Equal(t, want, projectAnchor(model.Project{Id: 7, Name: name}), name)
	}
}

func TestAgoAndMonthLabels(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	assert.Equal(t, "", agoLabel(time.Time{}, now))
	assert.Equal(t, "今天", agoLabel(now.Add(-3*time.Hour), now))
	assert.Equal(t, "昨天", agoLabel(now.Add(-30*time.Hour), now))
	assert.Equal(t, "5 天前", agoLabel(now.AddDate(0, 0, -5), now))
	assert.Equal(t, "3 个月前", agoLabel(now.AddDate(0, -3, 0), now))
	assert.Equal(t, "2 年前", agoLabel(now.AddDate(-2, 0, -1), now))

	assert.Equal(t, "2026 年 9 月", monthLabel("2026-09"))
	assert.Equal(t, "", monthLabel("2026-9"))
	assert.Equal(t, "", monthLabel(""))
}

func TestRepoAddress(t *testing.T) {
	assert.Equal(t, "https://github.com/whrss9527/goblog", repoAddress("https://github.com/whrss9527/goblog"))
	assert.Equal(t, "https://github.com/whrss9527/goblog", repoAddress("whrss9527/goblog"), "GitHub shorthand from a hand-edited file")
	assert.Equal(t, "https://gitee.com/a/b", repoAddress("https://gitee.com/a/b"))
	assert.Equal(t, "", repoAddress("javascript:alert(1)"))
	assert.Equal(t, "", repoAddress("ftp://example.com/x"))
	assert.Equal(t, "", repoAddress("just words"))
	assert.Equal(t, "https://example.com", webAddress("https://example.com"))
	assert.Equal(t, "", webAddress("javascript:alert(1)"))
	assert.Equal(t, "", webAddress("//example.com/a.png"))

	assert.Equal(t, "GitHub", repoLabel("https://www.github.com/a/b"))
	assert.Equal(t, "Gitee", repoLabel("https://gitee.com/a/b"))
	assert.Equal(t, "源码", repoLabel("https://git.example.com/a/b"))
	assert.Equal(t, "", repoLabel(""))
}

func TestTechLabels(t *testing.T) {
	labels := techLabels("Go", []string{"Gin", "PWA"})
	assert.Equal(t, []TechLabel{{"Go", "#00add8"}, {"Gin", ""}, {"PWA", ""}}, labels, "the repository language comes first")

	labels = techLabels("Go", []string{"SwiftUI", "go"})
	assert.Equal(t, []TechLabel{{"SwiftUI", ""}, {"go", "#00add8"}}, labels, "a language the author listed is not repeated")

	assert.Empty(t, techLabels("", nil))
	assert.Equal(t, TechLabel{}, ProjectCard{}.PrimaryTech())
}

func TestCountProjectsAndHighlights(t *testing.T) {
	cards := []ProjectCard{
		{Project: model.Project{Name: "a", Status: model.ProjectStatusActive, Featured: true}, Stars: 3},
		{Project: model.Project{Name: "b", Status: model.ProjectStatusArchived}, Stars: 10},
		{Project: model.Project{Name: "c", Status: model.ProjectStatusDone}},
		{Project: model.Project{Name: "d", Status: model.ProjectStatusActive}},
	}
	counts := countProjects(cards)
	assert.Equal(t, ProjectCounts{Total: 4, Active: 2, Done: 1, Archived: 1, Stars: 13}, counts)
	assert.Equal(t, 3, counts.Statuses())
	assert.Equal(t, 1, countProjects(cards[:1]).Statuses())

	var names []string
	for _, card := range highlightProjects(cards, 2) {
		names = append(names, card.Name)
	}
	assert.Equal(t, []string{"a", "c"}, names, "archived projects are skipped")
}

func TestSearchProjects(t *testing.T) {
	cards := []ProjectCard{
		{Project: model.Project{Name: "proxyswitch", Description: "切换系统代理", Repo: "https://github.com/whrss9527/proxyswitch"}, Anchor: "project-proxyswitch",
			Stack: []TechLabel{{Name: "Go"}}, StatusLabel: "进行中"},
		{Project: model.Project{Name: "ProxySwitch for Mac", Description: "菜单栏 <切换> 代理", Highlights: []string{"按 Wi-Fi 自动切换"}}, Anchor: "project-proxyswitch-for-mac",
			Stack: []TechLabel{{Name: "Swift"}}, StatusLabel: "进行中"},
		{Project: model.Project{Name: "goblog", Description: "博客程序"}, Anchor: "project-goblog", Stack: []TechLabel{{Name: "Go"}}, StatusLabel: "已完成"},
	}

	names := func(items []SearchItem) []string {
		var out []string
		for _, item := range items {
			out = append(out, item.Title)
		}
		return out
	}
	assert.Equal(t, []string{"proxyswitch", "ProxySwitch for Mac"}, names(searchProjects(cards, "proxy", 3)))
	assert.Equal(t, []string{"ProxySwitch for Mac"}, names(searchProjects(cards, "proxy wi-fi", 3)), "every term has to match")
	assert.Equal(t, []string{"goblog", "proxyswitch"}, names(searchProjects(cards, "go", 3)), "a name match outranks a technology match")
	assert.Len(t, searchProjects(cards, "o", 1), 1)
	assert.Empty(t, searchProjects(cards, "   ", 3))
	assert.Empty(t, searchProjects(cards, "rust", 3))

	item := searchProjects(cards, "切换", 3)[0] // description and highlight beat description alone
	assert.Equal(t, "/projects#project-proxyswitch-for-mac", item.URL)
	assert.Equal(t, "菜单栏 &lt;<mark>切换</mark>&gt; 代理", item.SnippetHTML, "escaped, then highlighted")
	assert.Equal(t, "项目 · 进行中 · Swift", item.Meta)
}
