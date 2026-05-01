package model

type Tag struct {
	Id        int
	Name      string
	Count     int
	CreatedAt string
	UpdatedAt string
}

func (Tag) TableName() string {
	return "tags"
}
