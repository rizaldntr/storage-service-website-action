package core

import (
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"

	"github.com/sethvargo/go-githubactions"
)

var sema = make(chan struct{}, 20)

type RegexFileConfig struct {
	Pattern      *regexp.Regexp
	ACL          string
	CacheControl string
}

type FileConfig struct {
	DefaultACL               string
	DefaultCacheControl      string
	DefaultHTMLCacheControl  string
	DefaultImageCacheControl string
	DefaultPDFCacheControl   string
	ExcludePatterns          []*regexp.Regexp
	RegexFileConfigs         []RegexFileConfig
}

type FileInfo struct {
	ACL          string
	CacheControl string
	ContentType  string
	ContentMD5   string
	Dir          string
	Name         string
	SourcePath   string
	TargetPath   string
}

func WalkDir(root string, config *FileConfig) <-chan FileInfo {
	files := make(chan FileInfo)
	var sw sync.WaitGroup
	sw.Add(1)
	go walkDir(root, &sw, config, files)
	go func() {
		sw.Wait()
		close(files)
	}()
	return files
}

func walkDir(root string, sw *sync.WaitGroup, config *FileConfig, files chan<- FileInfo) {
	defer sw.Done()

	for _, entry := range dirents(root) {
		if entry.IsDir() {
			sw.Add(1)
			subdir := filepath.Join(root, entry.Name())
			go walkDir(subdir, sw, config, files)
		} else {
			path := filepath.Join(root, entry.Name())
			if isExcluded(path, config) {
				continue
			}

			md5, err := HashMD5(path)
			if err != nil {
				githubactions.Debugf("Error hashing file: %v", err)
			}

			file := FileInfo{
				ACL:          config.DefaultACL,
				CacheControl: config.DefaultCacheControl,
				ContentMD5:   md5,
				Dir:          root,
				Name:         entry.Name(),
				SourcePath:   path,
				TargetPath:   filepath.ToSlash(path),
			}
			processRegexConfig(&file, config.RegexFileConfigs)
			files <- file
		}
	}
}

func dirents(dir string) []fs.DirEntry {
	sema <- struct{}{}
	defer func() { <-sema }()

	entries, err := os.ReadDir(dir)
	if err != nil {
		log.Printf("du1: %v\n", err)
		return nil
	}
	return entries
}

func isExcluded(path string, config *FileConfig) bool {
	for _, pattern := range config.ExcludePatterns {
		if pattern.MatchString(path) {
			return true
		}
	}
	return false
}

func processRegexConfig(file *FileInfo, regexConfigs []RegexFileConfig) {
	var regexConfig *RegexFileConfig
	for _, config := range regexConfigs {
		if config.Pattern.MatchString(file.Name) {
			regexConfig = &config
			break
		}
	}

	if regexConfig != nil {
		file.ACL = regexConfig.ACL
		file.CacheControl = regexConfig.CacheControl
	}
}
