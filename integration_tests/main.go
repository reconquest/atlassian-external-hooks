package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
	"github.com/reconquest/pom"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/status"
)

var version = "[manual build]"

var usage = `external-hooks-tests - run external hooks test suites.

Usage:
  external-hooks-test [options] --container=<container>
  external-hooks-test [options] [--keep]
  external-hooks-test -h | --help

Options:
  -l --list                   List testcases.
  -C --container <container>  Use specified container.  
  -K --keep                   Keep work dir & bitbucket instance.
  --no-upgrade                Do not run suites with upgrades.
  --no-reproduce              Do not run suites with bug reproduces.
  -r --run <name>             Run only specified testcases.
  --no-randomize              Do not randomize tests order.
  --debug                     Set debug log level.
  --trace                     Set trace log level.
  -h --help                   Show this help.
`

type Opts struct {
	FlagKeep        bool `docopt:"--keep"`
	FlagTrace       bool `docopt:"--trace"`
	FlagDebug       bool `docopt:"--debug"`
	FlagNoUpgrade   bool `docopt:"--no-upgrade"`
	FlagNoReproduce bool `docopt:"--no-reproduce"`
	FlagList        bool `docopt:"--list"`
	FlagNoRandomize bool `docopt:"--no-randomize"`

	ValueContainer string `docopt:"--container"`
	ValueRun       string `docopt:"--run"`
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	args, err := docopt.ParseArgs(usage, nil, "external-hooks-tests "+version)
	if err != nil {
		log.Fatal(err)
	}

	var opts Opts

	err = args.Bind(&opts)
	if err != nil {
		log.Fatal(err)
	}

	defer status.Destroy()

	switch {
	case opts.FlagDebug:
		log.SetLevel(log.LevelDebug)
	case opts.FlagTrace:
		log.SetLevel(log.LevelTrace)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	dir, err := ioutil.TempDir("", "external-hooks.test.")
	if err != nil {
		log.Fatalf(err, "unable to create work dir")
	}

	ensureAddons()

	var (
		baseBitbucket = "6.2.0"
		latestAddon   = getAddon(getLatestVersionXML())
	)

	mode := ModeRun
	if opts.FlagList {
		mode = ModeList
	}

	suite := NewSuite(
		baseBitbucket,
		mode == ModeRun && !opts.FlagNoRandomize,
		mode,
		Filter{
			upgrade:   !opts.FlagNoUpgrade,
			reproduce: !opts.FlagNoReproduce,
			glob:      opts.ValueRun,
		},
	)

	// TODO: add tests for different trigger configurations
	// TODO: add tests for BB 5.x.x

	run := runner.New(suite.CleanupHooks)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("10.1.0"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_ProjectEnabledRepositoryOverriddenHooks_Reproduced,
			suite.TestBug_ProjectEnabledRepositoryOverriddenHooks_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("10.0.0"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_ProjectHookCreatedBeforeRepository_Reproduced,
			suite.TestBug_ProjectHookCreatedBeforeRepository_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("9.1.0"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_ProjectEnabledRepositoryDisabledHooks_Reproduced,
			suite.TestBug_ProjectEnabledRepositoryDisabledHooks_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket": baseBitbucket,
				"addon":     latestAddon,
			},
			suite.TestProjectHooks_DoNotCreateDisabledHooks,

			suite.TestHookScriptsLeak_NoLeakAfterRepositoryDelete,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("10.2.1"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_UserWithoutProjectAccessModifiesInheritedHook_Reproduced,
			suite.TestBug_UserWithoutProjectAccessModifiesInheritedHook_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("11.1.0"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_RepositoryHookCreatedBeforeProject_Reproduced,
			suite.TestBug_RepositoryHookCreatedBeforeProject_Fixed,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket": baseBitbucket,
				"addon":     latestAddon,
			},
			suite.TestGlobalHooks,
			suite.TestGlobalHooks_PersonalRepositoriesFilter,
			suite.TestProjectHooks,
			suite.TestRepositoryHooks,
			suite.TestPersonalRepositoriesHooks,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket_from": baseBitbucket,
				"bitbucket_to":   "6.9.0",
				"addon":          latestAddon,
			},
			suite.TestBitbucketUpgrade,
		),
	)

	_ = baseBitbucket

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket": "7.0.0",
				"addon":     latestAddon,
			},
			suite.TestProjectHooks,
			suite.TestRepositoryHooks,
			suite.TestPersonalRepositoriesHooks,
		),
	)

	run.Run(dir, runner.RunOpts{
		Randomize: mode == ModeRun && !opts.FlagNoRandomize,
		Container: opts.ValueContainer,
	})

	if !opts.FlagList {
		log.Infof(nil, "{run} all tests passed")
	}

	log.Debugf(nil, "{run} removing work dir: %s", dir)
	err = os.RemoveAll(dir)
	if err != nil {
		log.Errorf(err, "unable to remove work dir")
	}

	if !opts.FlagKeep && opts.ValueContainer == "" {
		err := run.Cleanup()
		if err != nil {
			log.Fatalf(err, "unable to cleanup runner")
		}
	} else {
		if run.Bitbucket() != nil {
			log.Infof(
				karma.
					Describe("container", run.Bitbucket().GetContainerID()).
					Describe("volume", run.Bitbucket().GetVolume()),
				"{run} following resources can be reused",
			)
		}
	}
}

