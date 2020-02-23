package main

import (
	"github.com/reconquest/pkg/log"
)

type RunOpts struct {
	Container string
}

func (suite *Suite) Run(dir string, opts RunOpts) {
	suite.run.dir = dir
	suite.run.container = opts.Container

	log.Infof(nil, "work dir: %s", suite.run.dir)

	for _, testcase := range suite.testcases {
		testcase(suite)
	}
}
