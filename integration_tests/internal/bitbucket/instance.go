package bitbucket

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kovetskiy/stash"
	cp "github.com/otiai10/copy"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/database"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/docker"
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/pkg/log"
)

type StartupStatus struct {
	State    string
	Progress struct {
		Message    string
		Percentage int
	}
}

type AtlToken struct {
	value   string
	cookies []*http.Cookie
}

type Instance struct {
	id        string
	version   string
	container string
	database  database.Database
	volumes   struct {
		data   string
		shared string
	}
	network string
	ip      string
	opts    struct {
		RunOpts
		//ConfigureOpts
	}

	stacktraceLogs *docker.Logs
	testcaseLogs   *docker.Logs
}

func (instance *Instance) ID() string {
	return instance.id
}

func (instance *Instance) Opts() struct {
	RunOpts
} {
	return instance.opts
}

func (instance *Instance) ConnectorURI(user *stash.User) string {
	var auth *url.Userinfo

	if user == nil {
		auth = url.UserPassword(ADMIN_USERNAME, ADMIN_PASSWORD)
	} else {
		auth = url.UserPassword(user.Name, user.Password)
	}

	url := url.URL{
		Scheme: "http",
		User:   auth,
		Host:   fmt.Sprintf("%s:%d", instance.ip, instance.opts.PortHTTP),
	}

	return url.String()
}

func (instance *Instance) URI(path string) string {
	url := url.URL{
		Scheme: "http",
		Host:   fmt.Sprintf("%s:%d", instance.ip, instance.opts.PortHTTP),
		Path:   path,
	}

	return url.String()
}

func (instance *Instance) ClonePathSSH(repo, project string) string {
	url := url.URL{
		Scheme: "ssh",
		User:   url.User("git"),
		Host:   fmt.Sprintf("%s:%d", instance.ip, instance.opts.PortSSH),
		Path:   fmt.Sprintf("%s/%s.git", strings.ToLower(repo), project),
	}

	return url.String()
}

func (instance *Instance) ClonePathHTTP(repo, project string) string {
	return instance.URI(
		fmt.Sprintf(
			"scm/%s/%s.git",
			strings.ToLower(repo),
			project,
		),
	)
}

func (instance *Instance) Container() string {
	return instance.container
}

func (instance *Instance) IP() string {
	return instance.ip
}

func (instance *Instance) Version() string {
	return instance.version
}

func (instance *Instance) ReadFile(path string) (string, error) {
	return docker.ReadFile(instance.container, path)
}

func (instance *Instance) ListFiles(path string) ([]string, error) {
	execution := exec.New(
		"docker",
		"cp",
		fmt.Sprintf("%s:%s",
			instance.container,
			filepath.Join(instance.ApplicationDataDir(), path),
		),
		"-",
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, karma.Format(
			err,
			"get stdout pipe for docker cp",
		)
	}

	err = execution.Start()
	if err != nil {
		return nil, karma.Format(
			err,
			"start docker cp",
		)
	}

	files := []string{}

	reader := tar.NewReader(stdout)

	for {
		next, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, karma.Format(
				err,
				"read next file from docker cp tar output",
			)
		}

		files = append(
			files,
			strings.TrimPrefix(
				next.Name,
				filepath.Base(path)+"/",
			),
		)
	}

	err = execution.Wait()
	if err != nil {
		return nil, karma.Format(
			err,
			"finalize docker cp",
		)
	}

	// First item is always directory itself.
	return files[1:], nil
}

type File struct {
	Name     string
	Contents string
}

