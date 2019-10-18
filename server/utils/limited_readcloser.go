// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import (
	"fmt"
	"io"
)

type LimitReadCloser struct {
	inner io.ReadCloser

	TotalRead ByteSize
	limit     ByteSize
}

func NewLimitReadCloser(inner io.ReadCloser, limit int64) *LimitReadCloser {
	fmt.Println("<><> 1: inner:", inner)
	return &LimitReadCloser{
		inner: inner,
		limit: ByteSize(limit),
	}
}

func (r *LimitReadCloser) Read(data []byte) (int, error) {
	if r.inner == nil {
		fmt.Println("<><> 2")
		return 0, io.EOF
	}
	if r.limit <= 0 {
		return 0, io.EOF
	}
	if len(data) > int(r.limit) {
		data = data[0:r.limit]
	}
	n, err := r.inner.Read(data)
	r.limit -= ByteSize(n)
	r.TotalRead += ByteSize(n)
	return n, err
}

func (r *LimitReadCloser) Close() error {
	if r.inner == nil {
		return nil
	}
	return r.inner.Close()
}
