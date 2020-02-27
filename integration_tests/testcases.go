package main

import (
	"fmt"
	"strings"

	"github.com/kovetskiy/stash"
)

func Testcase_PushRejected(
	suite *Suite,
	repository *stash.Repository,
	message string,
) {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	stdout, err := git.Push()
	suite.Error(err, "git push should fail")
	suite.Contains(
		string(stdout),
		"remote: external-pre-receive-hook declined",
		"pre-receive-hook should decline push",
	)
	suite.Contains(
		string(stdout),
		"remote rejected",
		"bitbicket should reject push",
	)
	suite.Contains(
		string(stdout),
		"remote: "+strings.ReplaceAll(message, "\n", "\nremote: "),
		"git push message should contain output from hook",
	)
}

func Testcase_PushOutputsMessages(
	suite *Suite,
	repository *stash.Repository,
	messages ...string,
) {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	stdout, err := git.Push()
	suite.NoError(err, "git push should succeed")

	for _, message := range messages {
		suite.Contains(
			string(stdout),
			"remote: "+message,
			"git push message should contain output from hook",
		)
	}
}

func Testcase_PushDoesNotOutputMessage(
	suite *Suite,
	repository *stash.Repository,
	message string,
) {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	stdout, err := git.Push()
	suite.NoError(err, "git push should succeed")
	suite.NotContains(
		string(stdout),
		"remote: external-post-receive-hook declined",
		"post-receive-hook should not decline push",
	)
	suite.NotContains(
		string(stdout),
		fmt.Sprintf("remote: %s", message),
		"post-receive-hook should not contain output from post-receive hook",
	)
}

func Testcase_PushOutputsRefInfo(
	suite *Suite,
	repository *stash.Repository,
) {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	revs, err := git.RevList("HEAD")
	suite.NoError(err, "git rev-list should list available revisions")
	suite.GreaterOrEqual(
		len(revs),
		1,
		"git rev-list should contain at least 1 revision",
	)

	if len(revs) == 1 {
		revs = append(revs, strings.Repeat("0", 40))
	}

	stdout, err := git.Push()
	suite.NoError(err, "git push should succeed")

	suite.Contains(
		string(stdout),
		fmt.Sprintf(
			"remote: %s %s %s",
			revs[1],
			revs[0],
			"refs/heads/master",
		),
	)
}
