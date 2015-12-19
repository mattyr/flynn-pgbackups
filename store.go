package main

import (
	"fmt"
	"io"
	"time"

	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/s3"
	"github.com/rlmcpherson/s3gof3r"
)

type Storer interface {
	DownloadUrl(appId string, backupId string) (string, error)
	GetPutter(appId string, backupId string) (io.WriteCloser, error)
	Delete(appId string, backupId string) error
}

type s3store struct {
	bucketName string
	bucket     *s3gof3r.Bucket
}

func NewS3Store(bucketName string) (Storer, error) {
	keys, err := s3gof3r.EnvKeys()
	if err != nil {
		return nil, err
	}

	s3 := s3gof3r.New("", keys)
	bucket := s3.Bucket(bucketName)

	return &s3store{bucketName: bucketName, bucket: bucket}, nil
}

func (s *s3store) DownloadUrl(appId string, backupId string) (string, error) {
	auth, err := aws.EnvAuth()
	if err != nil {
		return "", err
	}
	svc := s3.New(auth, aws.GetRegion("us-east-1")) // TODO: configurable?
	b := svc.Bucket(s.bucketName)
	return b.SignedURL(s.pathFor(appId, backupId), time.Now().Add(20*time.Minute)), nil
}

func (s *s3store) GetPutter(appId string, backupId string) (io.WriteCloser, error) {
	s3Path := s.pathFor(appId, backupId)

	return s.bucket.PutWriter(s3Path, nil, nil)
}

func (s *s3store) Delete(appId string, backupId string) error {
	return s.bucket.Delete(s.pathFor(appId, backupId))
}

func (*s3store) pathFor(appId string, backupId string) string {
	return fmt.Sprintf("pgbackups/%s/%s.backup", appId, backupId)
}
