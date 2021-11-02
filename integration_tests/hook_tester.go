package main

import (
	"fmt"
	"strconv"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/cog"
)

type HookTester struct {
	log        *cog.Logger
	hook       *external_hooks.Hook
	repository *stash.Repository
	suite      *Suite
	exit       int
}

type HookTesterAssert func(*Suite, *stash.Repository, ...string)

func NewHookTester(
	log *cog.Logger,
	hook *external_hooks.Hook,
	suite *Suite,
	repository *stash.Repository,
) *HookTester {
	return &HookTester{
		log:        log,
		hook:       hook,
		repository: repository,
		suite:      suite,
	}
}

func (tester *HookTester) WithExitCode(code int) *HookTester {
	tester.exit = code
	return tester
}

func (tester *HookTester) TestStdin(
	assert HookTesterAssert,
	preamble ...string,
) {
	tester.log.Infof(nil, ">> testing hook stdin")

	tester.suite.ConfigureSampleHook(
		tester.hook,
		HookOptions{WaitHookScripts: true},
		string(text(
			append(preamble, `cat`, fmt.Sprintf("exit %d", tester.exit))...,
		)),
	)

	assert(tester.suite, tester.repository)
}

func (tester *HookTester) TestArgs(
	assert HookTesterAssert,
) {
	tester.log.Infof(nil, ">> testing hook positional args")

	tester.suite.ConfigureSampleHook(
		tester.hook,
		HookOptions{WaitHookScripts: true},
		string(text(
			`printf "[%s]\n" "$@"`,
			fmt.Sprintf("exit %d", tester.exit),
		)),
		"arg-1",
		"arg-2",
		"multi\nline",
	)

	assert(
		tester.suite,
		tester.repository,
		"[arg-1]",
		"[arg-2]",
		"[multi",
		"line]",
	)
}

func (tester *HookTester) TestEnableDisable(
	assertEnabled, assertDisabled HookTesterAssert,
) {
	tester.log.Infof(nil, ">> testing hook enable/disable")

	const message = `XXX`

	tester.suite.ConfigureSampleHook_FailWithMessage(
		tester.hook,
		HookOptions{WaitHookScripts: true},
		message,
	)

	assertEnabled(tester.suite, tester.repository, message)

	err := tester.hook.Disable()
	tester.suite.NoError(err, "disable hook")

	err = tester.hook.Wait()
	tester.suite.NoError(err, "wait for disable hook")

	assertDisabled(tester.suite, tester.repository, message)
}

func (tester *HookTester) TestEnv(
	assert HookTesterAssert,
	name string,
	value string,
) {
	tester.log.Infof(nil, ">> testing hook env var $%s", name)

	tester.suite.ConfigureSampleHook(
		tester.hook,
		HookOptions{WaitHookScripts: true},
		string(text(
			fmt.Sprintf(`echo [$%s]`, name),
			fmt.Sprintf(`exit %d`, tester.exit),
		)),
	)

	assert(tester.suite, tester.repository, fmt.Sprintf("[%s]", value))
}

func (tester *HookTester) TestEnv_BB_HOOK_TRIGGER_ID(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_HOOK_TRIGGER_ID", value)
}

func (tester *HookTester) TestEnv_BB_HOOK_TYPE(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_HOOK_TYPE", value)
}

func (tester *HookTester) TestEnv_BB_IS_DRY_RUN(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_IS_DRY_RUN", value)
}

func (tester *HookTester) TestEnv_BB_PROJECT_KEY(
	assert HookTesterAssert,
) {
	tester.TestEnv(assert, "BB_PROJECT_KEY", tester.repository.Project.Key)
}

func (tester *HookTester) TestEnv_BB_REPO_SLUG(
	assert HookTesterAssert,
) {
	tester.TestEnv(assert, "BB_REPO_SLUG", tester.repository.Slug)
}

func (tester *HookTester) TestEnv_BB_REPO_IS_FORK(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_REPO_IS_FORK", value)
}

func (tester *HookTester) TestEnv_BB_REPO_IS_PUBLIC(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_REPO_IS_PUBLIC", value)
}

func (tester *HookTester) TestEnv_BB_BASE_URL(
	assert HookTesterAssert,
) {
	tester.TestEnv(assert, "BB_BASE_URL", tester.suite.Bitbucket().GetURI(""))
}

func (tester *HookTester) TestEnv_BB_REPO_CLONE_SSH(
	assert HookTesterAssert,
) {
	tester.TestEnv(
		assert,
		"BB_REPO_CLONE_SSH",
		tester.suite.Bitbucket().GetClonePathSSH(
			tester.repository.Project.Key,
			tester.repository.Slug,
		),
	)
}

func (tester *HookTester) TestEnv_BB_REPO_CLONE_HTTP(
	assert HookTesterAssert,
) {
	tester.TestEnv(
		assert,
		"BB_REPO_CLONE_HTTP",
		tester.suite.Bitbucket().GetClonePathHTTP(
			tester.repository.Project.Key,
			tester.repository.Slug,
		),
	)
}

