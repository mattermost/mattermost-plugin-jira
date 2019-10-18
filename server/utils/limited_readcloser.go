// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package utils

import (
	"fmt"
	"io"
)

type LimitReadCloser struct {
	ReadCloser io.ReadCloser
	TotalRead  ByteSize
	Limit      ByteSize
	OnClose    func(*LimitReadCloser) error
}

func (r *LimitReadCloser) Read(data []byte) (int, error) {
	if r.Limit <= 0 {
		return 0, io.EOF
	}
	if len(data) > int(r.Limit) {
		data = data[0:r.Limit]
	}
	n, err := r.ReadCloser.Read(data)
	r.Limit -= ByteSize(n)
	r.TotalRead += ByteSize(n)
	return n, err
}

func (r *LimitReadCloser) Close() error {
	if r.OnClose != nil {
		err := r.OnClose(r)
		if err != nil {
			return err
		}
	}
	fmt.Println("<><> LimitReadCloser.Close: 1:", r.ReadCloser)
	return r.ReadCloser.Close()
}
