package main

import (
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/pkg/log"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/runner"
)

var version = "[manual build]"

var usage = `external-hooks-tests - run external hooks test suites.

Usage:
  external-hooks-test [options] run --container=<container>
  external-hooks-test [options] run [--keep]
  external-hooks-test -h | --help

Options:
  -h --help  Show this help.
  --trace    Set trace log level.
  --keep     Keep work dir & bitbucket instance.
`

type Opts struct {
	ModeRun bool `docopt:"run"`

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

	run := runner.New()
	run.Suite(SuiteBasic)
	run.Run(dir, runner.RunOpts{
		opts.ValueContainer,
	})

	if !opts.FlagKeep && opts.ValueContainer == "" {
		err := run.Cleanup()
		if err != nil {
			log.Fatalf(err, "unable to cleanup runner")
		}
	}
}
