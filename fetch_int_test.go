// +build integration

package fetch

import (
	"encoding/json"
	"fmt"
	"github.com/open-horizon/horizon-pkg-fetch/horizonpkg"
	"github.com/open-horizon/rsapss-tool/sign"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	testMaterialDirName = "test_material"
	pkgID               = "29ef6969f5cc871153e6a00ec197bb071ce8ceae"
)

func fromTestMaterialDir(pp string, t *testing.T) []byte {
	testMaterialDir, err := filepath.Abs(testMaterialDirName)
	assert.Nil(t, err)

	raw, err := ioutil.ReadFile(path.Join(testMaterialDir, pp))
	return raw
}

func setup(t *testing.T, tmpDir string, serverURL string) *horizonpkg.Pkg {
	raw := fromTestMaterialDir(fmt.Sprintf("%v.json", pkgID), t)

	var pkg horizonpkg.Pkg
	err := json.Unmarshal(raw, &pkg)
	assert.Nil(t, err)

	// modify pkg so the paths match our server's domain and port
	for id, _ := range pkg.Parts {
		// only modify those with scheme and domain, ignore the absolute path source URLs
		if strings.HasPrefix(pkg.Parts[id].Sources[0].URL, "http") {
			pkg.Parts[id].Sources[0] = horizonpkg.PartSource{fmt.Sprintf("%s/%s/%s.tgz", serverURL, pkg.ID, id)}
		}
	}

	bytes, err := json.Marshal(pkg)
	assert.Nil(t, err)

	err = os.Mkdir(fmt.Sprintf("%s/srv", tmpDir), 0770)
	assert.Nil(t, err)

	tmpPkgFile := fmt.Sprintf("%s/srv/%s.json", tmpDir, pkgID)

	// write it out
	err = ioutil.WriteFile(tmpPkgFile, bytes, 0666)
	assert.Nil(t, err)

	// and sign the pkg file content
	pkgSig, err := sign.Input(fmt.Sprintf("%s/keys/private/private.key", testMaterialDirName), bytes)
	assert.Nil(t, err)

	pkgSigFile := fmt.Sprintf("%s.sig", tmpPkgFile)
	err = ioutil.WriteFile(pkgSigFile, []byte(pkgSig), 0644)
	assert.Nil(t, err)

	contentDirAbs, err := filepath.Abs(fmt.Sprintf("%s/%s", testMaterialDirName, pkg.ID))
	assert.Nil(t, err)

	// now link all of the served content to the tmpDir
	os.Symlink(contentDirAbs, fmt.Sprintf("%s/srv/%s", tmpDir, pkg.ID))
	return &pkg
}

// we'd need to create a dep on anax to not fake this up; make sure to cover
// anax's use of configure HTTPS auth in that project
func fakeHTTPClientFactory(timeoutS *uint) *http.Client {
	var timeout *uint
	if timeoutS == nil {
		t := uint(10)
		timeout = &t
	}

	return &http.Client{
		Timeout: time.Second * time.Duration(*timeout),
	}
}

func Test_PkgFetch_Suite(suite *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fetch-test-int-")
	assert.Nil(suite, err)
	//defer os.RemoveAll(tmpDir)

	// serve out of tmpDir, setup will change content of the Pkg to match the ad-hoc server set up here
	server := httptest.NewServer(http.FileServer(http.Dir(fmt.Sprintf("%v/srv", tmpDir))))
	defer server.Close()

	pkg := setup(suite, tmpDir, server.URL)

	destinationDir := path.Join(tmpDir, "destination")

	keysDir, err := filepath.Abs(path.Join(testMaterialDirName, "keys"))
	assert.Nil(suite, err)

	emptyAuth := make(map[string]map[string]string, 0)

	suite.Run("Confirm testMaterialDir is available and pkg metadata is readable", func(t *testing.T) {
		assert.EqualValues(t, pkgID, pkg.ID)
	})

	suite.Run("Confirm HTTP handler serves test material", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/%s.json.sig", server.URL, pkgID))
		assert.Nil(t, err)

		assert.EqualValues(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get(fmt.Sprintf("%s/%s/%s", server.URL, pkgID, "ce623bdd773c7527b48a1d9ce7ccd6b6cffee4a6e16849d061bd55c2c455b8fc.tgz"))
		assert.Nil(t, err)

		assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	})

	suite.Run("PkgFetch pkg signature verification argument", func(t *testing.T) {
		ur, err := url.Parse(fmt.Sprintf("%s/%s.json", server.URL, pkgID))
		assert.Nil(t, err)
		assert.NotNil(t, *ur)

		_, err = PkgFetch(fakeHTTPClientFactory, *ur, "", destinationDir, "", keysDir, emptyAuth)
		assert.NotNil(t, err)
	})

	suite.Run("PkgFetch fetches served Pkg files and content, verifies them", func(t *testing.T) {
		ur, err := url.Parse(fmt.Sprintf("%s/%s.json", server.URL, pkgID))
		assert.Nil(t, err)
		assert.NotNil(t, *ur)

		resp, err := http.Get(fmt.Sprintf("%s/%s.json.sig", server.URL, pkgID))
		assert.Nil(t, err)
		assert.EqualValues(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()

		sigBytes, err := ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)

		pkgs, err := PkgFetch(fakeHTTPClientFactory, *ur, string(sigBytes), destinationDir, "", keysDir, emptyAuth)
		assert.Nil(t, err)

		assert.EqualValues(t, 2, len(pkgs))

		// get whatever comes up first
		var id string
		for id, _ = range pkg.Parts {
			break
		}

		abs, err := filepath.Abs(path.Join(destinationDir, pkg.ID, id))
		assert.Nil(t, err)
		assert.Contains(t, pkgs, abs)
	})

	// TODO: expand these cases, test the edges
}
