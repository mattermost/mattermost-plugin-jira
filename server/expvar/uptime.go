package expvar

import (
	"expvar"
	"time"
)

var startedAt = time.Now()

func init() {
	expvar.Publish("uptime", expvar.Func(func() interface{} {
		up := (time.Since(startedAt) + time.Second/2) / time.Second * time.Second
		return up.String()
	}))
}
