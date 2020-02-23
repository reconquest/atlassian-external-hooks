package runner

import (
	"github.com/reconquest/pkg/log"
	"github.com/stretchr/testify/assert"
)

type Suite func(*Runner, *assert.Assertions)

type Testing struct{}

func (testing Testing) Errorf(format string, args ...interface{}) {
	log.Fatalf(nil, "<testify> assertion failed:"+format, args...)
}
