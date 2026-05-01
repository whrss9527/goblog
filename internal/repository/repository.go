package repository

import (
	"goblog/internal/pkg/model"
)

type PostParams struct {
	Ids        map[string][]string
	CategoryId string
	TagId      string
	PerPage    int
	Page       int
	Keyword    string
}

type PageParams struct {
	PerPage int
	Page    int
}

type PostRepository interface {
	GetPosts(params PostParams) ([]*model.Post, int64, error)
	GetPostsWithContent() ([]*model.Post, error)
	GetPostsArchive() ([]*model.Post, error)
	GetPost(id string) (model.Post, error)
	GetPostByIdentity(identity string) (model.Post, error)
	IncrView(id string) error
	PostDelete(post model.Post) (string, error)
	PostSave(post model.Post) (string, error)
	GetPostCountByTagId(id string) (int, error)
	GetPostIdsByContent(content string) ([]string, error)
	PostExist(id string) (bool, error)
}

type CategoryRepository interface {
	GetCategories() ([]model.Category, error)
	GetCategory(id int) (model.Category, error)
	GetCategoryIdsByName(name string) ([]string, error)
	CategoryDelete(category model.Category) (int, error)
	CategorySave(category model.Category) (int, error)
}

type TagRepository interface {
	GetTags() ([]model.Tag, error)
	GetTagsByIds(ids []int) ([]model.Tag, error)
	GetTagIdsByName(name string) ([]string, error)
	AddTag(tag model.Tag) (int, error)
	IncrTagCount(id string) error
}

type PageRepository interface {
	GetPages(params PageParams) ([]model.Page, error)
	GetPage(ident string) (model.Page, error)
	PageDelete(page model.Page) (string, error)
	PageSave(page model.Page) (string, error)
	PageExist(id string) (bool, error)
}

type UserRepository interface {
	GetUserByEmail(email string) (model.User, error)
	AddUser(user model.User) (uint, error)
	UpdateUser(user model.User) error
	DeleteUserByEmail(email string) error
	GetAllUsers() ([]model.User, error)
}

type BookRepository interface {
	GetBooks() ([]model.Book, error)
	GetBooksByYear(year int) ([]model.Book, error)
	GetBookYears() ([]int, error)
	GetBook(id int) (model.Book, error)
	BookSave(book model.Book) (int, error)
	BookDelete(id int) error
}