func (instance *Instance) ReadFiles(path string) ([]File, error) {
	execution := exec.New(
		"docker",
		"cp",
		fmt.Sprintf("%s:%s",
			instance.container,
			filepath.Join(instance.ApplicationDataDir(), path),
		),
		"-",
	)

	stdout, err := execution.StdoutPipe()
	if err != nil {
		return nil, karma.Format(
			err,
			"get stdout pipe for docker cp",
		)
	}

	err = execution.Start()
	if err != nil {
		return nil, karma.Format(
			err,
			"start docker cp",
		)
	}

	files := []File{}

	reader := tar.NewReader(stdout)

	for {
		next, err := reader.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, karma.Format(
				err,
				"read next file from docker cp tar output",
			)
		}

		name := strings.TrimPrefix(
			next.Name,
			filepath.Base(path)+"/",
		)

		contents, err := ioutil.ReadAll(reader)
		if err != nil {
			return nil, karma.Format(
				err,
				"read next file contents from the docker cp tar output",
			)
		}

		files = append(
			files,
			File{
				Name:     name,
				Contents: string(contents),
			},
		)
	}

	err = execution.Wait()
	if err != nil {
		return nil, karma.Format(
			err,
			"finalize docker cp",
		)
	}

	// First item is always directory itself.
	return files[1:], nil
}

func (instance *Instance) WriteFile(
	path string,
	content []byte,
	mode os.FileMode,
) error {
	return docker.WriteFile(
		instance.container,
		instance.ApplicationDataDir(),
		path,
		content,
		mode,
	)
}

func (instance *Instance) Stop() error {
	err := exec.New(
		"docker",
		"kill",
		"-s", "INT",
		instance.container,
	).Run()
	if err != nil {
		return karma.Format(
			err,
			"send docker stop",
		)
	}

	return exec.New("docker", "wait", instance.container).Run()
}

func (instance *Instance) RemoveContainer() error {
	return exec.New(
		"docker",
		"rm", "-f",
		instance.container,
	).Run()
}

func (instance *Instance) RemoveVolumes() error {
	return nil
	//return os.RemoveAll(instance.volumes)
}

func (instance *Instance) Configure() error {
	configured, err := instance.isConfigured()
	if err != nil {
		return err
	}

	if configured {
		return nil
	}

	token, err := instance.getAtlToken(nil)
	if err != nil {
		return err
	}

	err = instance.configureLicense(token)
	if err != nil {
		return err
	}

	err = instance.configureAdministrator(token)
	if err != nil {
		return err
	}

	return nil
}

func (instance *Instance) VolumeData() string {
	return instance.volumes.data
}

func (instance *Instance) VolumeShared() string {
	return instance.volumes.shared
}

func (instance *Instance) Network() string {
	return instance.network
}

func (instance *Instance) StacktraceLogs() *docker.Logs {
	return instance.stacktraceLogs
}

func (instance *Instance) Logs(kind LogsKind) *docker.Logs {
	if kind == LOGS_STACKTRACE {
		return instance.stacktraceLogs
	}

	return instance.testcaseLogs
}

func (instance *Instance) WaitLog(
	ctx context.Context,
	kind LogsKind,
	fn func(string) bool,
	duration time.Duration,
) docker.LogWaiter {
	return docker.WaitLog(ctx, instance.Logs(kind), fn, duration)
}

func (instance *Instance) FlushLogs(kind LogsKind) {
	instance.Logs(kind).Flush()
}

func (instance *Instance) getAtlToken(
	response *http.Response,
) (*AtlToken, error) {
	if response == nil {
		var err error

		response, err = http.Get(instance.URI("/setup"))
		if err != nil {
			return nil, karma.Format(
				err,
				"request setup page for bitbucket instance",
			)
		}
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, karma.Format(
			err,
			"read response body from bitbucket setup page",
		)
	}

	matches := regexp.MustCompile(
		`<input type="hidden" name="atl_token" value="([^"]+)">`,
	).FindStringSubmatch(string(body))

	if len(matches) == 0 {
		return nil, karma.Format(
			err,
			"match atl_token from bitbucket setup page",
		)
	}

	return &AtlToken{
		value:   matches[1],
		cookies: response.Cookies(),
	}, nil
}

