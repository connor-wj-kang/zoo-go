package httpapi

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"zoo-go/internal/domain"
)

func (s *Server) communityPosts(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(query(r, "limit", "20"))
	feed, err := s.store.CommunityPosts(currentUser(r).ID, query(r, "cursor", ""), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, feed)
}

func (s *Server) createCommunityPost(w http.ResponseWriter, r *http.Request) {
	var req domain.CreatePostRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	post, err := s.store.CreateCommunityPost(currentUser(r).ID, req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, envelope{Data: post})
}

func (s *Server) upload(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 9<<20)
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, &uploadError{message: "请选择需要上传的图片"})
		return
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 8<<20+1))
	if err != nil {
		writeError(w, err)
		return
	}
	if len(data) > 8<<20 {
		writeError(w, &uploadError{message: "图片不能超过 8MB"})
		return
	}
	mimeType := http.DetectContentType(data)
	extensions := map[string]string{
		"image/jpeg": ".jpg",
		"image/png":  ".png",
		"image/webp": ".webp",
		"image/gif":  ".gif",
	}
	extension, ok := extensions[mimeType]
	if !ok {
		writeError(w, &uploadError{message: "仅支持 JPG、PNG、WebP、GIF 图片"})
		return
	}
	filename := newUploadFilename() + extension
	path := s.uploadPath(filename)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		writeError(w, err)
		return
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		writeError(w, err)
		return
	}
	if err := s.store.RecordUpload(currentUser(r).ID, filename, header.Filename, mimeType, int64(len(data)), s.now()); err != nil {
		_ = os.Remove(path)
		writeError(w, err)
		return
	}
	writeData(w, map[string]any{
		"url":      "/uploads/" + filename,
		"filename": filename,
		"mimeType": mimeType,
		"size":     len(data),
	})
}

type uploadError struct {
	message string
}

func (e *uploadError) Error() string {
	return "UPLOAD_ERROR: " + e.message
}

func newUploadFilename() string {
	raw := make([]byte, 12)
	if _, err := rand.Read(raw); err != nil {
		panic(err)
	}
	return "media_" + strings.TrimSpace(hex.EncodeToString(raw))
}
