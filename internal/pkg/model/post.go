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
	Likes        int
	Description  string
	WordCount    int
	Identity     string
	TagIds       []int
	CategoryName string
	Content      string
	TagNames     []string
	// Tags is filled by handlers for display; it is not persisted.
	Tags []Tag
}
