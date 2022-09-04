package bitbucket

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

const (
	// https://developer.atlassian.com/platform/marketplace/timebomb-licenses-for-testing-server-apps/
	LICENSE_DATACENTER_3H = `` +
		`AAABrQ0ODAoPeNp9kVFvmzAQx9/9KSztLZIJZIu0RkJqA6yNViAK0G3d+uDApXgjNrKPbPn2dYG0a` +
		`6fuwS8+393v//O7vAMa8yN1PerOFrPZYn5GL+OczlzvI0m6/RZ0uisMaON7LgmURF5iwvfgVy3XW` +
		`pj6nGPDjRFcOqXaE4Pc1M61KEEayI8t9I+DNI6jTbC6uP73wd/FdafLmhsIOYL/yMDcOXM98p95Y` +
		`yn60wp97PvW769OpFHMRfMWagb6AHoV+svLs5x9LW4+sM+3t1ds6XpfRkw7jwcgEbSPugOSdVtTa` +
		`tGiUHK4mUwmSZqzT+mGrTdpWAT5Kk1YkUW24AcaLFBFt0eKNdARlUayVBVo2mr1E0qk32vE9sdiO` +
		`r1XzgvEaTN0MBg67hwaKioV0koY1GLbIdjJwlBUtOwMqr39KYfY1JZZclm+9jLEsmbEAZ4CBJvoI` +
		`o9Ctvz2CP2GrRHe6irkL6l+S5JFiW8Pm7suSfU9l8LwXkwIB2hUaxPmYPAUm/Q2bP315w5MGXL95` +
		`DmEZ839jFEE3SlNedvS6rTCkOjAm25YvOON3fMAVTj4nTAtAhRH4o+fI5MQ7xSh2mtA1bPJrq0WA` +
		`gIVAIGperR8m2N0fl/GfUUJfQnd+T1aX02kk`

	ADMIN_EMAIL        = `we@reconquest.io`
	ADMIN_DISPLAY_NAME = `Admin`
	ADMIN_USERNAME     = `admin`
	ADMIN_PASSWORD     = `admin`
)

type Properties map[string]string

func NewProperties() Properties {
	return make(Properties)
}

// This method doesn't really work because we can't figure out baseURL before
// starting the container
//func (properties Properties) WithSysadmin() Properties {
//    properties["setup.displayName"] = "Bitbucket"
//    properties["setup.sysadmin.username"] = ADMIN_USERNAME
//    properties["setup.sysadmin.password"] = ADMIN_PASSWORD
//    properties["setup.sysadmin.displayName"] = ADMIN_DISPLAY_NAME
//    properties["setup.sysadmin.emailAddress"] = ADMIN_EMAIL
//    return properties
//}

func (props Properties) WithSidecarMeshEnabled(value bool) Properties {
	props["plugin.bitbucket-git.mesh.sidecar.enabled"] = strconv.FormatBool(
		value,
	)

	return props
}

func (props Properties) WithLicense(license string) Properties {
	props["setup.license"] = license
	return props
}

func (props Properties) WithHazelcast() Properties {
	props["hazelcast.network.multicast"] = "true"
	props["hazelcast.group.name"] = "bitbucket"
	props["hazelcast.group.password"] = "bitbucket"
	return props
}

func (props Properties) String() string {
	var keys []string
	for key := range props {
		keys = append(keys, key)
	}

	sort.Strings(keys)

	var lines []string
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, props[key]))
	}

	return strings.Join(lines, "\n")
}
