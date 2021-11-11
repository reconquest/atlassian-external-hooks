package main

import (
	"fmt"
	"strings"

	"github.com/kovetskiy/stash"
)

func Assert_PushRejected(
	suite *Suite,
	repository *stash.Repository,
	messages ...string,
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

	for _, message := range messages {
		suite.Contains(
			string(stdout),
			"remote: "+message,
			"git push message should contain output from hook",
		)
	}
}

func Assert_PushOutputsMessages(
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

func Assert_PushDoesNotOutputMessages(
	suite *Suite,
	repository *stash.Repository,
	messages ...string,
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

	for _, message := range messages {
		suite.NotContains(
			string(stdout),
			fmt.Sprintf("remote: %s", message),
			"post-receive-hook should not contain output from post-receive hook",
		)
	}
}

func Assert_PushOutputsRefInfo(
	suite *Suite,
	repository *stash.Repository,
	_ ...string,
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

func Assert_MergeCheck_Callback(
	pullRequest *stash.PullRequest,
	suite *Suite,
	repository *stash.Repository,
	callback func(reply string),
) {
	service := suite.Bitbucket().Repositories(repository.Project.Key).
		PullRequests(repository.Slug)

	pullRequest, err := service.Get(pullRequest.ID)
	suite.NoError(err, "get pull request object")

	result, err := service.Merge(
		pullRequest.ID,
		pullRequest.Version,
	)
	suite.NoError(err, "get merge pull request result")

	suite.Equal(
		len(result.Errors),
		1,
		"no messages found in merge response",
	)
	suite.Equal(
		len(result.Errors[0].Vetoes),
		1,
		"no vetoes found in merge response",
	)
	suite.Equal(
		result.Errors[0].Vetoes[0].SummaryMessage,
		"external-merge-check-hook declined",
	)

	callback(result.Errors[0].Vetoes[0].DetailedMessage)
}

func Assert_MergeCheckOutputsMessages(
	pullRequest *stash.PullRequest,
	suite *Suite,
	repository *stash.Repository,
	messages ...string,
) {
	Assert_MergeCheck_Callback(
		pullRequest,
		suite,
		repository,
		func(reply string) {
			if len(messages) == 0 {
				suite.Equal("", reply)
			} else {
				for _, message := range messages {
					suite.Contains(
						reply,
						message,
						"merge check veto message must contain specified message",
					)
				}
			}
		},
	)
}

func Assert_MergeCheckPassed(
	pullRequest *stash.PullRequest,
	suite *Suite,
	repository *stash.Repository,
) {
	service := suite.Bitbucket().Repositories(repository.Project.Key).
		PullRequests(repository.Slug)

	pullRequest, err := service.Get(pullRequest.ID)
	suite.NoError(err, "get pull request object")

	result, err := service.Merge(
		pullRequest.ID,
		pullRequest.Version,
	)
	suite.NoError(err, "get merge pull request result")
	suite.Equal(
		len(result.Errors),
		0,
		"should be able to merge pull request",
	)
	suite.Equal(
		"MERGED",
		result.State,
		"pull request should be in merged state",
	)
}

func AssertWithPullRequest(
	pullRequest *stash.PullRequest,
	assert func(
		pullRequest *stash.PullRequest,
		suite *Suite,
		repository *stash.Repository,
		messages ...string,
	),
) HookTesterAssert {
	return func(
		suite *Suite,
		repository *stash.Repository,
		messages ...string,
	) {
		assert(pullRequest, suite, repository, messages...)
	}
}
