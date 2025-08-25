package bitbucket

import (
	"io/ioutil"
	"log"

	"github.com/reconquest/stash-go"
)

func init() {
	stash.Log = log.New(ioutil.Discard, "", 0)
	log.SetFlags(0)
}
