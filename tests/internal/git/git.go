package git

import (
	"github.com/reconquest/atlassian-external-hooks/tests/internal/exec"
	"github.com/reconquest/karma-go"
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
	return git.command("add", paths...)
}

func (git *Git) Commit(message string) error {
	return git.command("commit", "-m", message)
}

func (git *Git) Push() error {
	return git.command("push")
}

func (git *Git) command(command string, args ...string) error {
	args = append([]string{"-C", git.dir, command}, args...)

	return exec.New("git", args...).Run()
}
