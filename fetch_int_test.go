// +build integration

package fetch

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
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
	urlPath             = "/api/v2/images/common"
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
			pkg.Parts[id].Sources[0] = horizonpkg.PartSource{fmt.Sprintf("%s%s/%s/%s.tgz", serverURL, urlPath, pkg.ID, id)}
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
	} else {
		timeout = timeoutS
	}

	return &http.Client{
		Timeout: time.Second * time.Duration(*timeout),
	}
}

func Test_PkgFetch_Suite(suite *testing.T) {
	tmpDir, err := ioutil.TempDir("", "fetch-test-int-")
	assert.Nil(suite, err)
	defer os.RemoveAll(tmpDir)

	router := mux.NewRouter()
	router.PathPrefix(urlPath).Handler(http.StripPrefix(urlPath, http.FileServer(http.Dir(fmt.Sprintf("%v/srv", tmpDir)))))

	// serve out of tmpDir, setup will change content of the Pkg to match the ad-hoc server set up here
	server := httptest.NewServer(router)
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
		resp, err := http.Get(fmt.Sprintf("%s%s/%s.json.sig", server.URL, urlPath, pkgID))
		assert.Nil(t, err)

		assert.EqualValues(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get(fmt.Sprintf("%s%s/%s/%s", server.URL, urlPath, pkgID, "ce623bdd773c7527b48a1d9ce7ccd6b6cffee4a6e16849d061bd55c2c455b8fc.tgz"))
		assert.Nil(t, err)

		assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	})

	suite.Run("PkgFetch pkg signature verification argument", func(t *testing.T) {
		ur, err := url.Parse(fmt.Sprintf("%s%s/%s.json", server.URL, urlPath, pkgID))
		assert.Nil(t, err)
		assert.NotNil(t, *ur)

		keyfile := filepath.Join(keysDir, "public.pem")
		_, err = PkgFetch(fakeHTTPClientFactory, nil, *ur, "", destinationDir, []string{keyfile}, emptyAuth)
		assert.NotNil(t, err)
	})

	suite.Run("PkgFetch fetches served Pkg files and content, verifies them", func(t *testing.T) {
		ur, err := url.Parse(fmt.Sprintf("%s%s/%s.json", server.URL, urlPath, pkgID))
		assert.Nil(t, err)
		assert.NotNil(t, *ur)

		resp, err := http.Get(fmt.Sprintf("%s%s/%s.json.sig", server.URL, urlPath, pkgID))
		assert.Nil(t, err)
		assert.EqualValues(t, http.StatusOK, resp.StatusCode)
		defer resp.Body.Close()

		sigBytes, err := ioutil.ReadAll(resp.Body)
		assert.Nil(t, err)

		keyfile := filepath.Join(keysDir, "public.pem")
		pkgs, err := PkgFetch(fakeHTTPClientFactory, nil, *ur, string(sigBytes), destinationDir, []string{keyfile}, emptyAuth)
		assert.Nil(t, err)

		assert.EqualValues(t, 2, len(pkgs))

		abspath, exists := pkgs["alpine:3.5"]
		assert.True(t, exists)
		assert.True(t, strings.HasSuffix(abspath, "29ef6969f5cc871153e6a00ec197bb071ce8ceae/ab42a0b95e1f1b6addd36256482a9dd034565a962ac792f89d6bd99694d34d92"))

	})

	// TODO: expand these cases, test the edges
}
