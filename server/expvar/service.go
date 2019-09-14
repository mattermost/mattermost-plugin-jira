package expvar

import (
	"encoding/json"
	"expvar"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

const all = "_all"
const processing = "processing"
const response = "response"
const allResponse = all + "/" + response
const allProcessing = all + "/" + processing

// Service exposes `<name>/Response` expvar
type Service interface {
	Response(name string, size utils.ByteSize, elapsed time.Duration, isError, isIgnored bool)
}

// AsyncService exposes `<name>/Response` and `<name>/Processing`Endpoint expvars
type AsyncService interface {
	Service
	Processing(name string, elapsed time.Duration, isError, isIgnored bool)
}

type service struct {
	Name string

	// A map of all sndpoints, plus one for "_all"
	endpoints expvar.Map

	// A cached pointer to the _all Endpoints
	allResponse   *Endpoint
	allProcessing *Endpoint
}

// NewService creates a new Service and registers its expvars
func NewService(name string) Service {
	return newService(name, false)
}

// NewAsyncService creates a new AsyncService and registers its expvars
func NewAsyncService(name string) AsyncService {
	return newService(name, true)
}

func newService(name string, async bool) *service {
	s := &service{
		Name: name,
	}
	s.allResponse = s.useAPI(allResponse)
	if async {
		s.allProcessing = s.useAPI(allProcessing)
	}
	return s
}

// MarshalJSON implements json.Marshaller
func (s *service) MarshalJSON() ([]byte, error) {
	return []byte(s.endpoints.String()), nil
}

// UnmarshalJSON implements json.Unmarshaller
func (s *service) UnmarshalJSON(data []byte) error {
	m := map[string]*Endpoint{}
	err := json.Unmarshal(data, &m)
	if err != nil {
		return err
	}
	for k, v := range m {
		switch k {
		case allResponse:
			s.allResponse = v
		case allProcessing:
			s.allProcessing = v
		}
		s.endpoints.Set(k, v)
	}
	return nil
}

func (s *service) useAPI(name string) *Endpoint {
	e := NewEndpoint(s.Name + "/" + name)
	s.endpoints.Set(name, e)
	return e
}

// Response records a response event
func (s *service) Response(endpointName string, size utils.ByteSize, dur time.Duration, isError, isIgnored bool) {
	fmt.Println("<><> expvar service.Response: ", endpointName)
	s.allResponse.Record(size, dur, isError, isIgnored)
	if endpointName != "" {
		s.useAPI(endpointName+"/"+response).Record(size, dur, isError, false)
	}
}

// Response records a response event
func (s *service) Processing(endpointName string, dur time.Duration, isError, isIgnored bool) {
	s.allProcessing.Record(0, dur, isError, isIgnored)
	if endpointName != "" {
		s.useAPI(endpointName+"/"+processing).Record(0, dur, isError, false)
	}
}
