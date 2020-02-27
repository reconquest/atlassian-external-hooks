package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
	"github.com/stretchr/testify/assert"
)

type Suite struct {
	*runner.Runner
	*assert.Assertions
}

type TestParams map[string]interface{}

type Addon struct {
	Version string
	Path    string
}

func NewSuite() *Suite {
	return &Suite{}
}

func (suite *Suite) WithParams(
	params TestParams,
	tests ...func(TestParams),
) runner.Suite {
	return func(run *runner.Runner, assert *assert.Assertions) {
		suite.Runner = run
		suite.Assertions = assert

		for _, test := range tests {
			name := runtime.FuncForPC(reflect.ValueOf(test).Pointer()).Name()
			name = strings.TrimPrefix(name, "main.(*Suite).")
			name = strings.TrimSuffix(name, "-fm")

			log.Infof(
				karma.Describe("params", params),
				"{test} running %s",
				name,
			)

			test(params)
		}
	}
}

func (suite *Suite) ConfigureHook(
	key string,
	context *external_hooks.Context,
	settings *external_hooks.Settings,
	script []byte,
) *external_hooks.Hook {
	path :=
		filepath.Join("shared", "external-hooks", settings.Executable)

	log.Debugf(
		karma.Describe("script", "\n"+string(script)),
		"{hook} writing hook script %q",
		path,
	)

	err := suite.Bitbucket().WriteFile(path, append(script, '\n'), 0777)
	suite.NoError(err, "should be able to write hook script to container")

	var hook *external_hooks.Hook

	switch key {
	case external_hooks.HOOK_KEY_PRE_RECEIVE:
		hook = context.PreReceive(settings)
	case external_hooks.HOOK_KEY_POST_RECEIVE:
		hook = context.PostReceive(settings)
	case external_hooks.HOOK_KEY_MERGE_CHECK:
		hook = context.MergeCheck(settings)
	}

	err = hook.Configure()
	suite.NoError(err, "should be able to configure hook")

	err = hook.Enable()
	suite.NoError(err, "should be able to enable hook")

	return hook
}

func (suite *Suite) ConfigurePreReceiveHook(
	context *external_hooks.Context,
	settings *external_hooks.Settings,
	script []byte,
) *external_hooks.Hook {
	return suite.ConfigureHook(
		external_hooks.HOOK_KEY_PRE_RECEIVE,
		context,
		settings,
		script,
	)
}

func (suite *Suite) ConfigurePostReceiveHook(
	context *external_hooks.Context,
	settings *external_hooks.Settings,
	script []byte,
) *external_hooks.Hook {
	return suite.ConfigureHook(
		external_hooks.HOOK_KEY_POST_RECEIVE,
		context,
		settings,
		script,
	)
}

func (suite *Suite) ConfigureMergeCheckHook(
	context *external_hooks.Context,
	settings *external_hooks.Settings,
	script []byte,
) *external_hooks.Hook {
	return suite.ConfigureHook(
		external_hooks.HOOK_KEY_MERGE_CHECK,
		context,
		settings,
		script,
	)
}

func (suite *Suite) ExternalHooks() *external_hooks.Addon {
	return &external_hooks.Addon{
		BitbucketURI: suite.Bitbucket().GetConnectorURI(),
	}
}

func (suite *Suite) CreateRandomProject() *stash.Project {
	project, err := suite.Bitbucket().Projects().
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create project")

	return project
}

func (suite *Suite) CreateRandomRepository(
	project *stash.Project,
) *stash.Repository {
	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(4))
	suite.NoError(err, "unable to create repository")

	return repository
}

func (suite *Suite) CreateRandomPullRequest(
	project *stash.Project,
	repository *stash.Repository,
) *stash.PullRequest {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	_, err := git.Push()
	suite.NoError(err, "unable to git push into master")

	branch := suite.GitCreateRandomBranch(git)

	suite.GitCommitRandomFile(git)

	_, err = git.Push("origin", branch)
	suite.NoErrorf(err, "unable to git push into branch %s", branch)

	pullRequest, err := suite.Bitbucket().Repositories(project.Key).
		PullRequests(repository.Slug).
		Create(
			"pr."+lojban.GetRandomID(8),
			lojban.GetRandomID(20),
			branch,
			"master",
		)
	suite.NoError(err, "unable to create pull request")

	return pullRequest
}

