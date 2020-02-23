package git

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

type Git struct {
	dir string
}

func Clone(href string, path string) (*Git, error) {
	err := exec.New("git", "clone", href, path).Run()
	if err != nil {
		return nil, karma.Format(
			err,
			"unable to run git clone",
		)
	}

	git := &Git{
		dir: path,
	}

	return git, nil
}

func (git *Git) GetWorkDir() string {
	return git.dir
}

func (git *Git) Add(paths ...string) error {
	return git.command("add", paths...).Run()
}

func (git *Git) Commit(message string) error {
	return git.command("commit", "-m", message).Run()
}

func (git *Git) Push() (string, error) {
	_, stderr, err := git.command("push").Output()
	return string(stderr), err
}

func (git *Git) command(command string, args ...string) *lexec.Execution {
	args = append([]string{"-C", git.dir, command}, args...)

	return exec.New("git", args...)
}
