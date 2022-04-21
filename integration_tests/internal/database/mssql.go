package database

import (
	"github.com/reconquest/atlassian-external-hooks/integration_tests/internal/exec"
	"github.com/reconquest/karma-go"
	"github.com/reconquest/lexec-go"
)

type Mssql struct {
	container string
}

func (mssql *Mssql) Driver() string {
	return "com.microsoft.sqlserver.jdbc.SQLServerDriver"
}

func (mssql *Mssql) URL() string {
	return "jdbc:sqlserver://" + mssql.container + ":1433;databaseName=bitbucket"
}

func (mssql *Mssql) User() string {
	return "bitbucket"
}

func (mssql *Mssql) Password() string {
	return "bitbucket"
}

func (mssql *Mssql) Container() string {
	return mssql.container
}

func (mssql *Mssql) Volume() string {
	return mssql.container
}

const mssqlSuperUserPassword = "saStrong(!)Password"

func (mssql *Mssql) start() error {
	execution := exec.New(
		"docker",
		"run",
		"--name", mssql.container,
		"-d",
		"-v", mssql.container+":/var/opt/mssql",
		"-e", "ACCEPT_EULA=Y",
		"-e", "SA_PASSWORD="+mssqlSuperUserPassword,
		"-e", "MSSQL_PID=Enterprise",
		"mcr.microsoft.com/mssql/server:2019-CU15-ubuntu-20.04",
	)
	err := execution.Run()
	if err != nil {
		return karma.Format(err, "start mssql container")
	}

	ip, err := inspectIP(mssql.container)
	if err != nil {
		return karma.Format(err, "inspect ip address")
	}

	waitService(ip, 1433, "mssql")

	err = mssql.createUserAndDatabase()
	if err != nil {
		return karma.Format(err, "create user and database")
	}

	return nil
}

func (mssql *Mssql) createUserAndDatabase() error {
	execution := exec.New(
		"docker", "exec", "-i", mssql.container,
		"/opt/mssql-tools/bin/sqlcmd",
		"-S", "localhost",
		"-U", "sa",
		"-P", mssqlSuperUserPassword,
	)

	err := execution.Start()
	if err != nil {
		return err
	}

	stdin := execution.GetStdin()

	write := func(line string) {
		_, err = stdin.Write([]byte(line + "\n"))
	}

	write("CREATE DATABASE bitbucket")
	write("GO")
	write("USE bitbucket")
	write("GO")
	write("ALTER DATABASE bitbucket SET ALLOW_SNAPSHOT_ISOLATION ON")
	write("GO")
	write("ALTER DATABASE bitbucket SET READ_COMMITTED_SNAPSHOT ON")
	write("GO")
	write("ALTER DATABASE bitbucket COLLATE SQL_Latin1_General_CP1_CS_AS")
	write("GO")
	write("SET NOCOUNT OFF")
	write("GO")
	write("USE master")
	write("GO")
	write("CREATE LOGIN bitbucket WITH PASSWORD=N'bitbucket', " +
		"DEFAULT_DATABASE=bitbucket, CHECK_EXPIRATION=OFF, CHECK_POLICY=OFF")
	write("GO")
	write("ALTER AUTHORIZATION ON DATABASE::bitbucket TO bitbucket")
	write("GO")

	if err != nil {
		return karma.Format(err, "write sql to create user/database")
	}

	err = stdin.Close()
	if err != nil {
		return karma.Format(err, "close stdin")
	}

	return execution.Wait()
}

func (mssql *Mssql) Stop() error {
	return stop(mssql.container)
}

func (mssql *Mssql) RemoveContainer() error {
	return removeContainer(mssql.container)
}

func newMssql(id string) (*Mssql, error) {
	instance := &Mssql{container: id + "-mssql"}

	ip, err := inspectIP(instance.container)
	switch {
	case err != nil && lexec.IsExitStatus(err):
		err := instance.start()
		if err != nil {
			return nil, karma.Format(err, "start new mssql instance")
		}

	case err != nil && !lexec.IsExitStatus(err):
		return nil, karma.Format(err, "docker inspect container")

	default:
		// if we found ip it means it's already started/starting, so we ensure
		// it's running and ready for bitbucket
		waitService(ip, 1433, "mssql")

		err = instance.createUserAndDatabase()
		if err != nil {
			return nil, karma.Format(err, "create user and database")
		}
	}

	return instance, nil
}
