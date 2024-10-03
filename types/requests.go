package types

import (
	"io"
)

type PutObjectRequest struct {
	Key          string
	Body         io.Reader
	ContentType  string
	CacheControl string
	ACL          ObjectACL
}
