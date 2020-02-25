package main

import (
	"fmt"

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
		fmt.Sprintf("remote: %s", message),
		"git push message should contain output from pre-receive hook",
	)

	//err = hook.Disable()
	//assert.NoError(err, "should be able to disable pre-receive hook")

	//stdout, err = git.Push()
	//assert.NoError(err, "git push should succeed")
	//assert.NotContains(
	//    string(stdout),
	//    "remote: external-pre-receive-hook declined",
	//    "pre-receive-hook should not decline push",
	//)
}

func Testcase_PushOutputsMessage(
	suite *Suite,
	repository *stash.Repository,
	message string,
) {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	stdout, err := git.Push()
	suite.NoError(err, "git push should succeed")
	suite.Contains(
		string(stdout),
		fmt.Sprintf("remote: %s", message),
		"git push message should contain output from post-receive hook",
	)
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
