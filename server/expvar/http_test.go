package expvar

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
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

	client := &http.Client{}
	client = WrapHTTPClient(client, utils.ByteSize(2), nil, nil)

	newRequest := func(path, body string) *http.Request {
		var reader io.Reader
		if body != "" {
			reader = strings.NewReader(body)
		}
		req, _ := http.NewRequest("GET", ts.URL+path, reader)
		return req
	}

	t.Run("response size limit", func(t *testing.T) {
		res, err := client.Do(newRequest("/hello", ""))
		require.Nil(t, err)
		got, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		require.Nil(t, err)
		require.Equal(t, 2, len(got))
		require.Equal(t, "12", string(got))
	})

	t.Run("request size limit", func(t *testing.T) {
		req := newRequest("/echo", "6789")
		req.ContentLength = 2
		res, err := client.Do(req)
		require.Nil(t, err)
		got, err := ioutil.ReadAll(res.Body)
		res.Body.Close()
		require.Nil(t, err)
		require.Equal(t, 2, len(got))
		require.Equal(t, "67", string(got))
	})

	t.Run("stats", func(t *testing.T) {
		stats := newStats(nil, true)

		client := WrapHTTPClient(http.DefaultClient, -1, stats,
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
