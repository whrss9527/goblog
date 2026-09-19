package front

import "testing"

func TestGetPageUrl(t *testing.T) {
	h := &PostHandler{}
	tests := []struct {
		name                     string
		categoryId, tagId, query string
		page                     int
		want                     string
	}{
		{name: "first page without filters is the bare root", page: 1, want: "/"},
		{name: "page only", page: 3, want: "/?page=3"},
		{name: "category is kept", categoryId: "2", page: 2, want: "/?category_id=2&page=2"},
		{name: "tag is kept", tagId: "61", page: 1, want: "/?tag_id=61"},
		{name: "keyword survives pagination and is escaped", query: "go 优雅", page: 2, want: "/?keyword=go+%E4%BC%98%E9%9B%85&page=2"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := h.getPageUrl(tt.categoryId, tt.tagId, tt.query, tt.page); got != tt.want {
				t.Errorf("getPageUrl() = %q, want %q", got, tt.want)
			}
		})
	}
}
