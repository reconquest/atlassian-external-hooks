package main

import (
	"io"
	"net/http"
	"time"

	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/bitbucket"
)

func (suite *Suite) TestAdministration(params TestParams) {
	suite.UseBitbucket(params.Bitbucket, params.Cluster)
	suite.InstallAddon(params.Addon)

	client := &http.Client{
		Timeout: 30 * time.Second,
		CheckRedirect: func(request *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	for _, endpoint := range []struct {
		path string
		body string
	}{
		{"/plugins/servlet/external-hooks/global", "External Hooks"},
		{"/plugins/servlet/external-hooks/settings", "External Hooks"},
		{"/plugins/servlet/external-hooks/diagnostics", "Current number of HookScripts installed:"},
		{"/rest/external-hooks/1.0/settings", "\"pre_receive\""},
		{"/rest/external-hooks/1.0/settings/default", "\"merge_check\""},
	} {
		request, err := http.NewRequest(http.MethodGet, suite.Bitbucket().URI(endpoint.path), nil)
		suite.NoError(err, "create admin request")
		request.SetBasicAuth(bitbucket.ADMIN_USERNAME, bitbucket.ADMIN_PASSWORD)
		response, err := client.Do(request)
		suite.NoError(err, "request admin endpoint")
		body, err := io.ReadAll(response.Body)
		response.Body.Close()
		suite.NoError(err, "read admin endpoint")
		suite.Equal(http.StatusOK, response.StatusCode, endpoint.path)
		suite.Contains(string(body), endpoint.body)
		suite.NotContains(string(body), "$hook_scripts_total")

		request.Header.Del("Authorization")
		response, err = client.Do(request)
		suite.NoError(err, "request anonymous endpoint")
		response.Body.Close()
		suite.True(
			response.StatusCode == http.StatusUnauthorized ||
				response.StatusCode == http.StatusForbidden ||
				response.StatusCode == http.StatusFound ||
				response.StatusCode == http.StatusSeeOther,
			"anonymous request must not access %s: %d", endpoint.path, response.StatusCode,
		)
	}
}
