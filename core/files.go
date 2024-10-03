package core

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/IGLOU-EU/go-wildcard/v2"
	"github.com/rizaldntr/storage-service-website-action/config"
	"github.com/rizaldntr/storage-service-website-action/types"
	"github.com/rizaldntr/storage-service-website-action/utils"
	"github.com/sethvargo/go-githubactions"
)

var sema = make(chan struct{}, 20)

func WalkDir(config config.Config) <-chan types.FileInfo {
	files := make(chan types.FileInfo)
	var sw sync.WaitGroup
	sw.Add(1)
	go walkDir(config.Folder, config.Folder, &sw, config.FileConfig, files)
	go func() {
		sw.Wait()
		close(files)
	}()
	return files
}

func walkDir(dir, root string, sw *sync.WaitGroup, config config.FileConfig, files chan<- types.FileInfo) {
	defer sw.Done()

	for _, entry := range dirents(dir) {
		if entry.IsDir() {
			sw.Add(1)
			subDir := filepath.Join(dir, entry.Name())
			go walkDir(subDir, root, sw, config, files)
		} else {
			path := filepath.Join(dir, entry.Name())
			if isExcluded(path, config) {
				continue
			}

			md5, err := utils.HashMD5(path)
			if err != nil {
				githubactions.Debugf("Error hashing file: %v", err)
			}

			file := types.FileInfo{
				ACL:          config.DefaultACL,
				CacheControl: config.DefaultCacheControl,
				ContentMD5:   md5,
				Dir:          root,
				Name:         entry.Name(),
				SourcePath:   path,
				TargetPath:   strings.TrimPrefix(path, root),
			}
			setCacheControlAndFileType(config, &file)
			processRegexConfig(&file, config.ObjectRules)
			files <- file
		}
	}
}

func dirents(dir string) []fs.DirEntry {
	sema <- struct{}{}
	defer func() { <-sema }()

	entries, err := os.ReadDir(dir)
	if err != nil {
		githubactions.Errorf("Error reading directory: %v", err)
		return nil
	}
	return entries
}

func isExcluded(path string, config config.FileConfig) bool {
	for _, pattern := range config.ExcludePatterns {
		if wildcard.Match(pattern, path) {
			githubactions.Infof("%s is excluded by pattern %s", path, pattern)
			return true
		}
	}
	return false
}

func processRegexConfig(file *types.FileInfo, regexConfigs []config.ObjectRule) {
	var regexConfig *config.ObjectRule
	for _, config := range regexConfigs {
		if wildcard.Match(config.Pattern, file.SourcePath) {
			regexConfig = &config
			break
		}
	}

	if regexConfig != nil {
		if regexConfig.ACL != "" {
			file.ACL = regexConfig.ACL
		}
		if regexConfig.CacheControl != "" {
			file.CacheControl = regexConfig.CacheControl
		}
	}
}

func setCacheControlAndFileType(config config.FileConfig, file *types.FileInfo) {
	path := file.SourcePath
	switch {
	case utils.IsHTML(path):
		file.CacheControl = config.DefaultHTMLCacheControl
		file.FileType = types.HTML
	case utils.IsPDF(path):
		file.CacheControl = config.DefaultPDFCacheControl
		file.FileType = types.PDF
	case utils.IsImage(path):
		file.CacheControl = config.DefaultImageCacheControl
		file.FileType = types.Image
	default:
		file.FileType = types.Other
		file.CacheControl = config.DefaultCacheControl
	}
}
