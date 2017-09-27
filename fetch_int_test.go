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
	"reflect"
	"testing"
	"time"
)

const (
	testMaterialDirName = "test_material"
	pkgID               = "0c5f9ebc62ea666ed60d4e9e308dee3505fcf65c"
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
		pkg.Parts[id].Sources[0] = horizonpkg.PartSource{fmt.Sprintf("%s/%s/%s.tar.gz", serverURL, pkg.ID, id)}
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

	suite.Run("Confirm testMaterialDir is available and pkg metadata is readable", func(t *testing.T) {
		assert.EqualValues(t, pkgID, pkg.ID)
	})

	suite.Run("Confirm HTTP handler serves test material", func(t *testing.T) {
		resp, err := http.Get(fmt.Sprintf("%s/%s.json.sig", server.URL, pkgID))
		assert.Nil(t, err)

		assert.EqualValues(t, http.StatusOK, resp.StatusCode)

		resp, err = http.Get(fmt.Sprintf("%s/%s/%s.tar.gz", server.URL, pkgID, "2cad91d0395c3b75e209509c018058c3735589694988223aa207f1e238fb33cf"))
		assert.Nil(t, err)

		assert.EqualValues(t, http.StatusOK, resp.StatusCode)
	})

	suite.Run("PkgFetch fetches Pkg metadata file", func(t *testing.T) {
		url, err := url.Parse(fmt.Sprintf("%s/%s.json", server.URL, pkgID))
		assert.Nil(t, err)

		_, err = PkgFetch(fakeHTTPClientFactory, url, tmpDir, path.Join(testMaterialDirName, "keys"))
		assert.NotNil(t, err)

		t.Logf("******** %T", err)

		if reflect.TypeOf(err).Name() != "VerificationError" {
			t.Errorf("Expected VerificationError type, got: %T. Error: %v", err, err)
		}
	})

	//suite.Run("", func(t *testing.T) {})

	//suite.Run("", func(t *testing.T) {})
	//suite.Run("", func(t *testing.T) {})
}
