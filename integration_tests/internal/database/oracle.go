package database

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

const (
	ORACLE_MESSAGE_READY_TO_USE = "DATABASE IS READY TO USE"
)

type Oracle struct {
	container string
}

func (oracle *Oracle) Driver() string {
	return "oracle.jdbc.OracleDriver"
}

func (oracle *Oracle) URL() string {
	return "jdbc:oracle:thin:@//" + oracle.container + ":1521/XE"
}

func (oracle *Oracle) Container() string {
	return oracle.container
}

func (oracle *Oracle) User() string {
	return "SYSTEM"
}

func (oracle *Oracle) Password() string {
	return "oracle"
}

func (oracle *Oracle) Volume() string {
	return oracle.container
}

func (oracle *Oracle) Stop() error {
	return stop(oracle.container)
}

func (oracle *Oracle) RemoveContainer() error {
	return removeContainer(oracle.container)
}

func (oracle *Oracle) start() error {
	execution := exec.New(
		"docker", "run",
		"--name", oracle.container,
		"-d",
		"-v", oracle.container+":/u01/app/oracle/oradata",
		"-e", "ORACLE_PWD=oracle",
		"--shm-size", "2147483648",
		"registry.reconquest.io/oracle/database:11.2.0.2-xe",
	)
	err := execution.Run()
	if err != nil {
		return karma.Format(err, "start oracle container")
	}

	err = waitLog(oracle.container, "oracle", ORACLE_MESSAGE_READY_TO_USE)
	if err != nil {
		return err
	}

	return nil
}

func newOracle(id string) (*Oracle, error) {
	instance := &Oracle{container: id + "-oracle"}

	_, err := inspectIP(instance.container)
	switch {
	case err != nil && lexec.IsExitStatus(err):
		err := instance.start()
		if err != nil {
			return nil, karma.Format(err, "start new oracle instance")
		}

	case err != nil && !lexec.IsExitStatus(err):
		return nil, karma.Format(err, "docker inspect container")

	default:
		// if we found ip it means it's already started/starting, so we ensure
		// it's running and ready for bitbucket
		err = waitLog(instance.container, "oracle", ORACLE_MESSAGE_READY_TO_USE)
		if err != nil {
			return nil, err
		}
	}

	return instance, nil
}
