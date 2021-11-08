package main

import (
	"context"
	"fmt"
	"math/rand"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/status"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
	"github.com/stretchr/testify/assert"
)

type SuiteMode int

const (
	ModeRun  SuiteMode = 1
	ModeList SuiteMode = 2
)

type Suite struct {
	*runner.Runner
	*assert.Assertions

	SuiteOpts

	hookScripts []HookScript
}

type HookScript struct {
	ID  string
	Tag string
}

type (
	TestParams map[string]interface{}
	Addon      struct {
		Version string
		Path    string
	}
)

type Filter struct {
	upgrade   bool
	reproduce bool
	glob      string
}

type SuiteOpts struct {
	baseBitbucket string
	randomize     bool
	mode          SuiteMode
	filter        Filter
	skipUntil     string
}

func NewSuite(opts SuiteOpts) *Suite {
	return &Suite{SuiteOpts: opts}
}

func getSuiteName(x interface{}) string {
	name := runtime.FuncForPC(reflect.ValueOf(x).Pointer()).Name()
	name = strings.TrimPrefix(name, "main.(*Suite).")
	name = strings.TrimSuffix(name, "-fm")
	return name
}

func (suite *Suite) WithParams(
	params TestParams,
	tests ...func(TestParams),
) runner.Suite {
	toRun := []func(TestParams){}
	for _, test := range tests {
		name := getSuiteName(test)

		if suite.filter.glob != "" {
			matched, err := regexp.MatchString(suite.filter.glob, name)
			if err != nil {
				log.Fatalf(err, "invalid regexp: %s", suite.filter.glob)
			}

			if !matched {
				continue
			}
		}

		if !suite.filter.upgrade {
			version, ok := params["bitbucket"]
			if !ok {
				version, _ = params["bitbucket_to"]
			}

			if version != suite.baseBitbucket {
				log.Debugf(
					nil,
					"{test} skip %s because --no-upgrade specified",
					name,
				)
				continue
			}
		}

		if !suite.filter.reproduce &&
			strings.HasSuffix(name, "_Reproduced") {
			log.Debugf(
				nil,
				"{test} skip %s because --no-reproduce specified",
				name,
			)
			continue
		}

		if suite.mode == ModeList {
			fmt.Println(name)
			continue
		}

		toRun = append(toRun, test)
	}

	return runner.Suite{
		Size: len(toRun),
		Run: func(run *runner.Runner, assert *assert.Assertions) {
			suite.Runner = run
			suite.Assertions = assert

			if suite.randomize {
				rand.Shuffle(
					len(tests),
					func(i, j int) { tests[i], tests[j] = tests[j], tests[i] },
				)
			}

			skippingUntil := suite.skipUntil != ""

			for _, test := range toRun {
				name := getSuiteName(test)

				if skippingUntil {
					matched, err := regexp.MatchString(suite.skipUntil, name)
					suite.NoError(err, "skip until regexp")

					if matched {
						skippingUntil = false
					} else {
						continue
					}
				}

				startedAt := time.Now()

				status.SetCurrentTest(name)

				log.Infof(
					karma.Describe("params", params),
					"{test} running %s",
					name,
				)

				var thrown chan bool
				var stop func()

				checkException := !strings.HasSuffix(name, "_Reproduced")

				if checkException {
					thrown, stop = suite.watchException()
				}

				test(params)

				if checkException {
					stop()

					if <-thrown {
						suite.FailNow(
							"got a java exception in logs",
							"testcase: %s",
							name,
						)
						break
					}
				}

				suite.Bitbucket().FlushLogs(
					suite.Bitbucket().StacktraceLogs(),
				)
				suite.Bitbucket().FlushLogs(suite.Bitbucket().TestcaseLogs())

				finishedAt := time.Now()
				took := finishedAt.Sub(startedAt)

				log.Infof(
					karma.Describe("took", took.Milliseconds()),
					"{test} %s finished",
					name,
				)

				status.AddDone()
				status.SetLastTest(name)
				status.SetLastDuration(took)
			}
		},
	}
}

func (suite *Suite) watchException() (result chan bool, stop func()) {
	ctx, stop := context.WithCancel(context.Background())

	result = make(chan bool)

	go func() {
		bb := suite.WaitBitbucket()

		re := regexp.MustCompile(
			`(at com.ngs.stash.externalhooks.|java.lang.\w+Exception)`,
		)

		found := false

		bb.WaitLogEntryContext(
			ctx,
			bb.Instance.StacktraceLogs(),
			func(line string) bool {
				if re.MatchString(line) {
					log.Errorf(nil, "got an exception: %s", line)
					found = true
					stop()
					return true
				}
				return false
			},
			bitbucket.DefaultWaitLogEntryDuration,
		)

		<-ctx.Done()
		result <- found
	}()

	return result, stop
}

