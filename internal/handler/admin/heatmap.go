package admin

import (
	"encoding/json"
	"log/slog"
	"os"

	"github.com/robfig/cron/v3"

	"goblog/internal/repository"
)

type HeatMapHandler struct {
	PostRepo repository.PostRepository
}

func NewHeatMapHandler(postRepo repository.PostRepository) *HeatMapHandler {
	return &HeatMapHandler{
		PostRepo: postRepo,
	}
}

type HeatMapGenerateJob struct {
	Name     string
	PostRepo repository.PostRepository
}

func (handler *HeatMapHandler) NewJob() *HeatMapGenerateJob {
	return &HeatMapGenerateJob{
		Name:     "heatmap",
		PostRepo: handler.PostRepo,
	}
}

func (handler *HeatMapHandler) RunTask() {
	handler.NewJob().Run()
	go func() {
		c := cron.New(cron.WithSeconds())
		job := handler.NewJob()
		_, err := c.AddJob("0 20 * * * ?", job)
		if err != nil {
			slog.Error("heatmap cron add failed", "err", err)
			return
		}
		c.Start()
		slog.Info("heatmap cron started")
		defer c.Stop()
		select {}
	}()
}

type HeatMapData struct {
	Date    string    `json:"date"`
	Total   int       `json:"total"`
	Details []Details `json:"details"`
	Summary []Summary `json:"summary"`
}

type Summary struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Details struct {
	Name  string `json:"name"`
	Date  string `json:"date"`
	Value int    `json:"value"`
	Link  string `json:"link"`
}

func (job HeatMapGenerateJob) Run() {
	posts, _, err := job.PostRepo.GetPosts(repository.PostParams{
		PerPage: 0,
		Page:    1,
	})
	if err != nil {
		slog.Error("heatmap get posts failed", "err", err)
		return
	}

	heatMaps := make([]HeatMapData, 0)
	summaryMap := make(map[string][]Summary)
	detailMap := make(map[string][]Details)

	for _, post := range posts {
		dateStr := post.CreatedAt.Format("2006-01-02")
		summaryMap[dateStr] = append(summaryMap[dateStr], Summary{
			Name:  post.Title,
			Value: post.CreatedAt.Format("15:04:05"),
		})
		detailMap[dateStr] = append(detailMap[dateStr], Details{
			Value: post.WordCount,
			Name:  post.Title,
			Date:  post.CreatedAt.Format("2006-01-02 15:04:05"),
			Link:  "/posts/" + post.Identity,
		})
	}

	for date, summaries := range summaryMap {
		heatMaps = append(heatMaps, HeatMapData{
			Date:    date,
			Total:   len(summaries),
			Summary: summaries,
			Details: detailMap[date],
		})
	}

	slog.Info("heatmap job finished", "task", job.Name, "entries", len(heatMaps))
	data, err := json.Marshal(heatMaps)
	if err != nil {
		slog.Error("heatmap marshal failed", "err", err)
		return
	}
	if err := os.WriteFile("./heatmap.txt", data, 0644); err != nil {
		slog.Error("heatmap write file failed", "err", err)
	}
}
