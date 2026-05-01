package util

import (
	"log/slog"
	"os"
	"time"
)

func SplitArray[E any](arr []E, chunkSize int) [][]E {
	var result [][]E
	for i := 0; i < len(arr); i += chunkSize {
		end := i + chunkSize
		if end > len(arr) {
			end = len(arr)
		}
		result = append(result, arr[i:end])
	}
	return result
}

func TimeToUnix(t time.Time) int64 {
	if t.IsZero() {
		return 0
	}
	return t.Unix()
}

func Exit(code int) {
	if code == 0 {
		slog.Info("Exit", "code", code)
	} else {
		slog.Error("Exit", "code", code)
	}
	os.Exit(code)
}
