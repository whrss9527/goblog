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
