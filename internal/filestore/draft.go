package filestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"goblog/internal/pkg/model"
	"goblog/pkg/utils"
)

// Drafts are unpublished posts. They live in <data_dir>/.drafts and, unlike
// everything else in the data directory, never enter the content repository:
// that repository is usually public, and an unfinished text should not be.
// The price is that drafts exist on this server only.
const draftsDir = ".drafts"

// ErrDraftNotFound is returned for a draft that does not exist (any more).
var ErrDraftNotFound = errors.New("draft not found")

func (r *FileRepository) draftPath(slug string) string {
	return filepath.Join(r.dataDir, draftsDir, slug+".md")
}

// ensureDraftsPrivate tells git to ignore the drafts through .git/info/exclude,
// which is local to this clone: nothing in the repository itself changes. It
// fails when that cannot be guaranteed, and no draft is written then.
func (r *FileRepository) ensureDraftsPrivate() error {
	if !r.gitEnabled {
		return nil
	}
	exclude := filepath.Join(r.dataDir, ".git", "info", "exclude")
	existing, err := os.ReadFile(exclude)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read git exclude file: %w", err)
	}
	for _, line := range strings.Split(string(existing), "\n") {
		if strings.TrimSpace(line) == draftsDir+"/" {
			return nil
		}
	}
	if err := os.MkdirAll(filepath.Dir(exclude), 0o755); err != nil {
		return fmt.Errorf("create git info dir: %w", err)
	}
	addition := "# goblog: drafts stay on this server\n" + draftsDir + "/\n"
	if len(existing) > 0 && !strings.HasSuffix(string(existing), "\n") {
		addition = "\n" + addition
	}
	f, err := os.OpenFile(exclude, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open git exclude file: %w", err)
	}
	defer f.Close()
	if _, err := f.WriteString(addition); err != nil {
		return fmt.Errorf("write git exclude file: %w", err)
	}
	return nil
}

// GetDrafts lists all drafts, most recently saved first.
func (r *FileRepository) GetDrafts() ([]model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	entries, err := os.ReadDir(filepath.Join(r.dataDir, draftsDir))
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var drafts []model.Post
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".md") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(r.dataDir, draftsDir, name))
		if err != nil {
			continue
		}
		drafts = append(drafts, *r.parsePost(string(raw), strings.TrimSuffix(name, ".md")))
	}
	sort.Slice(drafts, func(i, j int) bool { return drafts[i].UpdatedAt.After(drafts[j].UpdatedAt) })
	return drafts, nil
}

// GetDraft returns one draft.
func (r *FileRepository) GetDraft(slug string) (model.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !safeSlug(slug) {
		return model.Post{}, ErrDraftNotFound
	}
	raw, err := os.ReadFile(r.draftPath(slug))
	if err != nil {
		return model.Post{}, ErrDraftNotFound
	}
	return *r.parsePost(string(raw), slug), nil
}

// SaveDraft writes a draft. previousSlug is the slug it was loaded under ("" for
// a new one); a different post.Identity renames it. A draft may not take the
// address of a published post or of another draft.
func (r *FileRepository) SaveDraft(post model.Post, previousSlug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !safeSlug(post.Identity) || (previousSlug != "" && !safeSlug(previousSlug)) {
		return ErrInvalidSlug
	}
	if _, published := r.postBySlug[post.Identity]; published {
		return ErrSlugTaken
	}
	if previousSlug != post.Identity {
		if _, err := os.Stat(r.draftPath(post.Identity)); err == nil {
			return ErrSlugTaken
		}
	}
	if err := r.ensureDraftsPrivate(); err != nil {
		return err
	}

	now := time.Now()
	post.Id = post.Identity
	post.Status = 0
	post.UpdatedAt = now
	post.CreatedAt = now
	if previousSlug != "" {
		if raw, err := os.ReadFile(r.draftPath(previousSlug)); err == nil {
			if old := r.parsePost(string(raw), previousSlug); !old.CreatedAt.IsZero() {
				post.CreatedAt = old.CreatedAt
			}
		}
	}
	post.WordCount = utils.GetTotalWords(post.Content)

	if err := os.MkdirAll(filepath.Join(r.dataDir, draftsDir), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(r.draftPath(post.Identity), []byte(draftToFrontmatter(&post)), 0o644); err != nil {
		return err
	}
	if previousSlug != "" && previousSlug != post.Identity {
		os.Remove(r.draftPath(previousSlug))
	}
	return nil
}

// DeleteDraft removes a draft; a draft that is already gone is not an error.
func (r *FileRepository) DeleteDraft(slug string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !safeSlug(slug) {
		return ErrInvalidSlug
	}
	if err := os.Remove(r.draftPath(slug)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// draftToFrontmatter is postToFrontmatter plus the tag names as typed: tags
// that do not exist yet are only created (in tags.json, which is public) when
// the draft is published.
func draftToFrontmatter(post *model.Post) string {
	out := postToFrontmatter(post)
	if len(post.TagNames) == 0 {
		return out
	}
	names := make([]string, 0, len(post.TagNames))
	for _, name := range post.TagNames {
		if name = strings.TrimSpace(strings.ReplaceAll(name, ",", " ")); name != "" {
			names = append(names, name)
		}
	}
	line := fmt.Sprintf("tag_names: \"%s\"\n", escapeQuoted(strings.Join(names, ",")))
	// insert before the closing delimiter of the frontmatter block
	const closing = "---\n\n"
	head := len("---\n")
	if i := strings.Index(out[head:], closing); i >= 0 {
		return out[:head+i] + line + out[head+i:]
	}
	return out
}
