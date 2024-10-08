package core

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"sync/atomic"

	"github.com/rizaldntr/storage-service-website-action/backend"
	"github.com/rizaldntr/storage-service-website-action/config"
	"github.com/rizaldntr/storage-service-website-action/types"
	"github.com/sethvargo/go-githubactions"
)

const IncrementalConfig = ".incremental"

type Backend interface {
	GetObject(key string) ([]byte, error)
	PutObject(request types.PutObjectRequest) error
	DeleteObject(key string) error
	DeleteObjects(keys []string) error
	EmptyBucket() error
}

func Process(config config.Config) error {
	backend, err := backend.NewS3(config)
	if err != nil {
		return err
	}

	githubactions.Infof("Initiating incremental upload")
	githubactions.Group("Fetching .fileinfo from backend storage")
	ibytes, err := backend.GetObject(IncrementalConfig)
	if err != nil {
		githubactions.Warningf("Unable to retrieve .fileinfo: %v", err)
		githubactions.Warningf("Proceeding to upload all files")
	}

	incremental := types.NewIncrementalConfig()
	err = incremental.UnmarshalJSON(ibytes)
	if err != nil {
		githubactions.Warningf("Failed to unmarshal .fileinfo: %v", err)
	}
	githubactions.EndGroup()

	// Cleanup bucket for first run
	if incremental.Size() == 0 {
		githubactions.Group("Cleaning up bucket for first run")
		githubactions.Infof("Starting cleanup process")
		if err := backend.EmptyBucket(); err != nil {
			githubactions.Warningf("Error during bucket cleanup: %v", err)
		}
		githubactions.Infof("Cleanup process completed")
	}
	githubactions.EndGroup()

	githubactions.Group("Uploading files")
	githubactions.Infof("Commencing file upload")
	files := WalkDir(config)
	uploaded, _ := upload(backend, files, incremental)
	githubactions.Infof("File upload completed")
	githubactions.EndGroup()

	if incremental.Size() > 0 {
		githubactions.Group("Removing leftover files")
		githubactions.Infof("Commencing removal of leftover files")
		errs := delete(backend, incremental)
		if len(errs) > 0 {
			githubactions.Warningf("Error while removing leftover files: %v", errs)
		}
		githubactions.Infof("Removal of leftover files completed")
		githubactions.EndGroup()
	}

	githubactions.Group("Saving incremental configuration")
	githubactions.Infof("Generating incremental configuration")
	newIncremental := types.IncrementalConfigFromFileInfos(uploaded)
	nbytes, err := newIncremental.MarshalJSON()
	if err != nil {
		githubactions.Warningf("Error during .fileinfo marshalling: %v", err)
	}
	githubactions.Infof("Saving incremental configuration")
	err = backend.PutObject(types.PutObjectRequest{
		ACL:  types.PrivateACL,
		Body: bytes.NewReader(nbytes),
		Key:  IncrementalConfig,
	})
	if err != nil {
		githubactions.Warningf("Error while saving .fileinfo: %v", err)
	}
	githubactions.Infof("Incremental configuration saving completed")
	githubactions.EndGroup()

	return nil
}

func upload(backend Backend, files <-chan types.FileInfo, i *types.IncrementalConfig) ([]types.FileInfo, []error) {
	var sw sync.WaitGroup
	var sema = make(chan struct{}, 30)
	var errMutex sync.Mutex
	var uplMutex sync.Mutex
	var errs []error
	var totalError atomic.Int64
	var totalFile atomic.Int64
	var totalSkipped atomic.Int64
	var totalUploadedFiles atomic.Int64
	uploaded := make([]types.FileInfo, 0, 100)

	for file := range files {
		sw.Add(1)
		go func(file types.FileInfo) {
			defer sw.Done()
			objectKey := file.TargetPath
			totalFile.Add(1)

			if shouldSkip(file, i) {
				uplMutex.Lock()
				uploaded = append(uploaded, file)
				uplMutex.Unlock()
				totalSkipped.Add(1)
				githubactions.Infof("Skipping upload of %s as the content is unchanged", objectKey)
				return
			}

			sema <- struct{}{}
			upl, err := handleUpload(backend, file)
			<-sema
			if err != nil {
				errMutex.Lock()
				errs = append(errs, err)
				errMutex.Unlock()
				totalError.Add(1)
				githubactions.Errorf("Error while uploading %s: %v", objectKey, err)
				return
			}

			uplMutex.Lock()
			uploaded = append(uploaded, upl...)
			uplMutex.Unlock()
			totalUploadedFiles.Add(1)
			githubactions.Infof("Successfully uploaded %s", objectKey)
		}(file)
	}
	sw.Wait()

	githubactions.Infof("Total Files: ", totalFile.Load())
	githubactions.Infof("Total Skipped Files: ", totalSkipped.Load())
	githubactions.Infof("Total Uploaded Files: ", totalUploadedFiles.Load())
	githubactions.Infof("Total Errors: ", totalError.Load())

	return uploaded, errs
}

func delete(backend Backend, i *types.IncrementalConfig) []error {
	count := 0
	maxKeys := 1000
	keys := make([]string, 0, maxKeys)
	deletedKeys := make([]string, 0, 20)

	var errs []error
	var sw sync.WaitGroup
	var errMutex sync.Mutex
	var delMutex sync.Mutex

	sema := make(chan struct{}, 10)
	for k := range i.M {
		keys = append(keys, k)
		count++
		if (count > 0 && count%maxKeys == 0) || count == len(i.M) {
			sw.Add(1)
			go func(keys []string) {
				defer sw.Done()
				sema <- struct{}{}
				err := backend.DeleteObjects(keys)
				<-sema
				if err != nil {
					errMutex.Lock()
					errs = append(errs, err)
					errMutex.Unlock()
					githubactions.Errorf("Error while deleting objects: %v", err)
				} else {
					delMutex.Lock()
					deletedKeys = append(deletedKeys, keys...)
					delMutex.Unlock()
				}
			}(keys)
			keys = make([]string, 0, maxKeys)
		}
	}
	sw.Wait()
	for _, key := range deletedKeys {
		githubactions.Infof("Successfully deleted %s", key)
	}
	return errs
}

func handleUpload(backend Backend, file types.FileInfo) ([]types.FileInfo, error) {
	body, err := os.Open(file.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("Error opening file %s: %v", file.SourcePath, err)
	}

	objectKey := file.TargetPath
	result := make([]types.FileInfo, 0, 2)
	err = backend.PutObject(types.PutObjectRequest{
		ACL:          file.ACL,
		Body:         body,
		CacheControl: file.CacheControl,
		ContentType:  file.ContentType,
		Key:          objectKey,
	})
	if err != nil {
		return nil, fmt.Errorf("Error uploading file %s: %v", objectKey, err)
	}
	result = append(result, file)
	return result, nil
}

func shouldSkip(item types.FileInfo, i *types.IncrementalConfig) bool {
	remoteConfig, ok := i.Get(item)
	if !ok {
		return false
	}

	// delete the item from the incremental config
	// later we will delete leftover items from the incremental config
	i.Delete(item)

	if item.ContentMD5 != "" && item.ContentMD5 == remoteConfig.ContentMD5 &&
		item.CacheControl == remoteConfig.CacheControl && item.ContentType == remoteConfig.ContentType {
		return true
	}

	return false
}
