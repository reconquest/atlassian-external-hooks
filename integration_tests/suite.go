package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"

	"github.com/coreos/go-semver/semver"
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

	hookScripts []string
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
	hook *external_hooks.Hook,
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

	err = hook.Configure(settings)
	suite.NoError(err, "should be able to configure hook")

	err = hook.Enable()
	suite.NoError(err, "should be able to enable hook")

	return hook
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

func (suite *Suite) ConfigureSampleHook_FailWithMessage(
	hook *external_hooks.Hook,
	message string,
) *external_hooks.Hook {
	return suite.ConfigureSampleHook(
		hook,
		string(text(
			fmt.Sprintf(`echo %s`, message),
			`exit 1`,
		)),
	)
}

func (suite *Suite) ConfigureSampleHook(
	hook *external_hooks.Hook,
	script string,
	args ...string,
) *external_hooks.Hook {
	settings := external_hooks.NewSettings().
		UseSafePath(true).
		WithExecutable(`hook.` + lojban.GetRandomID(5)).
		WithArgs(args...)

	return suite.ConfigureHook(
		hook,
		settings,
		text(
			`#!/bin/bash`,
			script,
		),
	)
}

func (suite *Suite) InstallAddon(addon Addon) string {
	var (
		v       = *semver.New(addon.Version)
		v10_0_0 = *semver.New("10.0.0")
		v9_1_0  = *semver.New("9.1.0")
	)

	waiter := suite.Bitbucket().WaitLogEntry(func(line string) bool {
		switch {
		case v.Compare(v10_0_0) >= 0 &&
			strings.Contains(line, "Finished job for creating HookScripts"):
			return true
		case v.Compare(v10_0_0) < 0 && v.Compare(v9_1_0) >= 0 &&
			strings.Contains(line, "HookScripts created successfully"):
			return true
		default:
			return false
		}
	})

	key := suite.Runner.InstallAddon(addon.Version, addon.Path)

	log.Debugf(nil, "{add-on} waiting for add-on startup process to finish")

	waiter.Wait()

	return key
}

func (suite *Suite) DisableHook(hook *external_hooks.Hook) {
	// XXX: only for BB>6.2.0
	waiter := suite.Bitbucket().WaitLogEntry(func(line string) bool {
		switch {
		case regexp.MustCompile(
			`ExternalHookScript deleting .* hook script`,
		).MatchString(line):
			return true
		default:
			return false
		}
	})

	err := hook.Disable()
	suite.NoError(err, "should be able to disable hook")

	log.Debugf(nil, "{add-on} waiting for hook script to be deleted by bitbucket")

	waiter.Wait()
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

func (suite *Suite) RecordHookScripts() {
	var err error
	suite.hookScripts, err = suite.Bitbucket().
		ListFiles("shared/config/hook-scripts/")
	suite.NoError(err, "should be able to list existing hook scripts")

	log.Debugf(
		karma.Describe("scripts", strings.Join(suite.hookScripts, ", ")),
		"{leak detector} found %d currently registered hook scripts",
		len(suite.hookScripts),
	)
}

func (suite *Suite) DetectHookScriptsLeak() {
	current, err := suite.Bitbucket().
		ListFiles("shared/config/hook-scripts/")
	suite.NoError(err, "should be able to list current hook scripts")

	index := map[string]bool{}

	for _, name := range suite.hookScripts {
		index[name] = true
	}

	leak := []string{}

	for _, name := range current {
		if !index[name] {
			leak = append(leak, name)
		}
	}

	if len(leak) > 0 {
		suite.Empty(leak, "found leaking hook scripts")
	} else {
		log.Debugf(
			nil,
			"{leak detector} no hook scripts leak detected",
		)
	}

}
