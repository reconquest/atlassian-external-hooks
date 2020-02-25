package bitbucket

import (
	"net/url"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

type Bitbucket struct {
	*Instance

	client stash.Stash
}

func New(instance *Instance) (*Bitbucket, error) {
	bitbucket := &Bitbucket{
		Instance: instance,
	}

	url, err := url.Parse(instance.GetConnectorURI())
	if err != nil {
		return nil, karma.
			Describe("uri", instance.GetConnectorURI()).
			Format(
				err,
				"unable to parse bitbucket connector uri",
			)
	}

	var (
		user    = url.User.Username()
		pass, _ = url.User.Password()
	)

	bitbucket.client = stash.NewClient(user, pass, url)

	return bitbucket, nil
}

func (bitbucket *Bitbucket) Projects() *BitbucketProjectsAPI {
	return &BitbucketProjectsAPI{
		client: bitbucket.client,
	}
}

func (bitbucket *Bitbucket) Repositories(
	project string,
) *BitbucketRepositoriesAPI {
	return &BitbucketRepositoriesAPI{
		client:  bitbucket.client,
		project: project,
	}
}

func (bitbucket *Bitbucket) Addons() *BitbucketAddonsAPI {
	return &BitbucketAddonsAPI{
		client: bitbucket.client,
	}
}

type BitbucketProjectsAPI struct {
	client stash.Stash
}

func (api *BitbucketProjectsAPI) Create(key string) (*stash.Project, error) {
	project, err := api.client.CreateProject(key)
	if err != nil {
		return nil, err
	}

	return &project, nil
}

type BitbucketRepositoriesAPI struct {
	client  stash.Stash
	project string
}

func (api *BitbucketRepositoriesAPI) Create(slug string) (*stash.Repository, error) {
	repository, err := api.client.CreateRepository(api.project, slug)
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

type BitbucketAddonsAPI struct {
	client stash.Stash
}

func (api *BitbucketAddonsAPI) Install(path string) (string, error) {
	token, err := api.client.GetUPMToken()
	if err != nil {
		return "", karma.Format(
			err,
			"unable to get upm token",
		)
	}

	log.Debugf(
		karma.Describe("upm_token", token),
		"installing add-on: %s",
		path,
	)

	key, err := api.client.InstallAddon(token, path)
	if err != nil {
		return "", karma.Format(
			err,
			"unable to install add-on",
		)
	}

	return key, nil
}

func (api *BitbucketAddonsAPI) Uninstall(key string) error {
	token, err := api.client.GetUPMToken()
	if err != nil {
		return karma.Format(
			err,
			"unable to get upm token",
		)
	}

	log.Debugf(
		karma.Describe("upm_token", token),
		"uninstalling add-on: %s",
		key,
	)

	err = api.client.UninstallAddon(token, key)
	if err != nil {
		return karma.Format(
			err,
			"unable to uninstall add-on",
		)
	}

	return nil
}

func (api *BitbucketAddonsAPI) SetLicense(addon string, license string) error {
	log.Debugf(
		karma.Describe("license", license),
		"setting add-on license: %s",
		addon,
	)

	err := api.client.SetAddonLicense(addon, license)
	if err != nil {
		return karma.
			Describe("license", license).
			Describe("addon", addon).
			Format(
				err,
				"unable to set addon license",
			)
	}

	return nil
}
