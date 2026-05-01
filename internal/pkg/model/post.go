package model

import (
	"time"
)

// Post 文章模型，对应数据库中的 posts 表
type Post struct {
	Id          string    `gorm:"primaryKey;column:id;"`
	Title       string    `gorm:"column:title;"`
	Status      int       `gorm:"column:status;"`
	CreatedAt   time.Time `gorm:"column:created_at;"`
	UpdatedAt   time.Time `gorm:"column:updated_at;"`
	CategoryId  int       `gorm:"column:category_id;"`
	IsTop       int       `gorm:"column:is_top;"`
	TagIdString string    `gorm:"column:tag_ids;"`
	Views       int       `gorm:"column:views;"`
	Description string    `gorm:"column:description;"`
	WordCount   int       `gorm:"column:word_count;"`
	Identity    string    `gorm:"column:identity;"`
	// 数据库表中不存在以下字段，不做数据库映射
	TagIds       []int    `gorm:"-"`
	CategoryName string   `gorm:"-"`
	Content      string   `gorm:"-"`
	TagNames     []string `gorm:"-"`
}

// TableName 指定 Post 结构体对应的数据库表名
func (Post) TableName() string {
	return "posts"
}
