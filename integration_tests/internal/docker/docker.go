package docker

import (
	"archive/tar"
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

type Logs struct {
	cond *sync.Cond

	lines []string
}

type LogWaiter interface {
	Await() bool

	Wait(
		failer func(failureMessage string, msgAndArgs ...interface{}) bool,
		resource string,
		state string,
	)
}

type logWaiter struct {
	duration time.Duration
	group    *sync.WaitGroup
	ctx      context.Context
}

func (waiter *logWaiter) Wait(
	failer func(failureMessage string, msgAndArgs ...interface{}) bool,
	resource string,
	state string,
) {
	done := make(chan struct{})
	go func() {
		waiter.group.Wait()
		close(done)
	}()

	select {
	case <-time.After(waiter.duration):
		failer(
			resource+" should be "+state+" but no such log message occurred",
			"duration: %s",
			waiter.duration,
		)
	case <-done:
	case <-waiter.ctx.Done():
	}
}

func (waiter *logWaiter) Await() bool {
	done := make(chan struct{})
	go func() {
		waiter.group.Wait()
		close(done)
	}()
	select {
	case <-time.After(waiter.duration):
		return false
	case <-done:
		return true
	case <-waiter.ctx.Done():
		return false
	}
}

func WaitLog(
	ctx context.Context,
	logs *Logs,
	fn func(string) bool,
	duration time.Duration,
) LogWaiter {
	waiter := sync.WaitGroup{}
	waiter.Add(1)

	cursor := 0
	prev := 0

	go func() {
		defer waiter.Done()

		for {
			logs.cond.L.Lock()

			for {
				now := len(logs.lines)
				if now == prev {
					logs.cond.Wait()
					continue
				}

				prev = now
				break
			}

			// FlushLogs() was called before
			if cursor > len(logs.lines)-1 {
				cursor = 0
			}

			lines := logs.lines[cursor:]
			logs.cond.L.Unlock()

			cursor += len(lines)

			for _, line := range lines {
				if fn(line) {
					return
				}
			}

			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}()

	return &logWaiter{
		ctx:      ctx,
		group:    &waiter,
		duration: duration,
	}
}

type ReadOpts struct {
	Container string
	Trace     bool
	Tail      int
}

func ReadLogs(opts ReadOpts) (*Logs, error) {
	log.Debugf(
		nil,
		"{bitbucket} starting log reader for container %q",
		opts.Container,
	)

	execution := exec.New(
		"docker",
		"logs", "-f",
		"--tail", strconv.Itoa(opts.Tail),
		opts.Container,
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, karma.Format(
			err,
			"get stdout pipe for docker logs",
		)
	}

	err = execution.Start()
	if err != nil {
		return nil, karma.Format(
			err,
			"start docker logs",
		)
	}

	logs := &Logs{
		cond: sync.NewCond(&sync.Mutex{}),
	}

	go func() {
		scanner := bufio.NewScanner(stdout)

		for scanner.Scan() {
			logs.cond.L.Lock()

			logs.lines = append(logs.lines, scanner.Text())

			logs.cond.Broadcast()

			logs.cond.L.Unlock()

			if opts.Trace {
				log.Tracef(
					nil, "{%s} log | %s",
					opts.Container,
					scanner.Text(),
				)
			}
		}

		if err := scanner.Err(); err != nil {
			log.Errorf(nil, "error while reading bitbucket node logs")
		}
	}()

	return logs, nil
}

func (logs *Logs) Flush() {
	logs.cond.L.Lock()
	logs.lines = []string{}
	logs.cond.L.Unlock()
}

func WriteFile(
	container string,
	dir string,
	path string,
	content []byte,
	mode os.FileMode,
) error {
	context := karma.
		Describe("dir", dir).
		Describe("path", path).
		Describe("container", container)

	err := exec.NewContext(
		context,
		"docker", "exec", container,
		"mkdir", "-p", filepath.Join(dir, filepath.Dir(path)),
	).Run()
	if err != nil {
		return context.Format(err, "mkdir")
	}

	execution := exec.NewContext(
		context,
		"docker", "cp", "-", fmt.Sprintf("%s:%s", container, dir),
	)

	err = execution.Start()
	if err != nil {
		return context.Format(
			err,
			"start docker cp",
		)
	}

	stdin := execution.GetStdin()

	writer := tar.NewWriter(stdin)

	err = writer.WriteHeader(&tar.Header{
		Name: path,
		Mode: int64(mode),
		Size: int64(len(content)),
	})
	if err != nil {
		return context.Format(
			err,
			"write file header",
		)
	}

	_, err = writer.Write(content)
	if err != nil {
		return context.Format(
			err,
			"write file contents",
		)
	}

	err = writer.Close()
	if err != nil {
		return context.Format(
			err,
			"close file",
		)
	}

	err = stdin.Close()
	if err != nil {
		return context.Format(
			err,
			"close docker cp stdin",
		)
	}

	err = execution.Wait()
	if err != nil {
		return context.Format(
			err,
			"complete docker cp",
		)
	}

	return nil
}

func ReadFile(container, path string) (string, error) {
	execution := exec.New(
		"docker", "exec", container, "cat", path,
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return "", karma.Format(
			err,
			"get stdout pipe for docker exec",
		)
	}

	err = execution.Start()
	if err != nil {
		return "", karma.Format(
			err,
			"start docker cp",
		)
	}

	data, err := ioutil.ReadAll(stdout)
	if err != nil {
		return "", err
	}

	return string(data), execution.Wait()
}
