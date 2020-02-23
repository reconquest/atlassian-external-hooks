package main

import "github.com/reconquest/pkg/log"

type Testcase func(*Suite)

type Testing struct{}

func (testing Testing) Errorf(format string, args ...interface{}) {
	log.Fatalf(nil, "<testify> assertion failed:"+format, args...)
}
