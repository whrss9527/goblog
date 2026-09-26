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
	IncrLike(id string) (int, error)
	PostDelete(post model.Post) (string, error)
	PostSave(post model.Post) (string, error)
	GetPostCountByTagId(id string) (int, error)
	GetPostIdsByContent(content string) ([]string, error)
	PostExist(id string) (bool, error)
}

// DraftRepository stores unpublished posts. Drafts are private to the server
// they were written on: they are never committed to the content repository.
type DraftRepository interface {
	GetDrafts() ([]model.Post, error)
	GetDraft(slug string) (model.Post, error)
	SaveDraft(post model.Post, previousSlug string) error
	DeleteDraft(slug string) error
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
	// RenameTag renames a tag; if the name belongs to another tag already, the two are merged.
	// It returns the id the posts are filed under afterwards.
	RenameTag(id int, name string) (int, error)
	// DeleteTag removes a tag and takes it off every post.
	DeleteTag(id int) error
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

// ProjectRepository stores the projects shown on /projects (projects.json).
type ProjectRepository interface {
	// GetProjects returns every project in display order: featured first, archived last, newest first.
	GetProjects() ([]model.Project, error)
	GetProject(id int) (model.Project, error)
	// ProjectSave creates the project (Id 0 or unknown) or replaces it, and returns its id.
	ProjectSave(project model.Project) (int, error)
	ProjectDelete(id int) error
}
