// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import (
	"io"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLimitReadCloser(t *testing.T) {
	inner := ioutil.NopCloser(strings.NewReader("01234567890"))

	totalRead := ByteSize(0)
	r := &LimitReadCloser{
		ReadCloser: inner,
		Limit:      8,
		OnClose: func(rr *LimitReadCloser) error {
			totalRead = rr.TotalRead
			return io.EOF
		},
	}
	data := make([]byte, 10)

	n, err := r.Read(data[0:4])
	require.Nil(t, err)
	require.Equal(t, 4, n)
	require.Equal(t, "0123", string(data[0:4]))

	n, err = r.Read(data[0:5])
	require.Nil(t, err)
	// Note, truncated to 4, total 8
	require.Equal(t, 4, n)
	require.Equal(t, "4567", string(data[0:4]))

	n, err = r.Read(data[0:1])
	require.Equal(t, io.EOF, err)
	require.Equal(t, 0, n)

	err = r.Close()
	require.Equal(t, io.EOF, err)
	require.Equal(t, ByteSize(8), totalRead)
}
