package database

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

type Mysql struct {
	container string
}

func (mysql *Mysql) Driver() string {
	return "com.mysql.jdbc.Driver"
}

func (mysql *Mysql) URL() string {
	return "jdbc:mysql://" + mysql.container + ":3306/bitbucket"
}

func (mysql *Mysql) User() string {
	return "bitbucket"
}

func (mysql *Mysql) Password() string {
	return "bitbucket"
}

func (mysql *Mysql) Container() string {
	return mysql.container
}

func (mysql *Mysql) Volume() string {
	return mysql.container
}

func (mysql *Mysql) start() error {
	execution := exec.New(
		"docker",
		"run",
		"--name", mysql.container,
		"-d",
		"-v", mysql.container+":/var/lib/mysql",
		"-e", "MYSQL_ROOT_PASSWORD=bitbucketroot",
		"-e", "MYSQL_DATABASE=bitbucket",
		"-e", "MYSQL_USER=bitbucket",
		"-e", "MYSQL_PASSWORD=bitbucket",
		"mysql:latest",
		"--character-set-server=utf8mb4",
		"--collation-server=utf8mb4_unicode_ci",
	)
	err := execution.Run()
	if err != nil {
		return karma.Format(err, "start mysql container")
	}

	ip, err := inspectIP(mysql.container)
	if err != nil {
		return karma.Format(err, "inspect ip address")
	}

	waitService(ip, 3306, "mysql")

	return nil
}

func (mysql *Mysql) Stop() error {
	return stop(mysql.container)
}

func (mysql *Mysql) RemoveContainer() error {
	return removeContainer(mysql.container)
}

func newMysql(id string) (*Mysql, error) {
	instance := &Mysql{container: id + "-mysql"}

	ip, err := inspectIP(instance.container)
	switch {
	case err != nil && lexec.IsExitStatus(err):
		err := instance.start()
		if err != nil {
			return nil, karma.Format(err, "start new mysql instance")
		}

	case err != nil && !lexec.IsExitStatus(err):
		return nil, karma.Format(err, "docker inspect container")

	default:
		// if we found ip it means it's already started/starting, so we ensure
		// it's running and ready for bitbucket
		waitService(ip, 3306, "mysql")
	}

	return instance, nil
}
