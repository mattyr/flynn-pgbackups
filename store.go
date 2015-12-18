package main

import (
	"errors"
	"fmt"
	"io"

	"github.com/rlmcpherson/s3gof3r"
)

type Storer interface {
	DownloadUrl(appId string, backupId string) (string, error)
	// return bytes written
	Put(appId string, backupId string, r io.Reader) (int64, error)
	Delete(appId string, backupId string) error
}

type s3store struct {
	bucket *s3gof3r.Bucket
}

func NewS3Store(bucketName string) (Storer, error) {
	keys, err := s3gof3r.EnvKeys()
	if err != nil {
		return nil, err
	}

	s3 := s3gof3r.New("", keys)
	bucket := s3.Bucket(bucketName)

	return &s3store{bucket: bucket}, nil
}

func (s *s3store) DownloadUrl(appId string, backupId string) (string, error) {
	return "", errors.New("todo")
}

func (s *s3store) Put(appId string, backupId string, r io.Reader) (int64, error) {
	s3Path := fmt.Sprintf("pgbackups/%s/%s.backup", appId, backupId)

	s3Putter, err := s.bucket.PutWriter(s3Path, nil, nil)
	if err != nil {
		return -1, err
	}
	defer s3Putter.Close()

	return io.Copy(s3Putter, r)
}

func (*s3store) Delete(appId string, backupId string) error {
	panic("todo")
}
