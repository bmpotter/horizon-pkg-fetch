// +build ci

package fetch

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

const (
	testMaterialDirName = "test_material"
)

// we'd need to create a dep on anax to not fake this up; make sure to cover
// anax's use of configure HTTPS auth in that project
func fakeHTTPClientFactory(timeoutS *uint) *http.Client {
	var timeout *uint
	if timeoutS == nil {
		t := uint(10)
		timeout = &t
	} else {
		timeout = timeoutS
	}

	return &http.Client{
		Timeout: time.Second * time.Duration(*timeout),
	}
}

func Test_PkgFetch_CI_Suite(suite *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fetch-test-int-")
	assert.Nil(suite, err)
	defer os.RemoveAll(tmpDir)

	destinationDir := path.Join(tmpDir, "destination")

	keysDir, err := filepath.Abs(path.Join(testMaterialDirName, "keys"))
	assert.Nil(suite, err)

	emptyAuth := make(map[string]map[string]string, 0)

	suite.Run("PkgFetch fetches big pkg parts", func(t *testing.T) {
		remoteDomain := "http://1DD40.http.tor01.cdn.softlayer.net/horizon-test-ci"
		remotePkg := "45d9ace6d4a4c88432788aab8a1a01bf2826710d"
		ur, err := url.Parse(fmt.Sprintf("%s/%s.json", remoteDomain, remotePkg))
		assert.Nil(t, err)
		assert.NotNil(t, *ur)

		resp, err := http.Get(fmt.Sprintf("%s/%s.json.sig", remoteDomain, remotePkg))
		assert.Nil(t, err)
		assert.EqualValues(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()

		sigBytes, err := ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)

		keyfile := filepath.Join(keysDir, "public.pem")
		pkgs, err := PkgFetch(fakeHTTPClientFactory, *ur, string(sigBytes), destinationDir, []string{keyfile}, emptyAuth)
		assert.Nil(t, err)

		assert.EqualValues(t, 1, len(pkgs))

		abs, err := filepath.Abs(path.Join(destinationDir, remotePkg, "a05b7b7bb4d82ae5da5299592df3e89699031a7d5bcc6ba75ce45e28dca1898a"))
		assert.Nil(t, err)
		assert.Contains(t, pkgs, abs)
	})

	// TODO: expand these cases, test the edges
}
