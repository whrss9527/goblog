package model

// Tag is stored in tags.json. The JSON names match the files written by the
// database migration; without them Go would drop created_at / updated_at on
// load and rewrite the whole file with "Id", "Name"… on the first save.
type Tag struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Count     int    `json:"count"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}
