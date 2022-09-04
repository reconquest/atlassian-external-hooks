package runner

import (
	"fmt"
	"strings"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/cluster"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
)

func (runner *Runner) UseCluster(version string, replicas int) {
	var err error

	// the id is used as network domain, containers' prefix and volumes' prefix
	id := fmt.Sprintf("aeh-%s", lojban.GetRandomID(5))
	if runner.run.identifier != "" {
		id = strings.TrimSuffix(runner.run.identifier, "-bitbucket")
	}

	if runner.run.bitbucket != nil {
		runner.assert.Fail("cluster doesn't support upgrade operation yet")
	}

	if runner.run.cluster != nil {
		runner.ready()
		// we do not support upgrade yet
		return
	}

	runner.useDatabase(id)

	var started *cluster.Cluster
	if runner.run.identifier != "" {
		started, err = cluster.StartExisting(cluster.StartOpts{
			ID:      id,
			Volumes: runner.run.volumes,
			RunOpts: bitbucket.RunOpts{
				Version:  version,
				Database: runner.run.database,
				Network:  id,
			},
		})
		runner.assert.NoError(err, "start existing bitbucket cluster")
	} else {
		started, err = cluster.StartNew(cluster.StartOpts{
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
	runner.run.cluster = started

	err = runner.run.cluster.Configure()
	runner.assert.NoError(err, "unable to configure bitbucket")

	err = runner.run.cluster.Verify()
	runner.assert.NoError(err, "unable to verify bitbucket cluster state")

	err = runner.run.cluster.Configure()
	runner.assert.NoError(err, "unable to configure bitbucket cluster")

	runner.ready()

	runner.run.cleanup()
}
