package runner

import (
	"github.com/reconquest/pkg/log"
)

type RunOpts struct {
	Container string
}

func (runner *Runner) Run(dir string, opts RunOpts) {
	runner.run.dir = dir
	runner.run.container = opts.Container

	log.Infof(nil, "{run} work dir: %s", runner.run.dir)

	for _, suite := range runner.suites {
		suite(runner, runner.assert)
	}
}
