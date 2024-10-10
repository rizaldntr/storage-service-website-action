package utils

import (
	"mime"
	"path/filepath"
	"strings"
)

func init() {
	mime.AddExtensionType(".map", "application/json")
	mime.AddExtensionType(".svg", "image/svg+xml")
}

func AutoDetectContentType(path string) string {
	ext := filepath.Ext(path)
	if ext == "" {
		return "application/octet-stream"
	}
	ct := mime.TypeByExtension(ext)
	cts := strings.SplitN(ct, ";", 2)
	return cts[0]
}