func (suite *Suite) CreateSamplePreReceiveHook_FailWithMessage(
	context *external_hooks.Context,
	message string,
) *external_hooks.Hook {
	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`pre.fail.sh`)

	return suite.ConfigurePreReceiveHook(context, settings, text(
		`#!/bin/bash`,
		fmt.Sprintf(`echo %s`, message),
		`exit 1`,
	))
}

func (suite *Suite) CreateSamplePostReceiveHook_FailWithMessage(
	context *external_hooks.Context,
	message string,
) *external_hooks.Hook {
	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`post.fail.sh`)

	return suite.ConfigurePostReceiveHook(context, settings, text(
		`#!/bin/bash`,
		fmt.Sprintf(`echo %s`, message),
		`exit 1`,
	))
}

func (suite *Suite) CreateSampleMergeCheckHook_FailWithMessage(
	context *external_hooks.Context,
	message string,
) *external_hooks.Hook {
	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`merge.fail.sh`)

	return suite.ConfigureMergeCheckHook(context, settings, text(
		`#!/bin/bash`,
		fmt.Sprintf(`echo %s`, message),
		`exit 1`,
	))
}

//var (
//    debugHookScript = text(
//        `#!/bin/bash`,
//        `printf "%s\n" "$@"`,
//        `cat`,
//        `echo BB_HOOK_TRIGGER_ID=$BB_HOOK_TRIGGER_ID`,
//        `echo BB_HOOK_TYPE=$BB_HOOK_TYPE`,
//        `echo BB_IS_DRY_RUN=$BB_IS_DRY_RUN`,
//        `echo BB_PROJECT_KEY=$BB_PROJECT_KEY`,
//        `echo BB_REPO_SLUG=$BB_REPO_SLUG`,
//        `echo BB_REPO_IS_FORK=$BB_REPO_IS_FORK`,
//        `echo BB_REPO_IS_PUBLIC=$BB_REPO_IS_PUBLIC`,
//        `echo BB_BASE_URL=$BB_BASE_URL`,
//        `echo BB_REPO_CLONE_SSH=$BB_REPO_CLONE_SSH`,
//        `echo BB_REPO_CLONE_HTTP=$BB_REPO_CLONE_HTTP`,
//        `echo BB_USER_NAME=$BB_USER_NAME`,
//        `echo BB_USER_DISPLAY_NAME=$BB_USER_DISPLAY_NAME`,
//        `echo BB_USER_EMAIL=$BB_USER_EMAIL`,
//        `echo BB_USER_PERMISSION=$BB_USER_PERMISSION`,
//    )
//)

func (suite *Suite) CreateSamplePreReceiveHook_Debug(
	context *external_hooks.Context,
	script string,
	args ...string,
) *external_hooks.Hook {
	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`pre.debug.sh`).
		WithArgs(args...)

	return suite.ConfigurePreReceiveHook(
		context,
		settings,
		text(
			`#!/bin/bash`,
			script,
		),
	)
}

func (suite *Suite) CreateSamplePostReceiveHook_Debug(
	context *external_hooks.Context,
	script string,
	args ...string,
) *external_hooks.Hook {
	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`post.debug.sh`).
		WithArgs(args...)

	return suite.ConfigurePreReceiveHook(
		context,
		settings,
		text(
			`#!/bin/bash`,
			script,
		),
	)
}

func (suite *Suite) InstallAddon(addon Addon) string {
	err := suite.enableAddonLogger(
		"com.ngs.stash.externalhooks",
		"debug",
	)
	suite.NoError(err, "unable to enable addon logger")

	waiter := suite.Bitbucket().WaitLogEntry(func(line string) bool {
		if strings.Contains(line, "Finished job for creating HookScripts") {
			return true
		}

		return false
	})

	key := suite.Runner.InstallAddon(addon.Version, addon.Path)

	log.Debugf(nil, "{add-on} waiting for add-on startup process to finish")

	waiter.Wait()

	return key
}

func (suite *Suite) enableAddonLogger(key string, level string) error {
	request, err := http.NewRequest(
		http.MethodPut,
		fmt.Sprintf(
			"%s/rest/api/latest/logs/logger/%s/%s",
			suite.Bitbucket().GetConnectorURI(),
			key,
			level,
		),
		nil,
	)
	if err != nil {
		karma.Format(
			err,
			"unable to construct request for logger endpoint",
		)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return karma.Format(
			err,
			"unable to send put request on logger endpoint",
		)
	}

	if response.StatusCode != http.StatusNoContent {
		return karma.
			Describe("expected_code", http.StatusNoContent).
			Describe("response_code", response.StatusCode).
			Format(
				err,
				"unexpected status code from logger endpoint",
			)
	}

	return nil
}
