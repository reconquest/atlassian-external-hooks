package runner

import (
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/git"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/users"
)

func (runner *Runner) GitClone(repository *stash.Repository) *git.Git {
	var href string

	for _, link := range repository.Links.Clones {
		if link.Name == "http" {
			href = link.HREF
			break
		}
	}

	privateURL, err := url.Parse(href)
	runner.assert.NoError(err, "parse git repository uri")

	publicURL, err := url.Parse(
		runner.Bitbucket().ConnectorURI(users.USER_ADMIN),
	)
	runner.assert.NoError(err, "parse bitbucket connector uri")

	publicURL.Path = privateURL.Path

	runner.assert.NotEmpty(href, "git clone url is empty")

	dir, err := ioutil.TempDir(runner.run.workdir, "repo.")
	runner.assert.NoError(err, "create dir for repo")

	git, err := git.Clone(publicURL.String(), dir)
	runner.assert.NoError(err, "clone git repo")

	return git
}

func (runner *Runner) GitCreateRandomBranch(git *git.Git) string {
	name := "branch." + lojban.GetRandomID(6)

	err := git.Branch(name)
	runner.assert.NoError(err, "create random branch")

	return name
}

func (runner *Runner) GitCommitRandomFile(git *git.Git) {
	err := ioutil.WriteFile(
		filepath.Join(git.GetWorkDir(), "post."+lojban.GetRandomID(6)),
		[]byte("file."+lojban.GetRandomID(10)),
		0666,
	)
	runner.assert.NoError(err, "should be able to write file in git repo")

	err = git.Add(".")
	runner.assert.NoError(err, "should be able to add file to git repo")

	err = git.Commit("commit." + lojban.GetRandomID(8))
	runner.assert.NoError(err, "should be able to commit file to git repo")
}
