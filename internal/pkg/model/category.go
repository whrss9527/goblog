package model

// Category is stored in categories.json (see Tag for why the JSON names matter).
type Category struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	// Cur is the category of the post being edited; only the admin form uses it.
	Cur int `json:"-"`
}
