package database

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

type Postgres struct {
	container string
}

func (postgres *Postgres) Driver() string {
	return "org.postgresql.Driver"
}

func (postgres *Postgres) URL() string {
	return "jdbc:postgresql://" + postgres.container + ":5432/bitbucket"
}

func (postgres *Postgres) User() string {
	return "bitbucket"
}

func (postgres *Postgres) Password() string {
	return "bitbucket"
}

func (postgres *Postgres) Container() string {
	return postgres.container
}

func (postgres *Postgres) Volume() string {
	return postgres.container
}

func (postgres *Postgres) start() error {
	execution := exec.New(
		"docker",
		"run",
		"--name", postgres.container,
		"-d",
		"-v", postgres.container+":/var/lib/postgresql/data",
		"-e", "POSTGRES_DB=bitbucket",
		"-e", "POSTGRES_USER=bitbucket",
		"-e", "POSTGRES_PASSWORD=bitbucket",
		"postgres:latest",
	)
	err := execution.Run()
	if err != nil {
		return karma.Format(err, "start postgres container")
	}

	ip, err := inspectIP(postgres.container)
	if err != nil {
		return karma.Format(err, "inspect ip address")
	}

	waitService(ip, 5432, "postgres")

	return nil
}

func (postgres *Postgres) Stop() error {
	return stop(postgres.container)
}

func (postgres *Postgres) RemoveContainer() error {
	return removeContainer(postgres.container)
}

func newPostgres(id string) (*Postgres, error) {
	instance := &Postgres{container: id + "-postgres"}

	ip, err := inspectIP(instance.container)
	switch {
	case err != nil && lexec.IsExitStatus(err):
		err := instance.start()
		if err != nil {
			return nil, karma.Format(err, "start new postgres instance")
		}

	case err != nil && !lexec.IsExitStatus(err):
		return nil, karma.Format(err, "docker inspect container")

	default:
		// if we found ip it means it's already started/starting, so we ensure
		// it's running and ready for bitbucket
		waitService(ip, 5432, "postgres")
	}

	return instance, nil
}
