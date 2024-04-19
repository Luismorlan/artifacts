package file_store

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rnr-capital/newsfeed-backend/utils"
	Logger "github.com/rnr-capital/newsfeed-backend/utils/log"
)

const (
	TestS3Bucket      = "collector-dev-bucket"
	ProdS3ImageBucket = "newsfeed-crawler-image-output"
	ProdS3FileBucket  = "newsfeed-crawler-file-output"
	CouldFrontPrefix  = "https://d20uffqoe1h0vv.cloudfront.net/"
)

type S3FileStore struct {
	bucket                    string
	uploader                  *s3manager.Uploader
	svc                       *s3.S3
	processUrlBeforeFetchFunc ProcessUrlBeforeFetchFuncType
	customizeFileNameFunc     CustomizeFileNameFuncType
	customizeFileExtFunc      CustomizeFileExtFuncType
	customizeUploadedUrlFunc  CustomizeUploadedUrlType
}

func NewS3FileStore(bucket string) (*S3FileStore, error) {
	// AWS client session
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-1"),
	})
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(sess)

	return &S3FileStore{
		bucket:                    bucket,
		uploader:                  uploader,
		svc:                       s3.New(session.Must(sess, err)),
		processUrlBeforeFetchFunc: func(s string) string { return s },
		customizeFileNameFunc:     nil,
		customizeFileExtFunc:      nil,
		customizeUploadedUrlFunc:  nil,
	}, nil
}

func (s *S3FileStore) SetProcessUrlBeforeFetchFunc(f ProcessUrlBeforeFetchFuncType) {
	s.processUrlBeforeFetchFunc = f
}

func (s *S3FileStore) SetCustomizeFileNameFunc(f CustomizeFileNameFuncType) {
	s.customizeFileNameFunc = f
}

func (s *S3FileStore) SetCustomizeFileExtFunc(f CustomizeFileExtFuncType) {
	s.customizeFileExtFunc = f
}

func (s *S3FileStore) SetCustomizeUploadedUrlFunc(f CustomizeUploadedUrlType) {
	s.customizeUploadedUrlFunc = f
}

// S3 key is the file name
func (s *S3FileStore) GenerateS3KeyFromUrl(url, fileName string) (key string, err error) {
	if s.customizeFileNameFunc != nil {
		key = s.customizeFileNameFunc(url, fileName)
	} else {
		key, err = utils.TextToMd5Hash(url)
	}

	if len(key) == 0 {
		err = errors.New("generate empty s3 key, invalid")
	}

	// TODO: merge with customizeFileNameFunc
	if s.customizeFileExtFunc != nil {
		key = key + s.customizeFileExtFunc(url, fileName)
	} else {
		if fileName != "" {
			key = key + utils.GetUrlExtNameWithDot(fileName)
		} else {
			key = key + utils.GetUrlExtNameWithDot(url)
		}
	}

	return key, err
}

// If url key existed, just return the existing key without update file
func (s *S3FileStore) FetchAndStore(url, fileName string) (key string, err error) {
	// Download file
	eventualUrl := s.processUrlBeforeFetchFunc(url)
	response, err := http.Get(eventualUrl)
	if err != nil {
		return "", fmt.Errorf("failed to get image %w", err)
	}
	key, err = s.GenerateS3KeyFromUrl(url, fileName)
	if err != nil {
		Logger.LogV2.Info(fmt.Sprint("Fail to download file from url:", eventualUrl, "err:", err))
		return "", fmt.Errorf("failed to generate s3 key %w", err)
	}

	if !s.IsKeyExisted(key) {
		// Upload the file to S3.
		_, err = s.uploader.Upload(&s3manager.UploadInput{
			ACL:    aws.String("public-read"),
			Bucket: aws.String(s.bucket),
			Key:    aws.String(key),
			Body:   response.Body,
		})
	}
	if err != nil {
		return key, fmt.Errorf("failed to upload to s3 %w", err)
	}
	return key, nil
}

func (s *S3FileStore) IsKeyExisted(key string) bool {
	_, err := s.svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String("bucket_name"),
		Key:    aws.String("object_key"),
	})
	return err == nil
}

func (s *S3FileStore) GetUrlFromKey(key string) string {
	if s.customizeUploadedUrlFunc == nil {
		return CouldFrontPrefix + key
	}
	return s.customizeUploadedUrlFunc(key)
}

func (s *S3FileStore) CleanUp() {
	// do nothing for s3
}
