package main

import (
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/users"
)

func (suite *Suite) TestBitbucketUpgrade(params TestParams) {
	suite.UseBitbucket(params.BitbucketFrom, params.Cluster)
	suite.InstallAddon(params.Addon)

	var cases struct {
		public, personal struct {
			repo *stash.Repository
			pre  *external_hooks.Hook
			post *external_hooks.Hook
		}
	}

	{

		project := suite.CreateRandomProject()

		cases.public.repo = suite.CreateRandomRepository(project)

		context := suite.ExternalHooks().OnProject(project.Key)

		cases.public.pre, cases.public.post = suite.testBitbucketUpgrade_Before(
			project, cases.public.repo, context,
		)
	}

	{
		project := &stash.Project{
			Key: "~admin",
		}

		cases.personal.repo = suite.CreateRandomRepository(project)

		context := suite.ExternalHooks().OnProject(project.Key).
			OnRepository(cases.personal.repo.Slug)

		cases.personal.pre, cases.personal.post = suite.testBitbucketUpgrade_Before(
			project,
			cases.personal.repo,
			context,
		)
	}

	suite.UseBitbucket(params.BitbucketTo, params.Cluster)
	suite.RecordHookScripts()

	suite.testBitbucketUpgrade_After(
		cases.public.repo,
		cases.public.pre,
		cases.public.post,
	)

	suite.DetectHookScriptsLeak()
}

func (suite *Suite) testBitbucketUpgrade_Before(
	project *stash.Project,
	repo *stash.Repository,
	context *external_hooks.Context,
) (*external_hooks.Hook, *external_hooks.Hook) {
	pre := suite.ConfigureSampleHook_FailWithMessage(
		context.PreReceive(),
		`XXX`,
	)

	suite.WaitExternalHookEnabled(context.PreReceive())

	Assert_PushRejected(suite, repo, `XXX`)

	suite.DisableHook(pre)

	suite.WaitExternalHookDisabled(pre)

	Assert_PushDoesNotOutputMessages(suite, repo, `XXX`)

	post := suite.ConfigureSampleHook_FailWithMessage(
		context.PostReceive(),
		`YYY`,
	)

	suite.WaitExternalHookEnabled(context.PostReceive())

	Assert_PushOutputsMessages(suite, repo, `YYY`)

	suite.DisableHook(post)

	Assert_PushDoesNotOutputMessages(suite, repo, `YYY`)

	err := pre.Enable()
	suite.NoError(err, "enable pre-receive hook")

	err = post.Enable()
	suite.NoError(err, "enable post-receive hook")

	return pre, post
}

func (suite *Suite) testBitbucketUpgrade_After(
	repo *stash.Repository,
	pre, post *external_hooks.Hook,
) {
	pre.BitbucketURI = suite.Bitbucket().ConnectorURI(users.USER_ADMIN)
	post.BitbucketURI = suite.Bitbucket().ConnectorURI(users.USER_ADMIN)

	Assert_PushRejected(suite, repo, `XXX`)

	suite.DisableHook(pre)

	Assert_PushOutputsMessages(suite, repo, `YYY`)

	suite.DisableHook(post)

	Assert_PushDoesNotOutputMessages(suite, repo, `YYY`)
}
