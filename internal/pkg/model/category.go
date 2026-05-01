package model

type Category struct {
	Id        int
	Name      string
	CreatedAt string
	UpdatedAt string
	Cur       int `gorm:"-"`
}

func (Category) TableName() string {
	return "categories"
}