func (suite *Suite) ConfigureHook(
	hook *external_hooks.Hook,
	settings external_hooks.Settings,
	script []byte,
) *external_hooks.Hook {
	path := filepath.Join("shared", "external-hooks", settings.Exe())

	log.Debugf(
		karma.Describe("script", "\n"+string(script)),
		"{hook} writing hook script %q",
		path,
	)

	suite.Bitbucket().FlushLogs(suite.Bitbucket().TestcaseLogs())

	err := suite.Bitbucket().WriteFile(path, append(script, '\n'), 0o777)
	suite.NoError(err, "should be able to write hook script to container")

	err = hook.Configure(settings)
	suite.NoError(err, "should be able to configure hook")

	suite.EnableHook(hook)

	return hook
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

func (suite *Suite) ConfigureSampleHook_Message(
	hook *external_hooks.Hook,
	message string,
) *external_hooks.Hook {
	return suite.ConfigureSampleHook(
		hook,
		string(text(
			fmt.Sprintf(`echo %s`, message),
			`exit 0`,
		)),
	)
}

func (suite *Suite) ConfigureSampleHook(
	hook *external_hooks.Hook,
	script string,
	args ...string,
) *external_hooks.Hook {
	settings := external_hooks.NewScopeSettings().
		UseSafePath(true).
		WithExe(`hook.` + lojban.GetRandomID(5)).
		WithParams(args...)

	return suite.ConfigureHook(
		hook,
		settings,
		text(
			`#!/bin/bash`,
			script,
		),
	)
}

func (suite *Suite) ConfigureSettingsHook(
	hook *external_hooks.Hook,
	settings external_hooks.Settings,
	script string,
	args ...string,
) *external_hooks.Hook {
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

	suite.Bitbucket().FlushLogs(suite.Bitbucket().TestcaseLogs())

	waiter := suite.Bitbucket().WaitLogEntryContext(
		context.Background(),
		suite.Bitbucket().TestcaseLogs(),
		func(line string) bool {
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
		},
		time.Second*60,
	)

	key := suite.Runner.InstallAddon(addon.Version, addon.Path)

	log.Debugf(nil, "{add-on} waiting for add-on startup process to finish")

	waiter.Wait(suite.FailNow, "hook scripts", "created")

	return key
}

func (suite *Suite) DisableHook(
	hook interface {
		Disable() error
		Wait() error
	},
) {
	err := hook.Disable()
	suite.NoError(err, "should be able to disable hook")

	err = hook.Wait()
	suite.NoError(err, "should be able to wait for disable hook")
}

func (suite *Suite) WaitExternalHookEnabled(hook interface {
	Global() bool
}) {
	if hook.Global() {
		return
	}

	re := regexp.MustCompile(`(?i)external hook enabled`)
	waiter := suite.Bitbucket().WaitLogEntry(
		func(line string) bool {
			return re.MatchString(line)
		},
	)

	waiter.Wait(suite.FailNow, "external hook", "enabled")
}

func (suite *Suite) WaitExternalHookDisabled(hook interface {
	Global() bool
}) {
	if hook.Global() {
		return
	}

	re := regexp.MustCompile(`(?i)external hook disabled`)
	waiter := suite.Bitbucket().WaitLogEntry(
		func(line string) bool {
			return re.MatchString(line)
		},
	)

	waiter.Wait(suite.FailNow, "external hook", "disabled")
}

func (suite *Suite) WaitExternalHookConfigured(hook interface {
	Global() bool
}) {
	if hook.Global() {
		return
	}

	re := regexp.MustCompile(`(?i)external hook configured`)
	waiter := suite.Bitbucket().WaitLogEntry(
		func(line string) bool {
			return re.MatchString(line)
		},
	)

	waiter.Wait(suite.FailNow, "external hook", "configured")
}

func (suite *Suite) WaitExternalHookUnconfigured() {
	re := regexp.MustCompile(`(?i)external hook unconfigured`)
	waiter := suite.Bitbucket().WaitLogEntry(
		func(line string) bool {
			return re.MatchString(line)
		},
	)

	waiter.Wait(suite.FailNow, "external hook", "unconfigured")
}

func (suite *Suite) WaitHookScriptsCreated() {
	re := regexp.MustCompile(
		`(?i)(ExternalHookScript|HooksFactory)\W+(applied|created).*hook\s*script`,
	)
	waiter := suite.Bitbucket().WaitLogEntry(
		func(line string) bool {
			return re.MatchString(line)
		},
	)

	waiter.Wait(suite.FailNow, "hook scripts", "created")
}

func (suite *Suite) WaitHookScriptsInherited() {
	re := regexp.MustCompile(
		`(?i)ExternalHookScript`,
	)
	waiter := suite.Bitbucket().WaitLogEntry(
		func(line string) bool {
			return re.MatchString(line)
		},
	)

	waiter.Wait(suite.FailNow, "hook scripts", "inherited")
}

func (suite *Suite) EnableHook(
	hook interface {
		Enable() error
		Wait() error
	},
) {
	err := hook.Enable()
	suite.NoError(err, "should be able to enable hook")

	err = hook.Wait()
	suite.NoError(err, "should be able to wait for enable hook")
}

type InheritHookExpectedState string

const (
	InheritHookExpectedStateEnabledProject InheritHookExpectedState = "created project/repository hook script"
)

func (suite *Suite) InheritHook(
	hook interface{ Inherit() error },
	expectedState InheritHookExpectedState,
) {
	err := hook.Inherit()
	suite.NoError(err, "should be able to disable hook")

	log.Debugf(
		nil,
		"{add-on} waiting for hook script to be inherited by bitbucket: %s",
		expectedState,
	)
}

func (suite *Suite) getHookScripts() []HookScript {
	const tagPrefix = "# com.ngs.stash.externalhooks tag: "

	files, err := suite.Bitbucket().ReadFiles("shared/config/hook-scripts/")
	suite.NoError(err, "should be able to list existing hook scripts")

	scripts := []HookScript{}
	for _, file := range files {
		suite.NoError(err, "should be able to read hook script contents")

		lines := strings.Split(file.Contents, "\n")

		tag := ""
		for _, line := range lines {
			if line == "" {
				continue
			}

			if strings.HasPrefix(line, tagPrefix) {
				tag = strings.TrimPrefix(line, tagPrefix)
				break
			}
		}

		// this can happen if we are reproducing a bug
		if tag == "" {
			tag = "legacy-" + filepath.Base(file.Name)
		}

		scripts = append(scripts, HookScript{
			ID:  file.Name,
			Tag: tag,
		})
	}

	return scripts
}

func (suite *Suite) RecordHookScripts() {
	suite.hookScripts = suite.getHookScripts()

	log.Debugf(
		karma.Describe(
			"scripts (plugin paths)",
			joinHookScripts(suite.hookScripts),
		),
		"{leak detector} found %d currently registered hook scripts",
		len(suite.hookScripts),
	)
}

func (suite *Suite) DetectHookScriptsLeak() {
	current := suite.getHookScripts()

	index := map[string]struct{}{}

	for _, script := range suite.hookScripts {
		index[script.Tag] = struct{}{}
	}

	leak := []HookScript{}

	for _, script := range current {
		if _, ok := index[script.Tag]; !ok {
			leak = append(leak, script)
		}
	}

	if len(leak) > 0 {
		suite.Empty(joinHookScripts(leak), "found leaking hook scripts")
	} else {
		log.Debugf(
			nil,
			"{leak detector} no hook scripts leak detected",
		)
	}
}

func (suite *Suite) CreateUserAlice() *stash.User {
	return suite.CreateUser("alice")
}

func (suite *Suite) CreateUser(name string) *stash.User {
	password := "p" + name
	email := name + "@bitbucket.test"

	user, err := suite.Bitbucket().Admin().CreateUser(name, password, email)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return &stash.User{Name: name, Password: password}
		}

		suite.NoError(err, "create user")
	}

	return user
}

