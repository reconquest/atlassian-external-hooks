package database

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

type Database interface {
	Driver() string
	URL() string
	User() string
	Password() string
	Container() string
	Volume() string
	Stop() error
	RemoveContainer() error
}

func Start(kind string, id string) (Database, error) {
	switch kind {
	case "postgres":
		return newPostgres(id)
	case "mysql":
		return newMysql(id)
	case "oracle":
		return newOracle(id)
	}

	return nil, errors.New("unknown type of database: " + kind)
}

func stop(container string) error {
	return exec.New("docker", "stop", container).Run()
}

func removeContainer(container string) error {
	return exec.New("docker", "rm", container).Run()
}

func inspectIP(container string) (string, error) {
	stdout, _, err := exec.New(
		"docker",
		"inspect",
		"--type", "container",
		"-f",
		`{{range .NetworkSettings.Networks}}{{.IPAddress}}{{"\n"}}{{end}}`,
		container,
	).Output()
	if err != nil {
		return "", err
	}

	ips := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	if len(ips) == 0 {
		return "", karma.
			Describe("container", container).
			Format(
				err,
				"no ip addresses found on container",
			)
	}

	ip := ips[0]

	return ip, nil
}

func waitLog(container string, name string, substring string) error {
	execution := exec.New(
		"docker",
		"logs", "-f",
		container,
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return karma.Format(
			err,
			"get stdout pipe for docker logs",
		)
	}

	err = execution.Start()
	if err != nil {
		return karma.Format(
			err,
			"start docker logs",
		)
	}

	startedAt := time.Now()
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Second * 3)
		for {
			select {
			case <-ticker.C:
				log.Tracef(
					nil,
					"logs | waiting for %s to print %q | %s",
					name,
					substring,
					time.Since(startedAt),
				)

			case <-done:
				log.Tracef(
					nil,
					"%s started at %s took: %s",
					name,
					time.Since(startedAt),
				)
				return
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		text := scanner.Text()

		log.Tracef(nil, "{docker: %s} %s", container, text)

		if strings.Contains(text, substring) {
			break
		}
	}

	close(done)

	err = execution.Process().Signal(os.Interrupt)
	if err != nil {
		return karma.Format(err, "kill process after reading logs")
	}

	_ = execution.Wait()

	return nil
}

func waitService(ip string, port int, name string) {
	timeout := time.Millisecond * 150
	addr := fmt.Sprintf("%v:%v", ip, port)

	startedAt := time.Now()

	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Millisecond * 500)
		for {
			select {
			case <-ticker.C:
				log.Tracef(
					nil,
					"tcpcheck | waiting for %s on %s (timeout: %v) | %s",
					name,
					addr,
					timeout,
					time.Since(startedAt),
				)

			case <-done:
				log.Tracef(
					nil,
					"%s started at %s took: %s",
					name,
					addr,
					time.Since(startedAt),
				)
				return
			}
		}
	}()

	for {
		_, err := net.DialTimeout("tcp", addr, timeout)
		if err != nil {
			time.Sleep(time.Millisecond * 10)
			continue
		}

		break
	}

	close(done)
}
