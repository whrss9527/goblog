package model

import "time"

type Book struct {
	Id         int       `json:"id"`
	Title      string    `json:"title"`
	Author     string    `json:"author"`
	Cover      string    `json:"cover"`
	ISBN       string    `json:"isbn"`
	Status     int       `json:"status"`
	Progress   int       `json:"progress"`
	Rating     int       `json:"rating"`
	Comment    string    `json:"comment"`
	StartDate  string    `json:"start_date"`
	FinishDate string    `json:"finish_date"`
	Year       int       `json:"year"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const (
	BookStatusReading  = 1
	BookStatusFinished = 2
	BookStatusWish     = 3
	BookStatusDropped  = 4
)
