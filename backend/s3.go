package backend

import (
	"context"
	"errors"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	awstypes "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/rizaldntr/storage-service-website-action/config"
	"github.com/rizaldntr/storage-service-website-action/types"
)

type S3 struct {
	client *s3.Client
	bucket string
}

func NewS3(config config.Config) (*S3, error) {
	sdkConfig, err := awsconfig.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(sdkConfig)
	return &S3{
		client: s3Client,
		bucket: config.Bucket,
	}, nil
}

func (s *S3) GetObject(key string) ([]byte, error) {
	result, err := s.client.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFoundErr *awstypes.NoSuchKey
		if errors.As(err, &notFoundErr) {
			return nil, types.ObjectNotFoundError
		}
		return nil, err
	}
	defer result.Body.Close()

	data, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *S3) ListObjects(token *string) ([]string, error) {
	result, err := s.client.ListObjectsV2(context.TODO(), &s3.ListObjectsV2Input{
		Bucket:            aws.String(s.bucket),
		ContinuationToken: token,
	})
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, obj := range result.Contents {
		keys = append(keys, *obj.Key)
	}

	return keys, nil
}

func (s *S3) PutObject(request types.PutObjectRequest) error {
	_, err := s.client.PutObject(context.TODO(), &s3.PutObjectInput{
		Bucket:       aws.String(s.bucket),
		Key:          aws.String(request.Key),
		Body:         request.Body,
		CacheControl: aws.String(request.CacheControl),
		ContentType:  aws.String(request.ContentType),
		ACL: func() awstypes.ObjectCannedACL {
			if request.ACL == types.PublicACL {
				return awstypes.ObjectCannedACLPublicRead
			}
			return awstypes.ObjectCannedACLPrivate
		}(),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3) DeleteObject(key string) error {
	_, err := s.client.DeleteObject(context.TODO(), &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3) DeleteObjects(keys []string) error {
	objectsToDelete := make([]awstypes.ObjectIdentifier, 0, len(keys))
	for _, key := range keys {
		objectsToDelete = append(objectsToDelete, awstypes.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	resp, err := s.client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
		Bucket: aws.String(s.bucket),
		Delete: &awstypes.Delete{
			Objects: objectsToDelete,
		},
	})
	if err != nil {
		return err
	}
	if len(resp.Errors) > 0 {
		return errors.New("There are errors when deleting objects")
	}

	return nil
}

func (s *S3) EmptyBucket() error {
	params := &s3.ListObjectsInput{
		Bucket: aws.String(s.bucket),
	}

	for {
		resp, err := s.client.ListObjects(context.TODO(), params)
		if err != nil {
			return err
		}

		if len(resp.Contents) == 0 {
			return nil
		}

		objectsToDelete := make([]awstypes.ObjectIdentifier, 0, 1)
		for _, obj := range resp.Contents {
			objectsToDelete = append(objectsToDelete, awstypes.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		deleteResp, err := s.client.DeleteObjects(context.TODO(), &s3.DeleteObjectsInput{
			Bucket: aws.String(s.bucket),
			Delete: &awstypes.Delete{
				Objects: objectsToDelete,
			},
		})
		if err != nil {
			return err
		}

		if len(deleteResp.Errors) > 0 {
			return errors.New("There are errors when deleting objects")
		}

		if !*resp.IsTruncated {
			break
		}
		params.Marker = resp.Contents[len(resp.Contents)-1].Key
	}

	return nil
}
