// Copyright (c) 2017-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIDSet(t *testing.T) {
	t.Run("happy", func(t *testing.T) {
		in := NewIDSet("test1", "test2")

		data, err := json.Marshal(in)
		require.NoError(t, err)
		require.Equal(t, `["test1","test2"]`, string(data))

		out := NewIDSet()
		err = json.Unmarshal(data, &out)
		require.NoError(t, err)
		require.EqualValues(t, in.TestIDs(), out.TestIDs())
	})
}
