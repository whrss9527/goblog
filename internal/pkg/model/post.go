package model

import (
	"time"
)

type Post struct {
	Id           string
	Title        string
	Status       int
	CreatedAt    time.Time
	UpdatedAt    time.Time
	CategoryId   int
	IsTop        int
	TagIdString  string
	Views        int
	Description  string
	WordCount    int
	Identity     string
	TagIds       []int
	CategoryName string
	Content      string
	TagNames     []string
}
