package runner

import (
	"io/ioutil"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/git"
)

func (runner *Runner) GitClone(repository *stash.Repository) *git.Git {
	var href string

	for _, link := range repository.Links.Clones {
		if link.Name == "http" {
			href = link.HREF
			break
		}
	}

	runner.assert.NotEmpty(href, "git clone url is empty")

	dir, err := ioutil.TempDir(runner.run.dir, "repo.")
	runner.assert.NoError(err, "unable to create dir for repo")

	git, err := git.Clone(href, dir)
	runner.assert.NoError(err, "unable to clone git repo")

	return git
}
