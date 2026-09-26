package model

import "time"

// Project is something the author built: one entry of projects.json in the
// content repository, shown on /projects. Every field is always written (empty
// lists as []) so the file documents its own format for hand edits.
type Project struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	// Description is a sentence or two of plain text.
	Description string `json:"description"`
	// Highlights are optional one-line points worth knowing ("works offline").
	Highlights []string `json:"highlights"`
	// Repo is where the source lives (GitHub or elsewhere); URL is the
	// project's own site, demo or download page.
	Repo string   `json:"repo"`
	URL  string   `json:"url"`
	Tech []string `json:"tech"`
	// Status is one of the ProjectStatus constants.
	Status int `json:"status"`
	// Post is the slug of the article that introduces the project.
	Post string `json:"post"`
	// Cover is an optional screenshot or logo (absolute URL or /covers/…).
	Cover string `json:"cover"`
	// Featured projects come first on /projects and in the home sidebar.
	Featured bool `json:"featured"`
	// Started is the month work began, formatted "2006-01".
	Started string `json:"started"`
	// The timestamps are left out of entries added by hand without them.
	CreatedAt time.Time `json:"created_at,omitzero"`
	UpdatedAt time.Time `json:"updated_at,omitzero"`
}

const (
	ProjectStatusActive   = 1 // 进行中
	ProjectStatusDone     = 2 // 已完成
	ProjectStatusArchived = 3 // 已归档
)

// ProjectStatusLabel is the name of a project status as the site shows it.
func ProjectStatusLabel(status int) string {
	switch status {
	case ProjectStatusDone:
		return "已完成"
	case ProjectStatusArchived:
		return "已归档"
	}
	return "进行中"
}
