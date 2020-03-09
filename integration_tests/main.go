package main

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
)

var version = "[manual build]"

var usage = `external-hooks-tests - run external hooks test suites.

Usage:
  external-hooks-test [options] --container=<container>
  external-hooks-test [options] [--keep]
  external-hooks-test -h | --help

Options:
  -h --help  Show this help.
  --debug    Set debug log level.
  --trace    Set trace log level.
  --keep     Keep work dir & bitbucket instance.
`

type Opts struct {
	FlagKeep  bool `docopt:"--keep"`
	FlagTrace bool `docopt:"--trace"`
	FlagDebug bool `docopt:"--debug"`

	ValueContainer string `docopt:"--container"`
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

	var (
		baseBitbucket = "6.2.0"
		latestAddon   = getAddon("10.1.0")
	)

	run := runner.New()

	suite := NewSuite()

	// TODO: add tests for different trigger configurations
	// TODO: add tests for BB 5.x.x

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("10.0.0"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_ProjectHookCreatedBeforeRepository,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket":        baseBitbucket,
				"addon_reproduced": getAddon("9.1.0"),
				"addon_fixed":      latestAddon,
			},

			suite.TestBug_ProjectEnabledRepositoryDisabledHooks,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket": baseBitbucket,
				"addon":     latestAddon,
			},
			suite.TestProjectHooks_DoNotCreateDisabledHooks,

			// XXX: BB doesn't clean up hook scripts if repository was deleted.
			// suite.TestHookScriptsLeak_NoLeakAfterRepositoryDelete,
		),
	)

	run.Suite(
		suite.WithParams(
			TestParams{
				"bitbucket": baseBitbucket,
				"addon":     latestAddon,
			},
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
		Container: opts.ValueContainer,
	})

	log.Infof(nil, "{run} all tests passed")

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
		log.Infof(
			karma.
				Describe("container", run.Bitbucket().GetContainerID()).
				Describe("volume", run.Bitbucket().GetVolume()),
			"{run} following resources can be reused",
		)
	}
}

func getAddon(version string) Addon {
	builds := map[string]string{
		"10.1.0": "6532",
		"10.0.0": "6512",
		"9.1.0":  "6492",
	}

	path := fmt.Sprintf("target/external-hooks-%s.jar", version)

	_, err := os.Stat(path)
	if err != nil {
		if build, ok := builds[version]; ok {
			log.Infof(
				karma.Describe("build", build).Describe("version", version),
				"downloading add-on from Marketplace",
			)

			cmd := exec.New(
				"wget", "-O", path,
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
					version, path,
				)
			}
		} else {
			log.Fatalf(
				err,
				"unable to find add-on version %s at path %q",
				version, path,
			)
		}
	}

	return Addon{
		Version: version,
		Path:    path,
	}
}
