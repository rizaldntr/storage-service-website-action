package types

type FileInfo struct {
	ACL          ObjectACL
	CacheControl string
	ContentType  string
	ContentMD5   string
	Dir          string
	Name         string
	SourcePath   string
	TargetPath   string
	FileType     FileType
}
