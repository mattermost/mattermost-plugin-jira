// Copyright (c) 2019-present Mattermost, Inc. All Rights Reserved.
// See License for license information.

package types

import (
	"encoding/json"
	"sort"
)

type Value interface {
	GetID() ID
}

type Setter interface {
	Set(Value)
}

type Getter interface {
	Get(Value)
}

type ValueArray interface {
	Len() int
	GetAt(int) Value
	SetAt(int, Value)
	InstanceOf() ValueArray
	Ref() interface{}
	Resize(int)
}

type ValueSet struct {
	proto ValueArray
	ids   []ID
	m     map[ID]Value
}

func NewValueSet(proto ValueArray, vv ...Value) *ValueSet {
	i := &ValueSet{
		proto: proto,
	}
	for _, v := range vv {
		i.Set(v)
	}
	return i
}

func (set *ValueSet) From(other *ValueSet) {
	set.proto = other.proto
	set.ids = append([]ID{}, other.ids...)
	set.m = map[ID]Value{}
	for id, v := range other.m {
		set.m[id] = v
	}
}

func (set *ValueSet) Contains(id ID) bool {
	if set.IsEmpty() {
		return false
	}
	_, ok := set.m[id]
	return ok
}

func (set *ValueSet) Delete(toDelete ID) {
	if !set.Contains(toDelete) {
		return
	}

	for n, key := range set.ids {
		if key == toDelete {
			updated := set.ids[:n]
			if n+1 < len(set.ids) {
				updated = append(updated, set.ids[n+1:]...)
			}
			set.ids = updated
		}
	}
	delete(set.m, toDelete)
}

func (set *ValueSet) Get(id ID) Value {
	if set.IsEmpty() {
		return nil
	}
	return set.m[id]
}

func (set *ValueSet) GetAt(n int) Value {
	if set.IsEmpty() {
		return nil
	}
	return set.m[set.ids[n]]
}

func (set *ValueSet) Len() int {
	if set.IsEmpty() {
		return 0
	}
	return len(set.ids)
}

func (set *ValueSet) IDs() []ID {
	if set.IsEmpty() {
		return []ID{}
	}
	n := make([]ID, len(set.ids))
	copy(n, set.ids)
	return n
}

func (set *ValueSet) Set(vv ...Value) {
	if set.ids == nil {
		set.ids = []ID{}
	}
	if set.m == nil {
		set.m = map[ID]Value{}
	}

	for _, v := range vv {
		id := v.GetID()
		if !set.Contains(id) {
			set.ids = append(set.ids, id)
		}
		set.m[id] = v
	}
}

func (set *ValueSet) SetAt(n int, v Value) {
	if set.ids == nil {
		set.ids = []ID{}
	}
	if set.m == nil {
		set.m = map[ID]Value{}
	}
	id := v.GetID()
	if !set.Contains(id) {
		set.ids = append(set.ids, id)
	}
	set.m[id] = v
}

func (set *ValueSet) AsArray(out ValueArray) {
	if set.IsEmpty() {
		out.Resize(0)
		return
	}
	out.Resize(len(set.ids))
	for n, key := range set.ids {
		out.SetAt(n, set.m[key])
	}
}

func (set *ValueSet) IsEmpty() bool {
	if set == nil {
		return true
	}
	return len(set.ids) == 0
}

func (set *ValueSet) TestAsArray(out ValueArray) {
	if set.IsEmpty() {
		out.Resize(0)
		return
	}
	out.Resize(len(set.ids))
	for n, key := range set.ids {
		out.SetAt(n, set.m[key])
	}
}

func (set *ValueSet) TestIDs() []string {
	if set.IsEmpty() {
		return nil
	}
	n := []string{}
	for _, id := range set.IDs() {
		n = append(n, string(id))
	}
	sort.Strings(n)
	return n
}

func (set *ValueSet) MarshalJSON() ([]byte, error) {
	if set.IsEmpty() {
		return []byte("[]"), nil
	}
	proto := set.proto.InstanceOf()
	proto.Resize(len(set.ids))
	for n, id := range set.ids {
		proto.SetAt(n, set.m[id])
	}
	return json.Marshal(proto)
}

func (set *ValueSet) UnmarshalJSON(data []byte) error {
	proto := set.proto.InstanceOf()
	err := json.Unmarshal(data, proto.Ref())
	if err != nil {
		return err
	}

	set.ids = []ID{}
	set.m = map[ID]Value{}
	for n := 0; n < proto.Len(); n++ {
		set.Set(proto.GetAt(n))
	}
	return nil
}
