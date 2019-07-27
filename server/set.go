package main

import (
	"encoding/json"
	"sort"
)

type StringSet map[string]bool

func (a *StringSet) UnmarshalJSON(b []byte) error {
	var elems []string
	if err := json.Unmarshal(b, &elems); err != nil {
		return err
	}

	*a = NewStringSet(elems...)

	return nil
}

func (a StringSet) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Elems(false))
}

func NewStringSet(elems ...string) StringSet {
	result := StringSet{}

	for _, elem := range elems {
		result[elem] = true
	}

	return result
}

func (a StringSet) Elems(sorted bool) []string {
	result := make([]string, len(a))

	i := 0
	for key := range a {
		result[i] = key
		i++
	}

	if sorted {
		sort.Strings(result)
	}

	return result
}

func (a StringSet) Len() int {
	return len(a)
}

func (a StringSet) Add(elems ...string) StringSet {
	result := StringSet{}

	for elem := range a {
		result[elem] = true
	}
	for _, elem := range elems {
		result[elem] = true
	}

	return result
}

func (a StringSet) Subtract(elems ...string) StringSet {
	result := StringSet{}

	b := NewStringSet(elems...)

	for elem := range a {
		if !b.ContainsAny(elem) {
			result[elem] = true
		}
	}

	return result
}

func (a StringSet) Union(b StringSet) StringSet {
	result := StringSet{}

	for elem := range a {
		result[elem] = true
	}
	for elem := range b {
		result[elem] = true
	}

	return result
}

func (a StringSet) Intersection(b StringSet) StringSet {
	result := StringSet{}

	for elem := range a {
		if b.ContainsAny(elem) {
			result[elem] = true
		}
	}

	return result
}

func (a StringSet) ContainsAny(elems ...string) bool {
	for _, elem := range elems {
		if _, exists := a[elem]; exists {
			return true
		}
	}

	return false
}

func (a StringSet) Equals(b StringSet) bool {
	if a.Len() != b.Len() {
		return false
	}

	i := a.Intersection(b)
	return i.Len() == a.Len()
}
