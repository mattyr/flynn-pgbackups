package main

import (
	"encoding/base64"
	"fmt"
	"io"
	"os"

	"github.com/flynn/flynn/controller/client"
	ct "github.com/flynn/flynn/controller/types"
	"github.com/flynn/flynn/pkg/cluster"
)

type FlynnClient struct {
	client *controller.Client
}

type AppAndRelease struct {
	App     *ct.App
	Release *ct.Release
}

func NewFlynnClient() (*FlynnClient, error) {
	// get config from env vars
	controllerUrl := os.Getenv("CONTROLLER_URL")
	controllerKey := os.Getenv("CONTROLLER_KEY")
	controllerTlsPin := os.Getenv("CONTROLLER_TLS_PIN")

	var c *controller.Client
	var err error
	if controllerTlsPin == "" {
		c, err = controller.NewClient(controllerUrl, controllerKey)
		if err != nil {
			return nil, err
		}
	} else {
		pin, err := base64.StdEncoding.DecodeString(controllerTlsPin)
		if err != nil {
			return nil, err
		}
		c, err = controller.NewClientWithConfig(controllerUrl, controllerKey, controller.Config{Pin: pin})
		if err != nil {
			return nil, err
		}
	}

	return &FlynnClient{client: c}, nil
}

func (c *FlynnClient) GetApp(name string) (*ct.App, error) {
	return c.client.GetApp(name)
}

func (c *FlynnClient) AppList() ([]*AppAndRelease, error) {
	allApps, err := c.client.AppList()
	if err != nil {
		return nil, err
	}

	result := []*AppAndRelease{}

	for _, a := range allApps {
		r, _ := c.client.GetAppRelease(a.ID)
		// identify apps to backup by FLYNN_POSTGRES env var existing
		if r.Env["FLYNN_POSTGRES"] != "" {
			result = append(result, &AppAndRelease{App: a, Release: r})
		}
	}

	return result, nil
}

func (c *FlynnClient) StreamBackup(app *AppAndRelease, w io.Writer) error {
	req, err := c.createPgBackupJobRequest(app)
	if err != nil {
		return err
	}

	rwc, err := c.client.RunJobAttached(app.App.ID, req)
	if err != nil {
		return err
	}
	defer rwc.Close()

	attachClient := cluster.NewAttachClient(rwc)
	attachClient.CloseWrite()

	// not worried about exit status...?
	_, err = attachClient.Receive(w, os.Stderr)
	return err
}

func (c *FlynnClient) createPgBackupJobRequest(app *AppAndRelease) (*ct.NewJob, error) {
	// from: https://github.com/flynn/flynn/blob/master/cli/pg.go
	pgApp := app.Release.Env["FLYNN_POSTGRES"]
	if pgApp == "" {
		return nil, fmt.Errorf("no postgres database found.")
	}

	// TODO: this pgRelease is likely shared by all/most the apps.  cache result
	pgRelease, err := c.client.GetAppRelease(pgApp)
	if err != nil {
		return nil, fmt.Errorf("error getting postgres release: %s", err)
	}

	req := &ct.NewJob{
		Entrypoint: []string{"pg_dump"},
		Cmd:        []string{"--format=custom", "--no-owner", "--no-acl"},
		TTY:        false,
		ReleaseID:  pgRelease.ID,
		ReleaseEnv: false,
		Env:        make(map[string]string),
		DisableLog: true,
	}

	for _, k := range []string{"PGHOST", "PGUSER", "PGPASSWORD", "PGDATABASE"} {
		v := app.Release.Env[k]
		if v == "" {
			return nil, fmt.Errorf("missing %s in app environment", k)
		}
		req.Env[k] = v
	}

	return req, nil
}