func (suite *Suite) CleanupHooks() {
	context := suite.ExternalHooks().OnGlobal()

	if err := context.PreReceive().Disable(); err != nil {
		log.Errorf(err, "{suite:cleanup} disable pre-receive")
	}

	if err := context.PostReceive().Disable(); err != nil {
		log.Errorf(err, "{suite:cleanup} disable post-receive")
	}

	if err := context.MergeCheck().Disable(); err != nil {
		log.Errorf(err, "{suite:cleanup} disable merge-check")
	}

	err := context.Addon.Wait(context)
	if err != nil {
		log.Errorf(err, "{suite:cleanup} apply hooks factory")
	}
}

func joinHookScripts(scripts []HookScript) string {
	list := []string{}
	for _, script := range scripts {
		list = append(list, script.ID+" ("+script.Tag+")")
	}
	sort.Strings(list)
	return strings.Join(list, ", ")
}

func (suite *Suite) ExternalHooks(opts ...interface{}) *external_hooks.Addon {
	var user *stash.User

	for _, opt := range opts {
		switch opt := opt.(type) {
		case *stash.User:
			user = opt
		}
	}

	return &external_hooks.Addon{
		BitbucketURI: suite.Bitbucket().ConnectorURI(user),
	}
}

func (suite *Suite) CreateRandomProject() *stash.Project {
	project, err := suite.Bitbucket().Projects().
		Create(lojban.GetRandomID(6))
	suite.NoError(err, "create project")

	return project
}

func (suite *Suite) CreateRandomRepository(
	project *stash.Project,
) *stash.Repository {
	repository, err := suite.Bitbucket().Repositories(project.Key).
		Create(lojban.GetRandomID(6))
	suite.NoError(err, "create repository")

	return repository
}

func (suite *Suite) CreateRandomPullRequest(
	project *stash.Project,
	repository *stash.Repository,
) *stash.PullRequest {
	git := suite.GitClone(repository)

	suite.GitCommitRandomFile(git)

	_, err := git.Push()
	suite.NoError(err, "git push into master")

	branch := suite.GitCreateRandomBranch(git)

	suite.GitCommitRandomFile(git)

	_, err = git.Push("origin", branch)
	suite.NoErrorf(err, "git push into branch %s", branch)

	pullRequest, err := suite.Bitbucket().Repositories(project.Key).
		PullRequests(repository.Slug).
		Create(
			"pr."+lojban.GetRandomID(8),
			lojban.GetRandomID(20),
			branch,
			"master",
		)
	suite.NoError(err, "create pull request")

	return pullRequest
}
