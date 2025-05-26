// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package utils

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mattermost/mattermost-plugin-jira/server/utils/types"
)

func TestLimitReadCloser(t *testing.T) {
	inner := io.NopCloser(strings.NewReader("01234567890"))

	totalRead := types.ByteSize(0)
	r := NewLimitedReadCloser(inner, 8, func(lrc *LimitedReadCloser) error {
		totalRead = lrc.TotalRead
		return io.EOF
	})
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
	require.Equal(t, types.ByteSize(8), totalRead)
}
