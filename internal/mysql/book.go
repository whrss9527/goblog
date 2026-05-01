package mysql

import (
	"goblog/internal/pkg/model"
)

func (r *GormRepository) GetBooks() ([]model.Book, error) {
	var books []model.Book
	err := r.db.Order("year DESC, status ASC, updated_at DESC").Find(&books).Error
	return books, err
}

func (r *GormRepository) GetBooksByYear(year int) ([]model.Book, error) {
	var books []model.Book
	err := r.db.Where("year = ?", year).Order("status ASC, updated_at DESC").Find(&books).Error
	return books, err
}

func (r *GormRepository) GetBookYears() ([]int, error) {
	var years []int
	err := r.db.Model(&model.Book{}).Distinct("year").Order("year DESC").Pluck("year", &years).Error
	return years, err
}

func (r *GormRepository) GetBook(id int) (model.Book, error) {
	var book model.Book
	err := r.db.First(&book, id).Error
	return book, err
}

func (r *GormRepository) BookSave(book model.Book) (int, error) {
	if book.Id > 0 {
		err := r.db.Model(&model.Book{}).Where("id = ?", book.Id).Updates(map[string]any{
			"title":       book.Title,
			"author":      book.Author,
			"cover":       book.Cover,
			"isbn":        book.ISBN,
			"status":      book.Status,
			"progress":    book.Progress,
			"rating":      book.Rating,
			"comment":     book.Comment,
			"start_date":  book.StartDate,
			"finish_date": book.FinishDate,
			"year":        book.Year,
		}).Error
		return book.Id, err
	}
	err := r.db.Create(&book).Error
	return book.Id, err
}

func (r *GormRepository) BookDelete(id int) error {
	return r.db.Delete(&model.Book{}, id).Error
}
