package front

import (
	"log/slog"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/pkg/model"
	"goblog/internal/pkg/view"
	"goblog/internal/repository"
)

type BookHandler struct {
	BookRepo repository.BookRepository
	config   *config.Config
}

func NewBookHandler(bookRepo repository.BookRepository, config *config.Config) *BookHandler {
	return &BookHandler{
		BookRepo: bookRepo,
		config:   config,
	}
}

type BookYear struct {
	Year  int
	Books []model.Book
	Stats BookStats
}

type BookStats struct {
	Total    int
	Finished int
	Reading  int
}

func (h *BookHandler) ReadingList(ctx *gin.Context) {
	years, err := h.BookRepo.GetBookYears()
	if err != nil {
		slog.Error("get book years failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}

	var bookYears []BookYear
	var totalAll, finishedAll int
	for _, year := range years {
		books, err := h.BookRepo.GetBooksByYear(year)
		if err != nil {
			slog.Error("get books by year failed", "year", year, "err", err)
			continue
		}
		stats := BookStats{Total: len(books)}
		for _, b := range books {
			if b.Status == model.BookStatusFinished {
				stats.Finished++
			} else if b.Status == model.BookStatusReading {
				stats.Reading++
			}
		}
		totalAll += stats.Total
		finishedAll += stats.Finished
		bookYears = append(bookYears, BookYear{Year: year, Books: books, Stats: stats})
	}

	data := make(map[string]any)
	data["title"] = "阅读清单"
	data["description"] = "我的阅读清单"
	data["book_years"] = bookYears
	data["total_all"] = totalAll
	data["finished_all"] = finishedAll
	view.Render(data, ctx.Writer, "reading", h.config.App)
}
