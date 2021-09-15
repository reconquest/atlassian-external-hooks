package runner

import (
	"math/rand"

	"github.com/reconquest/pkg/log"
)

type RunOpts struct {
	Container string
	Randomize bool
}

func (runner *Runner) Run(dir string, opts RunOpts) {
	runner.run.dir = dir
	runner.run.container = opts.Container

	log.Debugf(nil, "{run} work dir: %s", runner.run.dir)

	if opts.Randomize {
		rand.Shuffle(
			len(runner.suites),
			func(i, j int) {
				runner.suites[i], runner.suites[j] = runner.suites[j], runner.suites[i]
			},
		)
	}

	for _, suite := range runner.suites {
		suite(runner, runner.assert)
	}
}
