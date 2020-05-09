// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package types

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

type testValue struct {
	ID   ID
	Data string
}

func (si testValue) GetID() ID { return si.ID }

type testValueArray []testValue

func (p testValueArray) Len() int             { return len(p) }
func (p testValueArray) GetAt(n int) Value    { return p[n] }
func (p testValueArray) SetAt(n int, v Value) { p[n] = v.(testValue) }

func (p testValueArray) InstanceOf() ValueArray {
	inst := make(testValueArray, 0)
	return &inst
}
func (p *testValueArray) Ref() interface{} { return &p }
func (p *testValueArray) Resize(n int) {
	*p = make(testValueArray, n)
}

func TestValueSetJSON(t *testing.T) {
	t.Run("strings", func(t *testing.T) {
		in := NewValueSet(IDArrayProto, ID("test1"), ID("test2"))

		data, err := json.Marshal(in)
		require.NoError(t, err)
		require.Equal(t, `["test1","test2"]`, string(data))

		out := NewValueSet(IDArrayProto)
		err = json.Unmarshal(data, &out)
		require.NoError(t, err)

		var ain, aout IDArray
		in.TestAsArray(&ain)
		out.TestAsArray(&aout)
		require.EqualValues(t, ain, aout)
	})
	t.Run("structs", func(t *testing.T) {
		proto := &testValueArray{}
		in := NewValueSet(proto,
			testValue{
				ID:   "id2",
				Data: "data2",
			},
			testValue{
				ID:   "id3",
				Data: "data3",
			},
			testValue{
				ID:   "id1",
				Data: "data1",
			},
		)

		data, err := json.Marshal(in)
		require.NoError(t, err)
		require.Equal(t, `[{"ID":"id2","Data":"data2"},{"ID":"id3","Data":"data3"},{"ID":"id1","Data":"data1"}]`, string(data))

		out := NewValueSet(proto)
		err = json.Unmarshal(data, &out)
		require.NoError(t, err)

		var ain, aout testValueArray
		in.TestAsArray(&ain)
		out.TestAsArray(&aout)
		require.EqualValues(t, ain, aout)
	})
}