var builds = map[string]string{
	"11.1.0": "6642",
	"10.2.2": "6592",
	"10.2.1": "6572",
	"10.1.0": "6532",
	"10.0.0": "6512",
	"9.1.0":  "6492",
}

func ensureAddons() {
	err := os.MkdirAll("builds", 0o755)
	if err != nil {
		log.Fatalf(err, "mkdir builds")
	}

	getters := &sync.WaitGroup{}
	for version := range builds {
		getters.Add(1)
		go func(version string) {
			defer getters.Done()
			getAddon(version)
		}(version)
	}

	getters.Wait()
}

func getAddon(version string) Addon {
	buildsPath := fmt.Sprintf("builds/external-hooks-%s.jar", version)

	_, err := os.Stat(buildsPath)
	if err == nil {
		return Addon{
			Version: version,
			Path:    buildsPath,
		}
	}

	if build, ok := builds[version]; ok {
		log.Infof(
			karma.Describe("build", build).Describe("version", version),
			"downloading add-on from Marketplace",
		)

		cmd := exec.New(
			"wget", "-O", buildsPath,
			fmt.Sprintf(
				"https://marketplace.atlassian.com/download/apps/1211631/version/%v",
				build,
			),
		)

		err := cmd.Run()
		if err != nil {
			log.Fatalf(
				karma.Describe("build", build).Reason(err),
				"unable to download add-on %s from Marketplace to %q",
				version, buildsPath,
			)
		}

		return Addon{
			Version: version,
			Path:    buildsPath,
		}
	}

	targetPath := fmt.Sprintf("target/external-hooks-%s.jar", version)
	_, err = os.Stat(targetPath)
	if err != nil {
		log.Fatalf(
			err,
			"unable to find add-on version %s at path %q and %q",
			version, buildsPath, targetPath,
		)
	}

	return Addon{
		Version: version,
		Path:    targetPath,
	}
}

func getLatestVersionXML() string {
	contents, err := ioutil.ReadFile("pom.xml")
	if err != nil {
		log.Fatalf(err, "unable to read pom.xml")
	}

	model, err := pom.Unmarshal(contents)
	if err != nil {
		log.Fatalf(err, "unable to unmarshal pom.xml")
	}

	version, err := model.Get("version")
	if err != nil {
		log.Fatalf(err, "unable to read pom.xml version")
	}

	return version
}

func text(lines ...string) []byte {
	return []byte(strings.Join(lines, "\n"))
}
