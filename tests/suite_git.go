package main

import (
	"io/ioutil"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/tests/internal/git"
)

func (suite *Suite) GitClone(repository *stash.Repository) *git.Git {
	var href string

	for _, link := range repository.Links.Clones {
		if link.Name == "http" {
			href = link.HREF
			break
		}
	}

	suite.NotEmpty(href, "git clone url is empty")

	dir, err := ioutil.TempDir(suite.run.dir, "repo.")
	suite.NoError(err, "unable to create dir for repo")

	git, err := git.Clone(href, dir)
	suite.NoError(err, "unable to clone git repo")

	return git
}
