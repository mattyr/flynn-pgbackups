package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"

	"github.com/rlmcpherson/s3gof3r"

	"github.com/flynn/flynn/controller/client"
)

func main() {
	bucketName := os.Getenv("S3_BUCKET")
	controllerUri := os.Getenv("CONTROLLER_URL")
	controllerKey := os.Getenv("CONTROLLER_KEY")

	keys, err := s3gof3r.EnvKeys()
	if err != nil {
		panic(err)
	}
	s3 := s3gof3r.New("", keys)
	bucket := s3.Bucket(bucketName)

	c, err := controller.NewClient(controllerUri, controllerKey)
	if err != nil {
		panic(err)
	}

	apps, err := c.AppList()
	for _, a := range apps {
		r, _ := c.GetAppRelease(a.ID)
		// TODO: log err, but continue
		if r.Env["PGDATABASE"] != "" {
			log.Printf("Backing up %s (%s)", a.Name, a.ID)

			cmd := exec.Command("pg_dump", "--format=custom", "--no-owner", "--no-acl")

			env := make([]string, 4)
			for i, k := range []string{"PGHOST", "PGUSER", "PGPASSWORD", "PGDATABASE"} {
				env[i] = fmt.Sprintf("%s=%s", k, r.Env[k])
			}
			cmd.Env = env

			now := time.Now()
			s3Path := fmt.Sprintf("pgbackups/%s/%s.backup", a.ID, now.Format(time.RFC3339))
			s3Putter, err := bucket.PutWriter(s3Path, nil, nil)
			if err == nil {
				// TODO: log err, but continue
				cmd.Stdout = s3Putter
				cmd.Stderr = os.Stderr
				cmd.Run()
				s3Putter.Close()
				log.Printf("Completed backing up %s (%s)", a.Name, a.ID)
			} else {
				log.Printf("Error backing up %s (%s): %s", a.Name, a.ID, err)
			}
		}
	}
}
