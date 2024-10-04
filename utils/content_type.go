package utils

import (
	"mime"
	"path/filepath"
	"strings"
)

func AutoDetectContentType(path string) string {
	path = strings.TrimSuffix(path, ".map")
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}
	ct := mime.TypeByExtension(ext)
	cts := strings.SplitN(ct, ";", 2)
	return cts[0]
}
