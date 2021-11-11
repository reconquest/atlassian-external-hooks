package bitbucket

import (
	"net/url"
	"strconv"

	"github.com/kovetskiy/stash"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/users"
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

	url, err := url.Parse(instance.ConnectorURI(users.USER_ADMIN))
	if err != nil {
		return nil, karma.
			Describe("uri", instance.ConnectorURI(users.USER_ADMIN)).
			Format(
				err,
				"parse bitbucket connector uri",
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

func (bitbucket *Bitbucket) Admin() *BitbucketAdminAPI {
	return &BitbucketAdminAPI{
		client: bitbucket.client,
	}
}

type BitbucketProjectsAPI struct {
	client stash.Stash
}

func (api *BitbucketProjectsAPI) Create(key string) (*stash.Project, error) {
	log.Debugf(nil, "{bitbucket} creating project: %s", key)

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

func (api *BitbucketRepositoriesAPI) Create(
	slug string,
) (*stash.Repository, error) {
	log.Debugf(
		nil,
		"{bitbucket} creating repository: %s / %s",
		api.project,
		slug,
	)

	repository, err := api.client.CreateRepository(api.project, slug)
	if err != nil {
		return nil, err
	}

	return &repository, nil
}

func (api *BitbucketRepositoriesAPI) Remove(slug string) error {
	log.Debugf(
		nil,
		"{bitbucket} removing repository: %s / %s",
		api.project,
		slug,
	)

	err := api.client.RemoveRepository(api.project, slug)
	if err != nil {
		return err
	}

	return nil
}

func (api *BitbucketRepositoriesAPI) Permissions(
	repository string,
) *BitbucketRepositoryPermissionsAPI {
	return &BitbucketRepositoryPermissionsAPI{
		client:     api.client,
		project:    api.project,
		repository: repository,
	}
}

func (api *BitbucketRepositoriesAPI) PullRequests(
	repository string,
) *BitbucketPullRequestsAPI {
	return &BitbucketPullRequestsAPI{
		client:     api.client,
		project:    api.project,
		repository: repository,
	}
}

type BitbucketPullRequestsAPI struct {
	client     stash.Stash
	project    string
	repository string
}

func (api *BitbucketPullRequestsAPI) Get(
	id int,
) (*stash.PullRequest, error) {
	pullRequest, err := api.client.GetPullRequest(
		api.project,
		api.repository,
		strconv.Itoa(id),
	)
	if err != nil {
		return nil, err
	}

	return &pullRequest, nil
}

func (api *BitbucketPullRequestsAPI) Create(
	title string,
	description string,
	fromRef string,
	toRef string,
) (*stash.PullRequest, error) {
	log.Debugf(
		nil,
		"{bitbucket} creating pull request: %s / %s / %q (%s -> %s)",
		api.project,
		api.repository,
		title,
		fromRef, toRef,
	)

	prRepo := stash.PullRequestRepository{
		Slug: api.repository,
		Project: stash.PullRequestProject{
			Key: api.project,
		},
	}

	pullRequest, err := api.client.CreatePullRequest(
		title,
		description,
		stash.PullRequestRef{
			Id:         fromRef,
			Repository: prRepo,
		},
		stash.PullRequestRef{
			Id:         toRef,
			Repository: prRepo,
		},
		nil,
	)
	if err != nil {
		return nil, err
	}

	return &pullRequest, nil
}

func (api *BitbucketPullRequestsAPI) Merge(
	id int,
	version int,
) (*stash.MergeResult, error) {
	log.Debugf(
		nil,
		"{bitbucket} merging pull request: %s / %s / %d (version %d)",
		api.project,
		api.repository,
		id,
		version,
	)

	result, err := api.client.MergePullRequest(
		api.project,
		api.repository,
		strconv.Itoa(id),
		version,
	)
	if err != nil {
		return nil, err
	}

	if result.State == "" {
		log.Tracef(
			karma.Describe("errors", result.Errors),
			"{bitbucket} merging pull request: %s / %s / %d -> %s",
			api.project,
			api.repository,
			id,
			"VETOED",
		)
	} else {
		log.Tracef(
			nil,
			"{bitbucket} merging pull request: %s / %s / %d -> %s",
			api.project,
			api.repository,
			id,
			result.State,
		)
	}

	return result, nil
}

type BitbucketAddonsAPI struct {
	client stash.Stash
}

func (api *BitbucketAddonsAPI) Install(path string) (string, error) {
	token, err := api.client.GetUPMToken()
	if err != nil {
		return "", karma.Format(
			err,
			"get upm token",
		)
	}

	log.Debugf(
		karma.Describe("upm_token", token),
		"{add-on} installing: %s",
		path,
	)

	key, err := api.client.InstallAddon(token, path)
	if err != nil {
		return "", karma.Format(
			err,
			"install add-on",
		)
	}

	addon, err := api.client.GetAddon(token, key)
	if err != nil {
		return "", karma.Format(
			err,
			"get add-on info",
		)
	}

	err = api.client.EnableAddon(token, addon)
	if err != nil {
		return "", karma.Format(
			err,
			"enable add-on",
		)
	}

	return key, nil
}

func (api *BitbucketAddonsAPI) Uninstall(key string) error {
	token, err := api.client.GetUPMToken()
	if err != nil {
		return karma.Format(
			err,
			"get upm token",
		)
	}

	log.Debugf(
		karma.Describe("upm_token", token),
		"{add-on} uninstalling: %s",
		key,
	)

	err = api.client.UninstallAddon(token, key)
	if err != nil {
		return karma.Format(
			err,
			"uninstall add-on",
		)
	}

	return nil
}

func (api *BitbucketAddonsAPI) Get(key string) (*stash.Addon, error) {
	token, err := api.client.GetUPMToken()
	if err != nil {
		return nil, karma.Format(
			err,
			"get upm token",
		)
	}

	addon, err := api.client.GetAddon(token, key)
	if err != nil {
		return nil, karma.Format(
			err,
			"uninstall add-on",
		)
	}

	return &addon, nil
}

func (api *BitbucketAddonsAPI) SetLicense(addon string, license string) error {
	log.Debugf(
		karma.Describe("license", license),
		"{add-on} setting license: %s",
		addon,
	)

	err := api.client.SetAddonLicense(addon, license)
	if err != nil {
		return karma.
			Describe("license", license).
			Describe("addon", addon).
			Format(
				err,
				"set addon license",
			)
	}

	return nil
}

type BitbucketAdminAPI struct {
	client stash.Stash
}

func (api *BitbucketAdminAPI) CreateUser(
	name, password, email string,
) (*stash.User, error) {
	displayName := name

	user, err := api.client.CreateUser(name, password, displayName, email)
	if err != nil {
		return nil, err
	}

	return &user, nil
}

type BitbucketRepositoryPermissionsAPI struct {
	client     stash.Stash
	project    string
	repository string
}

func (api *BitbucketRepositoryPermissionsAPI) GrantUserPermission(
	user string,
	permission string,
) error {
	return api.client.GrantRepositoryUserPermission(
		api.project,
		api.repository,
		user,
		permission,
	)
}
