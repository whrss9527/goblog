package model

// 假设 PostContent 结构体定义如下
type PostContent struct {
	PostId  string `gorm:"primaryKey;column:post_id;"`
	Content string `gorm:"column:content;"`
}

func (PostContent) TableName() string {
	return "post_contents"
}
