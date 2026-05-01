package model

type Page struct {
	Id      string `json:"page_id" gorm:"column:ident;"`
	Title   string `json:"title"  gorm:"column:title;"`
	Content string `json:"content"  gorm:"column:content;"`
}

func (Page) TableName() string {
	return "pages"
}
