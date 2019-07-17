package main

import (
	"encoding/json"
	"sort"
)

type Set map[string]bool

func (a *Set) UnmarshalJSON(b []byte) error {
	var elems []string
	if err := json.Unmarshal(b, &elems); err != nil {
		return err
	}

	*a = NewSet(elems...)

	return nil
}

func (a Set) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.Elems(false))
}

func NewSet(elems ...string) Set {
	result := Set{}

	for _, elem := range elems {
		result[elem] = true
	}

	return result
}

func (a Set) Elems(sorted bool) []string {
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

func (a Set) Len() int {
	return len(a)
}

func (a Set) Add(elems ...string) Set {
	result := Set{}

	for elem := range a {
		result[elem] = true
	}
	for _, elem := range elems {
		result[elem] = true
	}

	return result
}

func (a Set) Union(b Set) Set {
	result := Set{}

	for elem := range a {
		result[elem] = true
	}
	for elem := range b {
		result[elem] = true
	}

	return result
}

func (a Set) Intersection(b Set) Set {
	result := Set{}

	for elem := range a {
		if b.ContainsAny(elem) {
			result[elem] = true
		}
	}

	return result
}

func (a Set) ContainsAny(elems ...string) bool {
	for _, elem := range elems {
		if _, exists := a[elem]; exists {
			return true
		}
	}

	return false
}

func (a Set) Equals(b Set) bool {
	if a.Len() != b.Len() {
		return false
	}

	i := a.Intersection(b)
	return i.Len() == a.Len()
}
