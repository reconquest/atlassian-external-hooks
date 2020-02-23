package main

import "github.com/reconquest/atlassian-external-hooks/tests/internal/external_hooks"

func (suite *Suite) ExternalHooks() *external_hooks.Addon {
	return &external_hooks.Addon{
		BitbucketURI: suite.Bitbucket().GetConnectorURI(),
	}
}
