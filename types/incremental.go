package types

import (
	"encoding/json"
	"sync"
)

type IncrementalConfigValue struct {
	ContentMD5   string
	CacheControl string
}

type IncrementalConfig struct {
	sync.RWMutex
	M map[string]IncrementalConfigValue
}

func NewIncrementalConfig() *IncrementalConfig {
	return &IncrementalConfig{
		M: make(map[string]IncrementalConfigValue),
	}
}

func IncrementalConfigFromFileInfos(files []FileInfo) *IncrementalConfig {
	i := NewIncrementalConfig()
	for _, file := range files {
		i.M[file.TargetPath] = IncrementalConfigValue{
			ContentMD5:   file.ContentMD5,
			CacheControl: file.CacheControl,
		}
	}
	return i
}

func (i *IncrementalConfig) Get(file FileInfo) (IncrementalConfigValue, bool) {
	i.RLock()
	defer i.RUnlock()

	v, ok := i.M[file.TargetPath]
	return v, ok
}

func (i *IncrementalConfig) Set(file FileInfo) {
	i.Lock()
	defer i.Unlock()

	i.M[file.TargetPath] = IncrementalConfigValue{
		ContentMD5:   file.ContentMD5,
		CacheControl: file.CacheControl,
	}
}

func (i *IncrementalConfig) Delete(file FileInfo) {
	i.Lock()
	defer i.Unlock()

	delete(i.M, file.TargetPath)
}

func (i *IncrementalConfig) UnmarshalJSON(data []byte) error {
	i.Lock()
	defer i.Unlock()

	return json.Unmarshal(data, &i.M)
}

func (i *IncrementalConfig) MarshalJSON() ([]byte, error) {
	i.RLock()
	defer i.RUnlock()

	return json.Marshal(i.M)
}

func (i *IncrementalConfig) Size() int {
	i.RLock()
	defer i.RUnlock()

	return len(i.M)
}
