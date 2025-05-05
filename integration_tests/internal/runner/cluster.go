package runner

import (
	"fmt"
	"strings"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/cluster"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
)

func (runner *Runner) UseCluster(version bitbucket.Version, replicas int) {
	var err error

	// the id is used as network domain, containers' prefix and volumes' prefix
	id := fmt.Sprintf("aeh-%s", lojban.GetRandomID(5))
	if runner.run.identifier != "" {
		id = strings.TrimSuffix(runner.run.identifier, "-bitbucket")
	}

	if runner.run.bitbucket != nil {
		err := runner.run.bitbucket.Stop()
		runner.assert.NoError(err, "stopping bitbucket server instance")
		err = runner.run.bitbucket.RemoveContainer()
		runner.assert.NoError(err, "removing bitbucket server instance")
		runner.run.bitbucket = nil
	}

	runner.useDatabase(id)

	switch {
	case runner.run.cluster != nil:
		err = runner.run.cluster.Upgrade(version)
		runner.assert.NoError(err, "upgrading bitbucket cluster")
		id = runner.run.cluster.ID()

	case runner.run.identifier != "":
		runner.run.cluster, err = cluster.StartExisting(cluster.StartOpts{
			ID:      id,
			Volumes: runner.run.volumes,
			RunOpts: bitbucket.RunOpts{
				Version:  version,
				Database: runner.run.database,
				Network:  id,
			},
		})
		runner.assert.NoError(err, "start existing bitbucket cluster")
	default:
		runner.run.cluster, err = cluster.StartNew(cluster.StartOpts{
			ID:      id,
			Volumes: runner.run.volumes,
			RunOpts: bitbucket.RunOpts{
				Version:  version,
				Database: runner.run.database,
				Network:  id,
			},
		})
		runner.assert.NoError(err, "start new bitbucket cluster")
	}

	runner.run.identifier = id

	err = runner.run.cluster.Configure()
	runner.assert.NoError(err, "unable to configure bitbucket")

	err = runner.run.cluster.Verify()
	runner.assert.NoError(err, "unable to verify bitbucket cluster state")

	// err = runner.run.cluster.Configure()
	// runner.assert.NoError(err, "unable to configure bitbucket cluster")

	runner.ready()

	runner.run.cleanup()
}
