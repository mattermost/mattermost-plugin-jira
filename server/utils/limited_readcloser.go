// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import "io"

type LimitReadCloser struct {
	TotalRead ByteSize

	inner io.ReadCloser
	limit ByteSize
}

func NewLimitReadCloser(inner io.ReadCloser, limit int64) *LimitReadCloser {
	return &LimitReadCloser{
		inner: inner,
		limit: ByteSize(limit),
	}
}

func (r *LimitReadCloser) Close() error {
	return r.inner.Close()
}

func (r *LimitReadCloser) Read(data []byte) (int, error) {
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
