package stats

import (
	"expvar"
)

// WithCallback creates a new expvar from a callback function.
func WithCallback(name string, f func() interface{}) {
	expvar.Publish(name, expvar.Func(f))
}
