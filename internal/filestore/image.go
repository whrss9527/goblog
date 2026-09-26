package filestore

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// imagesDir holds the images uploaded in the editor (when they are not sent
// to S3-compatible storage). It is part of the content repository, so images
// travel with the posts that use them.
const imagesDir = "images"

var (
	// ErrImageExists is returned when an upload would replace an existing image.
	ErrImageExists = errors.New("an image with this name exists")
	imageName      = regexp.MustCompile(`^[0-9]{4}/[0-9]{2}/[0-9a-z-]+\.(png|jpg|gif|webp)$`)
)

// SaveImage stores an uploaded image as images/<name> ("2026/09/1758854400123.png"),
// commits and pushes it. Images are never replaced.
func (r *FileRepository) SaveImage(name string, data []byte) error {
	if !imageName.MatchString(name) {
		return fmt.Errorf("invalid image name %q", name)
	}
	rel := imagesDir + "/" + name
	target := filepath.Join(r.dataDir, filepath.FromSlash(rel))
	// not while a sync rebases the working tree (images do not touch the
	// content in memory, so they are accepted even while it is stale)
	r.wmu.Lock()
	defer r.wmu.Unlock()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(target); err == nil {
		return ErrImageExists
	}
	if err := r.writeFileAtomic(filepath.FromSlash(rel), data); err != nil {
		return err
	}
	r.gitCommitAndPushPath(rel, "add image: "+name)
	return nil
}

// ImagesDir is where uploaded images are stored on disk.
func (r *FileRepository) ImagesDir() string {
	return filepath.Join(r.dataDir, imagesDir)
}
