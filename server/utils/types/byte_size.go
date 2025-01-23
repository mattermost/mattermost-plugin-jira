// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package types

import (
	"math"
	"strconv"
	"strings"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

type ByteSize int64

const sizeB = ByteSize(1)
const sizeKb = 1024 * sizeB
const sizeMb = 1024 * sizeKb
const sizeGb = 1024 * sizeMb
const sizeTb = 1024 * sizeGb

var sizeUnits = []ByteSize{sizeTb, sizeGb, sizeMb, sizeKb, sizeB}
var sizeSuffixes = []string{"Tb", "Gb", "Mb", "Kb", "b"}

func (size ByteSize) String() string {
	if size == 0 {
		return "0"
	}

	withCommas := func(in string) string {
		out := ""
		for len(in) > 3 {
			out = "," + in[len(in)-3:] + out
			in = in[:len(in)-3]
		}
		out = in + out
		return out
	}

	for i, u := range sizeUnits {
		if size < u {
			continue
		}
		if u == sizeB {
			return withCommas(strconv.FormatUint(uint64(size), 10)) + sizeSuffixes[i]
		}

		if size > math.MaxInt64/10 {
			return utils.NotAvailable
		}

		s := strconv.FormatUint(uint64((size*10+u/2)/u), 10)
		l := len(s)
		switch {
		case l < 2:
			return utils.NotAvailable
		case s[l-1] == '0':
			return withCommas(s[:l-1]) + sizeSuffixes[i]
		default:
			return withCommas(s[:l-1]) + "." + s[l-1:] + sizeSuffixes[i]
		}
	}
	return utils.NotAvailable
}

func ParseByteSize(str string) (ByteSize, error) {
	u := sizeB
	str = strings.ToLower(str)
	for i, s := range sizeSuffixes {
		if strings.HasSuffix(str, strings.ToLower(s)) {
			str = str[:len(str)-len(s)]
			u = sizeUnits[i]
			break
		}
	}

	str = strings.ReplaceAll(str, ",", "")
	n, err := strconv.ParseInt(str, 10, 64)
	if err == nil {
		return ByteSize(n) * u, nil
	}
	numerr := err.(*strconv.NumError)
	if numerr.Err != strconv.ErrSyntax {
		return 0, err
	}
	fl, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0, err
	}
	return ByteSize(fl * float64(u)), nil
}
