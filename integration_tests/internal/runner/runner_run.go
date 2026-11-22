package runner

import (
	"math/rand"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/status"
	"github.com/reconquest/pkg/log"
)

type RunOpts struct {
	Keep      bool
	Workdir   string
	Database  string
	Container string
	Randomize bool
}

func (runner *Runner) Run(opts RunOpts) {
	runner.run.workdir = opts.Workdir
	runner.run.identifier = opts.Container
	runner.run.databaseKind = opts.Database

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
		if suite.Size == 0 {
			continue
		}

		if instance := runner.Bitbucket(); instance != nil &&
			(instance.Version() != suite.Bitbucket || runner.ClusteringEnabled() != suite.Cluster) {
			if opts.Container != "" {
				log.Fatalf(nil, "cannot switch Bitbucket environments with --container; select one with --run")
			}
			runner.release(opts.Keep)
		}

		suite.Run(runner, runner.assert)
	}

	runner.release(opts.Keep || opts.Container != "")
}

func (runner *Runner) release(keep bool) {
	if runner.Bitbucket() == nil {
		return
	}
	if keep {
		runner.logKeptResources()
	} else if err := runner.Cleanup(); err != nil {
		log.Fatalf(err, "cleanup runner")
	}

	runner.run.identifier = ""
	runner.run.bitbucket = nil
	runner.run.cluster = nil
	runner.run.database = nil
	runner.run.ready = make(chan struct{}, 1)
}