func (instance *Instance) postSetupForm(
	form url.Values,
	token *AtlToken,
) error {
	form.Set("atl_token", token.value)

	request, err := http.NewRequest(
		http.MethodPost,
		instance.URI("/setup"),
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return karma.Format(
			err,
			"create http request",
		)
	}

	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	for _, cookie := range token.cookies {
		request.AddCookie(cookie)
	}

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return karma.Format(
			err,
			"post setup form to bitbucket instance",
		)
	}

	if response.StatusCode != 200 {
		return karma.
			Describe("code", response.StatusCode).
			Reason(
				"unexpected status code after post to setup form",
			)
	}

	return nil
}

func (instance *Instance) isConfigured() (bool, error) {
	client := http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	request, err := http.NewRequest(
		http.MethodGet,
		instance.URI("/setup"),
		nil,
	)
	if err != nil {
		return false, karma.Format(
			err,
			"create http request",
		)
	}

	response, err := client.Do(request)
	if err != nil {
		return false, karma.Format(
			err,
			"check is bitbucket already configured or not",
		)
	}

	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return true, nil
	} else {
		return false, nil
	}
}

func (instance *Instance) configureLicense(token *AtlToken) error {
	form := url.Values{}
	form.Set("step", "settings")
	form.Set("license", LICENSE_DATACENTER_3H)
	form.Set("applicationTitle", "Bitbucket")
	form.Set("baseUrl", instance.URI("/"))

	return instance.postSetupForm(form, token)
}

func (instance *Instance) configureAdministrator(token *AtlToken) error {
	form := url.Values{}
	form.Set("step", "user")
	form.Set("username", ADMIN_USERNAME)
	form.Set("fullname", ADMIN_DISPLAY_NAME)
	form.Set("email", ADMIN_EMAIL)
	form.Set("password", ADMIN_PASSWORD)
	form.Set("confirmPassword", ADMIN_PASSWORD)

	return instance.postSetupForm(form, token)
}

func (instance *Instance) ApplicationDataDir() string {
	return BITBUCKET_DATA_DIR
}

