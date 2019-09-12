package stats

import (
	"expvar"
	"fmt"
	"strings"
	"time"

	"github.com/circonus-labs/circonusllhist"
)

// Client registers expvar variables for each API name, and "_" for the overall
// counters.
type Client struct {
	APIs Endpoints 

	All *Endpoint
}

func NewClient(kind string) *Client {
	prefix := "client/" + kind + "/"
	return &Client{
		All:      EnsureEndpoint(prefix + "_"),
		APIs: map[string],
	}
}
