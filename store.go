package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/AdRoll/goamz/aws"
	"github.com/AdRoll/goamz/s3"
	"github.com/rlmcpherson/s3gof3r"
)

type Storer interface {
	DownloadUrl(appId string, backupId string) (string, error)
	// return bytes written
	Put(appId string, backupId string, r io.Reader) (int64, error)
	Delete(appId string, backupId string) error
}

type s3store struct {
	bucketName string
	bucket     *s3gof3r.Bucket
	regionName string
}

func NewS3Store(bucketName string, regionName string) (Storer, error) {
	keys, err := s3gof3r.EnvKeys()
	if err != nil {
		return nil, err
	}
	regionName = getRegion(regionName)
	s3Domain := fmt.Sprintf("s3.%s.amazonaws.com", regionName)
	s3 := s3gof3r.New(s3Domain, keys)
	bucket := s3.Bucket(bucketName)

	return &s3store{bucketName: bucketName, bucket: bucket, regionName: regionName}, nil
}

func (s *s3store) DownloadUrl(appId string, backupId string) (string, error) {
	auth, err := aws.EnvAuth()
	if err != nil {
		return "", err
	}
	svc := s3.New(auth, aws.GetRegion(s.regionName))
	b := svc.Bucket(s.bucketName)
	return b.SignedURL(s.pathFor(appId, backupId), time.Now().Add(20*time.Minute)), nil
}

func (s *s3store) Put(appId string, backupId string, r io.Reader) (int64, error) {
	s3Path := s.pathFor(appId, backupId)

	s3Putter, err := s.bucket.PutWriter(s3Path, nil, nil)
	if err != nil {
		return -1, err
	}
	defer s3Putter.Close()

	return io.Copy(s3Putter, r)
}

func (s *s3store) Delete(appId string, backupId string) error {
	return s.bucket.Delete(s.pathFor(appId, backupId))
}

func (*s3store) pathFor(appId string, backupId string) string {
	return fmt.Sprintf("pgbackups/%s/%s.backup", appId, backupId)
}

func getRegion(regionName string) string {
	if regionName == "" {
		regionName = os.Getenv("AWS_REGION")
		if regionName == "" {
			regionName = "us-east-1"
		}
	}
	return regionName
}
