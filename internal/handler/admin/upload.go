package admin

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"goblog/internal/config"
	"goblog/internal/filestore"
	"goblog/internal/pkg/s3"
)

// imageTypes are the formats the editor may upload, by sniffed content type.
// No SVG: it can carry scripts.
var imageTypes = map[string]string{
	"image/png":  ".png",
	"image/jpeg": ".jpg",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// ImageStore keeps uploaded images in the content repository.
type ImageStore interface {
	SaveImage(name string, data []byte) error
}

// UploadHandler receives the images pasted, dropped or picked in the editor.
type UploadHandler struct {
	Images ImageStore
	bucket *s3.Client // nil: images go to the content repository
	upload *config.UploadConfig
	config *config.Config

	mu       sync.Mutex
	lastBase string // names handed out within the same millisecond get -1, -2…
	repeats  int
	now      func() time.Time
}

func NewUploadHandler(images ImageStore, config *config.Config) *UploadHandler {
	h := &UploadHandler{Images: images, upload: config.App.Upload, config: config, now: time.Now}
	if u := config.App.Upload; u.S3() {
		region := u.S3Region
		if region == "" {
			region = "auto"
		}
		h.bucket = &s3.Client{Endpoint: u.S3Endpoint, Region: region, Bucket: u.S3Bucket, AccessKey: u.S3AccessKey, SecretKey: u.S3SecretKey}
	}
	return h
}

// MaxBytes is the largest upload accepted.
func (h *UploadHandler) MaxBytes() int64 {
	return h.upload.MaxBytes()
}

// Upload handles POST /admin/uploads (multipart, field "file"). The answer
// carries the image's address and the next CSRF token: every POST rotates it,
// and the editor form still has to be saved afterwards.
func (h *UploadHandler) Upload(ctx *gin.Context) {
	token, _ := ctx.Get("csrf_token")
	fail := func(status int, message string) {
		ctx.JSON(status, gin.H{"error": message, "csrf": token})
	}
	max := h.MaxBytes()
	file, _, err := ctx.Request.FormFile("file")
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			fail(http.StatusRequestEntityTooLarge, "图片太大了，最大 "+strconv.FormatInt(max>>20, 10)+" MB")
			return
		}
		fail(http.StatusBadRequest, "没有收到图片")
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, max+1))
	if err != nil {
		fail(http.StatusBadRequest, "读取图片失败")
		return
	}
	if int64(len(data)) > max {
		fail(http.StatusRequestEntityTooLarge, "图片太大了，最大 "+strconv.FormatInt(max>>20, 10)+" MB")
		return
	}
	contentType := http.DetectContentType(data)
	ext, ok := imageTypes[contentType]
	if !ok {
		fail(http.StatusUnsupportedMediaType, "只能上传 PNG、JPEG、GIF 或 WebP 图片")
		return
	}

	name := h.newName(ext)
	var address string
	if h.bucket != nil {
		key := strings.TrimLeft(h.upload.Prefix, "/") + name
		c, cancel := context.WithTimeout(ctx.Request.Context(), time.Minute)
		defer cancel()
		if err := h.bucket.Put(c, key, contentType, data); err != nil {
			slog.Error("upload image to object storage failed", "err", err, "key", key)
			fail(http.StatusBadGateway, "上传到对象存储失败，详情见服务器日志")
			return
		}
		address = strings.TrimRight(h.upload.PublicURL, "/") + "/" + key
	} else {
		if err := h.Images.SaveImage(name, data); err != nil {
			slog.Error("save image failed", "err", err, "name", name)
			if errors.Is(err, filestore.ErrImageExists) {
				fail(http.StatusConflict, "图片重名了，请再试一次")
				return
			}
			fail(http.StatusInternalServerError, "保存图片失败，请稍后重试")
			return
		}
		address = "/images/" + name
	}
	ctx.JSON(http.StatusOK, gin.H{"url": address, "csrf": token})
}

// newName names an upload after the time it arrived, the scheme image hosts
// (and this blog's older images) use: 2026/09/1758854400123.png.
func (h *UploadHandler) newName(ext string) string {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now()
	base := now.Format("2006/01/") + strconv.FormatInt(now.UnixMilli(), 10)
	if base != h.lastBase {
		h.lastBase, h.repeats = base, 0
		return base + ext
	}
	h.repeats++
	return base + "-" + strconv.Itoa(h.repeats) + ext
}
