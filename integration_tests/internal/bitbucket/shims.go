package bitbucket

import (
	"io/ioutil"
	"log"

	"github.com/kovetskiy/stash"
)

func init() {
	stash.Log = log.New(ioutil.Discard, "", 0)
	log.SetFlags(0)
}
