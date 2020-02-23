package runner

import "github.com/reconquest/atlassian-external-hooks/integration_tests/internal/external_hooks"

func (runner *Runner) ExternalHooks() *external_hooks.Addon {
	return &external_hooks.Addon{
		BitbucketURI: runner.Bitbucket().GetConnectorURI(),
	}
}
