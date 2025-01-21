// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package utils

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapHTTPClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/echo":
			body, _ := io.ReadAll(r.Body)
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

	t.Run("response size limit", func(t *testing.T) {
		client := &http.Client{}
		closed := false
		client = WrapHTTPClient(client,
			WithResponseSizeLimit(2),
			WithResponsePreClose(func(lrc *LimitedReadCloser) error {
				assert.False(t, closed)
				closed = true
				return nil
			}),
		)

		res, err := client.Do(newRequest("/hello", "6789"))
		require.Nil(t, err)
		got, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.Nil(t, err)
		require.True(t, closed)
		require.Equal(t, 2, len(got))
		require.Equal(t, "12", string(got))
	})

	t.Run("request size limit", func(t *testing.T) {
		client := &http.Client{}
		closed := false
		client = WrapHTTPClient(client,
			WithRequestSizeLimit(2),
			WithRequestPreClose(func(lrc *LimitedReadCloser) error {
				assert.False(t, closed)
				closed = true
				return nil
			}),
		)

		req := newRequest("/echo", "6789")
		req.ContentLength = -1
		res, err := client.Do(req)
		require.Nil(t, err)
		got, err := io.ReadAll(res.Body)
		res.Body.Close()
		require.Nil(t, err)
		require.True(t, closed)
		// The response is the 3 characters: 2 from the truncated request body, plus a \n.
		require.Equal(t, 3, len(got))
		require.Equal(t, "67\n", string(got))
	})
}
