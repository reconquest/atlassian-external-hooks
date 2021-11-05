package runner

import (
	"math/rand"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/status"
	"github.com/reconquest/pkg/log"
)

type RunOpts struct {
	Workdir   string
	Database  string
	Container string
	Randomize bool
}

func (runner *Runner) Run(opts RunOpts) {
	runner.run.workdir = opts.Workdir
	runner.run.container = opts.Container
	runner.run.database = opts.Database

	log.Debugf(nil, "{run} work dir: %s", runner.run.workdir)

	if opts.Randomize {
		rand.Shuffle(
			len(runner.suites),
			func(i, j int) {
				runner.suites[i], runner.suites[j] = runner.suites[j], runner.suites[i]
			},
		)
	}

	total := 0
	for _, suite := range runner.suites {
		total += suite.Size
	}

	status.SetTotal(total)

	for _, suite := range runner.suites {
		suite.Run(runner, runner.assert)
	}
}
