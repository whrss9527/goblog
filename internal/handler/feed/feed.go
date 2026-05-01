package feed

import (
	"sort"
	"time"

	"github.com/gorilla/feeds"
)

type Article struct {
	Title     string
	Link      string
	Id        string
	Published time.Time
	Created   time.Time
	Updated   time.Time
	Content   string
	Summary   string
}

func GenerateFeed(articles []Article) (string, error) {
	// 对文章按发布日期排序
	sort.Slice(articles, func(i, j int) bool {
		return articles[i].Published.After(articles[j].Published)
	})

	// 获取当前时间
	now := time.Now()

	// 初始化 feed
	feed := &feeds.Feed{
		Title:       "了迹奇有没",
		Link:        &feeds.Link{Href: "https://whrss.com/"},
		Description: "",
		Author:      &feeds.Author{Name: "whrss"},
		Created:     now,
	}

	// 添加文章到 feed
	feed.Items = make([]*feeds.Item, len(articles))
	for i, article := range articles {
		feed.Items[i] = &feeds.Item{
			Title:       article.Title,
			Link:        &feeds.Link{Href: article.Link},
			Id:          article.Id,
			Created:     article.Created,
			Updated:     article.Updated,
			Content:     article.Content,
			Description: article.Summary,
		}
	}

	// 生成 Atom 格式的 feed
	return feed.ToAtom()
}
