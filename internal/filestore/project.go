package filestore

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"goblog/internal/pkg/model"
)

const projectsFile = "projects.json"

var (
	// ErrProjectNotFound is returned for a project that does not exist (any more).
	ErrProjectNotFound = errors.New("project not found")
	// ErrProjectNameEmpty is returned when a project would be saved without a name.
	ErrProjectNameEmpty = errors.New("project name is empty")
)

func (r *FileRepository) loadProjects() error {
	r.projects = nil
	data, err := os.ReadFile(filepath.Join(r.dataDir, projectsFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &r.projects); err != nil {
		return err
	}
	for i, p := range r.projects {
		// entries edited by hand may lack fields or carry stray spaces
		r.projects[i] = normalizeProject(p)
		if p.Id >= r.nextProjectId {
			r.nextProjectId = p.Id + 1
		}
	}
	return nil
}

// GetProjects returns copies of all projects in display order.
func (r *FileRepository) GetProjects() ([]model.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]model.Project, len(r.projects))
	for i, p := range r.projects {
		result[i] = cloneProject(p)
	}
	sortProjects(result)
	return result, nil
}

func (r *FileRepository) GetProject(id int) (model.Project, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, p := range r.projects {
		if p.Id == id {
			return cloneProject(p), nil
		}
	}
	return model.Project{}, ErrProjectNotFound
}

// ProjectSave creates a project (Id 0, or an id that no longer exists) or
// replaces an existing one. The file is written before memory changes, so a
// failed write leaves both as they were.
func (r *FileRepository) ProjectSave(project model.Project) (int, error) {
	project = normalizeProject(project)
	if project.Name == "" {
		return 0, ErrProjectNameEmpty
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().Truncate(time.Second)
	updated := make([]model.Project, len(r.projects), len(r.projects)+1)
	copy(updated, r.projects)
	action := "add"
	index := slices.IndexFunc(updated, func(p model.Project) bool { return project.Id > 0 && p.Id == project.Id })
	if index >= 0 {
		action = "update"
		project.CreatedAt = updated[index].CreatedAt
		project.UpdatedAt = now
		updated[index] = project
	} else {
		if r.nextProjectId < 1 {
			r.nextProjectId = 1
		}
		project.Id = r.nextProjectId
		project.CreatedAt = now
		project.UpdatedAt = now
		updated = append(updated, project)
	}

	if err := r.saveProjects(updated); err != nil {
		return 0, err
	}
	if action == "add" {
		r.nextProjectId++
	}
	r.projects = updated
	r.gitCommitAndPushPath(projectsFile, fmt.Sprintf("%s project: %s", action, project.Name))
	return project.Id, nil
}

func (r *FileRepository) ProjectDelete(id int) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	index := slices.IndexFunc(r.projects, func(p model.Project) bool { return p.Id == id })
	if index < 0 {
		return ErrProjectNotFound
	}
	name := r.projects[index].Name
	updated := slices.Delete(slices.Clone(r.projects), index, index+1)
	if err := r.saveProjects(updated); err != nil {
		return err
	}
	r.projects = updated
	r.gitCommitAndPushPath(projectsFile, fmt.Sprintf("delete project: %s", name))
	return nil
}

// saveProjects writes projects.json. Unlike the other JSON files it keeps "&",
// "<" and ">" as they are (descriptions are prose, and the file is meant to be
// readable on GitHub and editable by hand) and ends with a newline.
func (r *FileRepository) saveProjects(projects []model.Project) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(projects); err != nil {
		return err
	}
	return r.writeFileAtomic(projectsFile, buf.Bytes())
}

// sortProjects orders projects for display: featured ones first, archived ones
// last, and within that the most recently started first.
func sortProjects(projects []model.Project) {
	sort.SliceStable(projects, func(i, j int) bool {
		a, b := projects[i], projects[j]
		if a.Featured != b.Featured {
			return a.Featured
		}
		if archivedA, archivedB := a.Status == model.ProjectStatusArchived, b.Status == model.ProjectStatusArchived; archivedA != archivedB {
			return archivedB
		}
		if a.Started != b.Started {
			return a.Started > b.Started // "2006-01" sorts as text; "" (unknown) last
		}
		return a.Id > b.Id
	})
}

// normalizeProject trims every field and drops empty or repeated list entries,
// so the file only ever holds clean values.
func normalizeProject(p model.Project) model.Project {
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)
	p.Repo = strings.TrimSpace(p.Repo)
	p.URL = strings.TrimSpace(p.URL)
	p.Post = strings.TrimSpace(p.Post)
	p.Cover = strings.TrimSpace(p.Cover)
	p.Started = strings.TrimSpace(p.Started)
	if p.Status < model.ProjectStatusActive || p.Status > model.ProjectStatusArchived {
		p.Status = model.ProjectStatusActive
	}

	highlights := make([]string, 0, len(p.Highlights))
	for _, h := range p.Highlights {
		if h = strings.TrimSpace(h); h != "" {
			highlights = append(highlights, h)
		}
	}
	p.Highlights = highlights

	tech := make([]string, 0, len(p.Tech))
	seen := make(map[string]bool, len(p.Tech))
	for _, t := range p.Tech {
		t = strings.TrimSpace(t)
		if key := strings.ToLower(t); t != "" && !seen[key] {
			seen[key] = true
			tech = append(tech, t)
		}
	}
	p.Tech = tech
	return p
}

func cloneProject(p model.Project) model.Project {
	p.Highlights = slices.Clone(p.Highlights)
	p.Tech = slices.Clone(p.Tech)
	return p
}
