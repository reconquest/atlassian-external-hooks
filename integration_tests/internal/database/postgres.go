package database

import (
	"fmt"
	"strings"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/lexec-go"
)

type Postgres struct {
	container string
	volume    string
}

func (postgres *Postgres) Addr() string {
	return "postgres"
}

func (postgres *Postgres) Container() string {
	return postgres.container
}

func newPostgres() (*Postgres, error) {
	name := "aeh-postgres-" + lojban.GetRandomID(4)

	instance := &Postgres{}
	instance.volume = name

	execution := exec.New(
		"docker",
		"run", "-d",
		"-e", "POSTGRES_PASSWORD=bitbucket",
		"-e", "POSTGRES_DB=bitbucket",
		"-e", "POSTGRES_USER=bitbucket",
		"-v", fmt.Sprintf(
			"%s:%s",
			instance.volume,
			"/var/lib/postgresql/data",
		),
		"postgres",
	)

	stdout, _, err := execution.Output()
	if err != nil {
		return nil, err
	}

	instance.container = strings.TrimSpace(string(stdout))

	for {
		waiter := exec.New(
			"docker", "exec", instance.container,
			"pg_isready",
		)

		err := waiter.Run()
		if err != nil {
			if !lexec.IsExitStatus(err) {
				return nil, err
			}

			time.Sleep(time.Millisecond * 50)
			continue
		}

		break
	}

	return instance, nil
}
