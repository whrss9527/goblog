package admin

import (
	"log/slog"
	"net/http"
	"strconv"

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

func (h *BookHandler) BookList(ctx *gin.Context) {
	books, err := h.BookRepo.GetBooks()
	if err != nil {
		slog.Error("get books failed", "err", err)
		ctx.Writer.WriteHeader(500)
		return
	}
	data := make(map[string]any)
	data["books"] = books
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "books/list", h.config.App)
}

func (h *BookHandler) BookAdd(ctx *gin.Context) {
	data := make(map[string]any)
	idStr := ctx.Request.FormValue("id")
	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		book, err := h.BookRepo.GetBook(id)
		if err != nil {
			slog.Error("get book failed", "err", err)
			ctx.Writer.WriteHeader(500)
			return
		}
		data["book"] = book
	}
	data["csrf_token"], _ = ctx.Get("csrf_token")
	view.AdminRender(data, ctx.Writer, "books/add", h.config.App)
}

func (h *BookHandler) BookSave(ctx *gin.Context) {
	var book model.Book
	book.Id, _ = strconv.Atoi(ctx.Request.FormValue("id"))
	book.Title = ctx.Request.FormValue("title")
	book.Author = ctx.Request.FormValue("author")
	book.Cover = ctx.Request.FormValue("cover")
	book.ISBN = ctx.Request.FormValue("isbn")
	book.Status, _ = strconv.Atoi(ctx.Request.FormValue("status"))
	book.Progress, _ = strconv.Atoi(ctx.Request.FormValue("progress"))
	book.Rating, _ = strconv.Atoi(ctx.Request.FormValue("rating"))
	book.Comment = ctx.Request.FormValue("comment")
	book.StartDate = ctx.Request.FormValue("start_date")
	book.FinishDate = ctx.Request.FormValue("finish_date")
	book.Year, _ = strconv.Atoi(ctx.Request.FormValue("year"))

	if book.Cover == "" && book.ISBN != "" {
		book.Cover = "https://covers.openlibrary.org/b/isbn/" + book.ISBN + "-M.jpg"
	}

	_, err := h.BookRepo.BookSave(book)
	if err != nil {
		slog.Error("save book failed", "err", err)
		data := make(map[string]any)
		data["msg"] = "保存失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/books", http.StatusFound)
}

func (h *BookHandler) BookDelete(ctx *gin.Context) {
	id, _ := strconv.Atoi(ctx.Param("id"))
	if err := h.BookRepo.BookDelete(id); err != nil {
		slog.Error("delete book failed", "err", err)
		data := make(map[string]any)
		data["msg"] = "删除失败，请重试"
		view.AdminRender(data, ctx.Writer, "401", h.config.App)
		return
	}
	http.Redirect(ctx.Writer, ctx.Request, "/admin/books", http.StatusFound)
}
