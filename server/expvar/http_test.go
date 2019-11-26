package expvar

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapHTTPClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/echo":
			body, _ := ioutil.ReadAll(r.Body)
			fmt.Fprintln(w, string(body))
		case "/hello":
			fmt.Fprintln(w, "1234 Hello")
		}
	}))
	defer ts.Close()

	newRequest := func(path, body string) *http.Request {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, _ := http.NewRequest("GET", ts.URL+path, reader)
		return req
	}

	t.Run("stats", func(t *testing.T) {
		stats := NewUnpublishedStats(nil)

		client := WrapHTTPClient(http.DefaultClient, stats,
			func(r *http.Request) string {
				return "/echo"
			})
		for i := 0; i < 10; i++ {
			req := newRequest("/echo", "1234567890")
			res, err := client.Do(req)
			require.Nil(t, err)
			res.Body.Close()
		}

		endpoint := stats.EnsureEndpoint("/echo")
		require.Equal(t, int64(10), endpoint.Total)
	})
}