func (tester *HookTester) TestEnv_BB_USER_NAME(
	assert HookTesterAssert,
) {
	tester.TestEnv(
		assert,
		"BB_USER_NAME",
		tester.suite.Bitbucket().GetOpts().AdminUser,
	)
}

func (tester *HookTester) TestEnv_BB_USER_EMAIL(
	assert HookTesterAssert,
) {
	tester.TestEnv(
		assert,
		"BB_USER_EMAIL",
		tester.suite.Bitbucket().GetOpts().AdminEmail,
	)
}

func (tester *HookTester) TestEnv_BB_USER_DISPLAY_NAME(
	assert HookTesterAssert,
) {
	tester.TestEnv(
		assert,
		"BB_USER_DISPLAY_NAME",
		tester.suite.Bitbucket().GetOpts().AdminUser,
	)
}

func (tester *HookTester) TestEnv_BB_USER_PERMISSION(
	assert HookTesterAssert,
) {
	tester.TestEnv(assert, "BB_USER_PERMISSION", "SYS_ADMIN")
}

func (tester *HookTester) TestEnv_BB_FROM_PROJECT_KEY(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_FROM_PROJECT_KEY",
		pullRequest.FromRef.Repository.Project.Key,
	)
}

func (tester *HookTester) TestEnv_BB_FROM_REPO_SLUG(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_FROM_REPO_SLUG",
		pullRequest.FromRef.Repository.Slug,
	)
}

func (tester *HookTester) TestEnv_BB_FROM_REPO_IS_FORK(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_FROM_REPO_IS_FORK", value)
}

func (tester *HookTester) TestEnv_BB_FROM_REPO_IS_PUBLIC(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_FROM_REPO_IS_PUBLIC", value)
}

func (tester *HookTester) TestEnv_BB_FROM_REF(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(assert, "BB_FROM_REF", pullRequest.FromRef.DisplayID)
}

func (tester *HookTester) TestEnv_BB_FROM_HASH(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(assert, "BB_FROM_HASH", pullRequest.FromRef.LatestCommit)
}

func (tester *HookTester) TestEnv_BB_TO_REF(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(assert, "BB_TO_REF", pullRequest.ToRef.DisplayID)
}

func (tester *HookTester) TestEnv_BB_TO_HASH(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(assert, "BB_TO_HASH", pullRequest.ToRef.LatestCommit)
}

func (tester *HookTester) TestEnv_BB_FROM_REPO_CLONE_HTTP(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_FROM_REPO_CLONE_HTTP",
		tester.suite.Bitbucket().GetClonePathHTTP(
			pullRequest.FromRef.Repository.Project.Key,
			pullRequest.FromRef.Repository.Slug,
		),
	)
}

func (tester *HookTester) TestEnv_BB_FROM_REPO_CLONE_SSH(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_FROM_REPO_CLONE_SSH",
		tester.suite.Bitbucket().GetClonePathSSH(
			pullRequest.FromRef.Repository.Project.Key,
			pullRequest.FromRef.Repository.Slug,
		),
	)
}

func (tester *HookTester) TestEnv_BB_MERGE_IS_CROSS_REPO(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_MERGE_IS_CROSS_REPO", value)
}

func (tester *HookTester) TestEnv_BB_MERGE_STRATEGY_ID(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(assert, "BB_MERGE_STRATEGY_ID", value)
}

func (tester *HookTester) TestEnv_BB_PULL_REQUEST_ID(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(assert, "BB_PULL_REQUEST_ID", strconv.Itoa(pullRequest.ID))
}

func (tester *HookTester) TestEnv_BB_PULL_REQUEST_AUTHOR_USER_NAME(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_PULL_REQUEST_AUTHOR_USER_NAME",
		pullRequest.Author.User.Name,
	)
}

func (tester *HookTester) TestEnv_BB_PULL_REQUEST_AUTHOR_USER_DISPLAY_NAME(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_PULL_REQUEST_AUTHOR_USER_DISPLAY_NAME",
		pullRequest.Author.User.DisplayName,
	)
}

func (tester *HookTester) TestEnv_BB_PULL_REQUEST_AUTHOR_USER_EMAIL(
	assert HookTesterAssert,
	pullRequest *stash.PullRequest,
) {
	tester.TestEnv(
		assert,
		"BB_PULL_REQUEST_AUTHOR_USER_EMAIL",
		pullRequest.Author.User.Email,
	)
}

func (tester *HookTester) TestEnv_BB_PULL_REQUEST_AUTHOR_USER_PERMISION(
	assert HookTesterAssert,
	value string,
) {
	tester.TestEnv(
		assert,
		"BB_PULL_REQUEST_AUTHOR_USER_PERMISSION",
		value,
	)
}
