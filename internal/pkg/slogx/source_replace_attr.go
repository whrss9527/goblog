package slogx

import (
	"log/slog"
	"path/filepath"
)

type ReplaceAttr func([]string, slog.Attr) slog.Attr

// SourceBase Remove the directory from the source's filename.
func SourceBase(groups []string, a slog.Attr) slog.Attr {
	if a.Key == slog.SourceKey {
		source := a.Value.Any().(*slog.Source)
		source.File = filepath.Base(source.File)
	}
	return a
}
