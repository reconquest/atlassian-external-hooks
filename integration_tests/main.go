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
  --trace    Set trace log level.
  --keep     Keep work dir & bitbucket instance.
`

type Opts struct {
	FlagKeep  bool `docopt:"--keep"`
	FlagTrace bool `docopt:"--trace"`

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

	if opts.FlagTrace {
		log.SetLevel(log.LevelTrace)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	dir, err := ioutil.TempDir("", "external-hooks.test.")
	if err != nil {
		log.Fatalf(err, "unable to create work dir")
	}

	var (
		baseBitbucket = "6.2.0"
		latestAddon   = getAddon("10.0.0")
	)

	suite := NewSuite()

	run := runner.New()

	// TODO: add tests for env vars in all type of hooks
	// TODO: add tests for stdin for all type of hooks
	// TODO: add tests for different trigger configurations
	// TODO: add tests for BB 5.x.x

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
	path := fmt.Sprintf("target/external-hooks-%s.jar", version)

	if _, err := os.Stat(path); err != nil {
		log.Fatalf(
			err,
			"can not find addon version %s at path %q",
			version, path,
		)
	}

	return Addon{
		Version: version,
		Path:    path,
	}
}
