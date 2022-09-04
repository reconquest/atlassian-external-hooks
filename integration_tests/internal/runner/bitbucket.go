package runner

import (
	"fmt"
	"strings"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/lojban"
)

const (
	ClusterSize = 3
)

func (runner *Runner) UseBitbucket(version string, cluster bool) {
	if cluster {
		runner.UseCluster(version, ClusterSize)
		return
	}

	var err error

	// the id is used as network domain, containers' prefix and volumes' prefix
	id := fmt.Sprintf("aeh-%s", lojban.GetRandomID(5))
	if runner.run.identifier != "" {
		id = strings.TrimSuffix(runner.run.identifier, "-bitbucket")
	}

	properties := bitbucket.NewProperties().
		WithLicense(bitbucket.LICENSE_DATACENTER_3H).
		WithSidecarMeshEnabled(false)

	switch {
	case runner.run.bitbucket != nil:
		runner.upgrade(runner.run.bitbucket.ID(), version)

	case runner.run.identifier != "":
		runner.useDatabase(id)

		runner.run.bitbucket, err = bitbucket.StartExisting(
			bitbucket.StartExistingOpts{
				Container: runner.run.identifier,
				Volumes:   runner.run.volumes,
				RunOpts: bitbucket.RunOpts{
					Version:    version,
					Database:   runner.run.database,
					Network:    id,
					Properties: properties,
				},
			},
		)
		runner.assert.NoError(err, "start existing container")

	default:
		runner.useDatabase(id)

		runner.run.bitbucket, err = bitbucket.StartNew(
			bitbucket.StartNewOpts{
				ID:      string(id),
				Volumes: runner.run.volumes,
				RunOpts: bitbucket.RunOpts{
					Version:    version,
					Database:   runner.run.database,
					Network:    id,
					Properties: properties,
				},
			},
		)
		runner.assert.NoError(err, "start new bitbucket container")
	}

	runner.run.identifier = runner.run.bitbucket.Container()

	err = runner.run.bitbucket.Configure()
	if err == nil {
		runner.ready()
	}

	runner.run.cleanup()

	runner.assert.NoError(err, "unable configure bitbucket")
}

func (runner *Runner) ready() {
	select {
	case runner.run.ready <- struct{}{}:
	default:
	}
}
