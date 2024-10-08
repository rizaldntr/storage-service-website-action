package config

import (
	"os"
	"path"
	"sync"

	"github.com/joho/godotenv"
	"github.com/rizaldntr/storage-service-website-action/types"
	"github.com/rizaldntr/storage-service-website-action/utils"
	"github.com/sethvargo/go-githubactions"
	"gopkg.in/yaml.v2"
)

var (
	config Config
	once   sync.Once
)

type ObjectRule struct {
	Pattern      string          `yaml:"pattern"`
	ACL          types.ObjectACL `yaml:"acl"`
	CacheControl string          `yaml:"cache-control"`
}

type FileConfig struct {
	DefaultACL                   types.ObjectACL
	DefaultCacheControl          string
	DefaultHTMLCacheControl      string
	DefaultImageCacheControl     string
	DefaultPDFCacheControl       string
	ExcludePatterns              []string
	ObjectRules                  []ObjectRule
	RemoveHTMLExtension          bool
	DuplicateHTMLWithNoExtension bool
}

type Config struct {
	Folder     string
	FileConfig FileConfig
	Bucket     string
}

func getACL() types.ObjectACL {
	acl := os.Getenv("ACL")
	if acl == "private" {
		return types.PrivateACL
	}
	return types.PublicACL
}

func Get() Config {
	once.Do(func() {
		godotenv.Load(".env")
		var rules []ObjectRule
		if err := yaml.Unmarshal([]byte(os.Getenv("OBJECT_RULES")), &rules); err != nil {
			githubactions.Fatalf("Failed to unmarshal file-configs: %v", err)
		}

		config = Config{
			Folder: path.Clean(os.Getenv("FOLDER")) + "/",
			FileConfig: FileConfig{
				DefaultACL:                   getACL(),
				DefaultCacheControl:          utils.GetEnvOrDefault("DEFAULT_CACHE_CONTROL", "max-age=2592000"),
				DefaultHTMLCacheControl:      utils.GetEnvOrDefault("HTML_CACHE_CONTROL", "max-age=600"),
				DefaultImageCacheControl:     utils.GetEnvOrDefault("IMAGE_CACHE_CONTROL", "max-age=864000"),
				DefaultPDFCacheControl:       utils.GetEnvOrDefault("PDF_CACHE_CONTROL", "max-age=2592000"),
				ExcludePatterns:              utils.GetActionInputAsSlice("EXCLUDE"),
				ObjectRules:                  rules,
				RemoveHTMLExtension:          utils.GetEnvOrDefault("REMOVE_HTML_EXTENSION", "false") == "true",
				DuplicateHTMLWithNoExtension: utils.GetEnvOrDefault("DUPLICATE_HTML_WITH_NO_EXTENSION", "false") == "true",
			},
			Bucket: os.Getenv("BUCKET"),
		}
	})
	return config
}
