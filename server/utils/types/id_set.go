// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package types

import (
	"encoding/json"
)

type ID string

func (id ID) GetID() ID      { return id }
func (id ID) String() string { return string(id) }

type IDArray []ID

func (p IDArray) Len() int             { return len(p) }
func (p IDArray) GetAt(n int) Value    { return p[n] }
func (p IDArray) SetAt(n int, v Value) { p[n] = v.(ID) }
func (p *IDArray) Ref() interface{}    { return p }
func (p IDArray) InstanceOf() ValueArray {
	inst := make(IDArray, 0)
	return &inst
}
func (p *IDArray) Resize(n int) {
	*p = make(IDArray, n)
}

var IDArrayProto = &IDArray{}

type IDSet struct {
	ValueSet
}

func NewIDSet(vv ...ID) *IDSet {
	i := &IDSet{
		ValueSet: *NewValueSet(&IDArray{}),
	}
	for _, v := range vv {
		i.Set(v)
	}
	return i
}

func (i *IDSet) Set(v ID) {
	i.ValueSet.Set(v)
}

func (i *IDSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.IDs())
}

func (i *IDSet) UnmarshalJSON(data []byte) error {
	ids := []ID{}
	err := json.Unmarshal(data, &ids)
	if err != nil {
		return err
	}

	n := NewIDSet(ids...)
	*i = *n
	return nil
}
