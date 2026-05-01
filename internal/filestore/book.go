package filestore

import (
	"fmt"
	"sort"
	"time"

	"goblog/internal/pkg/model"
)

func (r *FileRepository) GetBooks() ([]model.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Book, len(r.books))
	copy(result, r.books)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Year != result[j].Year {
			return result[i].Year > result[j].Year
		}
		if result[i].Status != result[j].Status {
			return result[i].Status < result[j].Status
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (r *FileRepository) GetBooksByYear(year int) ([]model.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []model.Book
	for _, b := range r.books {
		if b.Year == year {
			result = append(result, b)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Status != result[j].Status {
			return result[i].Status < result[j].Status
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, nil
}

func (r *FileRepository) GetBookYears() ([]int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	yearSet := make(map[int]bool)
	for _, b := range r.books {
		yearSet[b.Year] = true
	}
	var years []int
	for y := range yearSet {
		years = append(years, y)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(years)))
	return years, nil
}

func (r *FileRepository) GetBook(id int) (model.Book, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, b := range r.books {
		if b.Id == id {
			return b, nil
		}
	}
	return model.Book{}, fmt.Errorf("book not found: %d", id)
}

func (r *FileRepository) BookSave(book model.Book) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()

	for i, b := range r.books {
		if b.Id == book.Id {
			book.CreatedAt = b.CreatedAt
			book.UpdatedAt = now
			r.books[i] = book
			if err := r.saveJSON("books.json", r.books); err != nil {
				return 0, err
			}
			r.gitCommitAndPush(fmt.Sprintf("update book: %s", book.Title))
			return book.Id, nil
		}
	}

	book.Id = r.nextBookId
	r.nextBookId++
	book.CreatedAt = now
	book.UpdatedAt = now
	r.books = append(r.books, book)
	if err := r.saveJSON("books.json", r.books); err != nil {
		return 0, err
	}
	r.gitCommitAndPush(fmt.Sprintf("add book: %s", book.Title))
	return book.Id, nil
}

func (r *FileRepository) BookDelete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, b := range r.books {
		if b.Id == id {
			r.books = append(r.books[:i], r.books[i+1:]...)
			if err := r.saveJSON("books.json", r.books); err != nil {
				return err
			}
			r.gitCommitAndPush(fmt.Sprintf("delete book: %s", b.Title))
			return nil
		}
	}
	return fmt.Errorf("book not found: %d", id)
}
