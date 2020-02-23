package main

import (
	"io/ioutil"
	"math/rand"
	"time"

	"github.com/docopt/docopt-go"
	"github.com/reconquest/pkg/log"
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
	Keep      bool   `docopt:"--keep"`
	Container string `docopt:"--container"`
	Trace     bool   `docopt:"--trace"`
	WorkDir   string `docopt:"<work-dir>"`
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

	if opts.Trace {
		log.SetLevel(log.LevelTrace)
	}

	rand.Seed(time.Now().UTC().UnixNano())

	dir, err := ioutil.TempDir("", "external-hooks.test.")
	if err != nil {
		log.Fatalf(err, "unable to create work dir")
	}

	suite := NewSuite()
	suite.Testcase(TestBasic)
	suite.Run(dir, RunOpts{
		opts.Container,
	})

	if !opts.Keep && opts.Container == "" {
		err := suite.Cleanup()
		if err != nil {
			log.Fatalf(err, "unable to cleanup suite")
		}
	}
}
