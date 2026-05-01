package model

type User struct {
	Id       uint   `gorm:"primaryKey,column:id;"`
	Email    string `gorm:"column:email;"`
	Password string `gorm:"column:password;"`
}

func (User) TableName() string {
	return "users"
}
