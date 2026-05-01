package model

import "time"

type Book struct {
	Id         int       `gorm:"primaryKey;autoIncrement" json:"id"`
	Title      string    `gorm:"column:title;size:200" json:"title"`
	Author     string    `gorm:"column:author;size:100" json:"author"`
	Cover      string    `gorm:"column:cover;size:500" json:"cover"`
	ISBN       string    `gorm:"column:isbn;size:20" json:"isbn"`
	Status     int       `gorm:"column:status" json:"status"`
	Progress   int       `gorm:"column:progress" json:"progress"`
	Rating     int       `gorm:"column:rating" json:"rating"`
	Comment    string    `gorm:"column:comment;size:1000" json:"comment"`
	StartDate  string    `gorm:"column:start_date;size:10" json:"start_date"`
	FinishDate string    `gorm:"column:finish_date;size:10" json:"finish_date"`
	Year       int       `gorm:"column:year" json:"year"`
	CreatedAt  time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt  time.Time `gorm:"column:updated_at" json:"updated_at"`
}

func (Book) TableName() string {
	return "books"
}

const (
	BookStatusReading  = 1
	BookStatusFinished = 2
	BookStatusWish     = 3
	BookStatusDropped  = 4
)
