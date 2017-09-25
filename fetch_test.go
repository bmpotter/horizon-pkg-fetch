package fetch

import (
	"net/http"
	"testing"
)

func Test_FetchPool_Suite(t *testing.T) {
	t.Run("FetchPool ", func(t *testing.T) {
		_, err := NewFetchPool("/tmp", func(d string) *http.Client { return nil })
		if err != nil {
			t.Error(err)
		}
	})
}