func (instance *Instance) create() error {
	type M map[string]interface{}

	libNative := filepath.Join(instance.VolumeData(), "lib", "native")

	err := cp.Copy("integration_tests/assets/bitbucket-data-dir-lib", libNative)
	if err != nil {
		return karma.Format(err, "copy jdbc drivers to bitbucket volume")
	}

	for _, dir := range []string{
		instance.VolumeShared(),
		instance.VolumeData(),
	} {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return karma.Format(err, "create directory: "+dir)
		}
	}

	propertiesPath := filepath.Join(
		instance.VolumeShared(),
		"bitbucket.properties",
	)

	_, err = os.Stat(propertiesPath)
	if os.IsNotExist(err) {
		log.Debugf(
			karma.
				Describe("container", instance.container).
				Describe("path", propertiesPath).
				Describe("properties", instance.opts.Properties.String()),
			"write bitbucket.properties",
		)

		err = ioutil.WriteFile(
			propertiesPath,
			[]byte(instance.opts.Properties.String()),
			0644,
		)
		if err != nil {
			return karma.Format(err, "write bitbucket.properties")
		}
	}

	for _, dir := range []string{instance.VolumeData(), instance.VolumeShared()} {
		_ = exec.New("chmod", "-R", "0777", dir).NoLog().Run()
	}

	rootCA, err := getRootCA()
	if err != nil {
		return karma.Format(err, "get root CA")
	}

	springApplicationConfig, _ := json.Marshal(M{
		"logging": M{
			"logger": M{
				"com.ngs.stash.externalhooks": "debug",
			},
		},
	})

	var (
		jdbcDriver   = instance.database.Driver()
		jdbcURL      = instance.database.URL()
		jdbcUser     = instance.database.User()
		jdbcPassword = instance.database.Password()
	)

	var userName = "integration_tester";

	var initScript = []string{
		"set -euo pipefail",
		"echo 'rootCA.pem' >> /etc/ca-certificates.conf",
		"update-ca-certificates",
		// we need same UID/GID so we can access shared & data BB dirs from host during ugprade process
		fmt.Sprintf("groupadd -g %d %s", os.Getgid(), userName),
		fmt.Sprintf("useradd -u %d -g %d %s", os.Getuid(), os.Getgid(), userName),
		fmt.Sprintf("export RUN_USER=%s", userName),
		"exec /entrypoint.py", // exec is required to propagate INT signal from docker kill
	}

	execution := exec.New(
		"docker", "container", "create",
		// "--add-host=marketplace.atlassian.com:127.0.0.1",
		"--network", instance.opts.Network,
		"-e", "JDBC_DRIVER="+jdbcDriver,
		"-e", "JDBC_URL="+jdbcURL,
		"-e", "JDBC_USER="+jdbcUser,
		"-e", "JDBC_PASSWORD="+jdbcPassword,
		"-e", "ELASTICSEARCH_ENABLED=false",
		"-e", "SEARCH_ENABLED=false", // starting with bitbucket 8
		"-e", fmt.Sprintf(
			`SPRING_APPLICATION_JSON=%s`,
			string(springApplicationConfig),
		),
		// required for Oracle
		"-e", "TZ=Europe/Moscow",
		"-v", fmt.Sprintf(
			"%s:%s",
			instance.VolumeData(),
			instance.ApplicationDataDir(),
		),
		"-v", fmt.Sprintf(
			"%s:%s",
			instance.VolumeShared(),
			filepath.Join(instance.ApplicationDataDir(), "shared"),
		),
		"-v", fmt.Sprintf(
			"%s:%s",
			filepath.Join(rootCA, "rootCA.pem"),
			"/usr/share/ca-certificates/rootCA.pem",
		),
		"--name", instance.container,
		fmt.Sprintf(BITBUCKET_IMAGE, instance.version),
		"bash", "-c",
		strings.Join(initScript, ";"),
	)

	err = execution.Run()
	if err != nil {
		return err
	}

	return instance.start()
}

func (instance *Instance) start() error {
	execution := exec.New("docker", "container", "start", instance.container)
	return execution.Run()
}

func (instance *Instance) connect() error {
	execution := exec.New(
		"docker",
		"inspect",
		"--type", "container",
		"-f",
		"{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}",
		instance.container,
	)

	stdout, _, err := execution.Output()
	if err != nil {
		return err
	}

	ips := strings.Split(strings.TrimSpace(string(stdout)), "\n")
	if len(ips) == 0 {
		return karma.
			Describe("container", instance.container).
			Format(
				err,
				"no ip addresses found on container",
			)
	}

	instance.ip = ips[0]

	return nil
}

func (instance *Instance) getStartupStatus() (*StartupStatus, error) {
	request, err := http.NewRequest(
		http.MethodGet,
		instance.URI("/system/startup"),
		nil,
	)
	if err != nil {
		return nil, karma.Format(
			err,
			"create http request",
		)
	}

	request.Header.Set("Accept", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		if err, ok := err.(*url.Error); ok {
			if _, ok := err.Err.(*net.OpError); ok {
				// skip network error while bitbucket is starting
				return nil, nil
			}

			if err.Err == io.EOF {
				// skip incomplete reads
				return nil, nil
			}

			if err.Err.Error() == "http: server closed idle connection" {
				return nil, nil
			}
		}

		return nil, karma.Format(
			err,
			"request startup status",
		)
	}

	defer response.Body.Close()

	var status StartupStatus

	err = json.NewDecoder(response.Body).Decode(&status)
	if err != nil {
		return nil, karma.Format(
			err,
			"decode startup status",
		)
	}

	return &status, nil
}

func getRootCA() (string, error) {
	cmd := exec.New("mkcert", "-CAROOT")

	err := cmd.Run()
	if err != nil {
		return "", err
	}

	stdout, err := ioutil.ReadAll(cmd.GetStdout())
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(stdout)), nil
}
