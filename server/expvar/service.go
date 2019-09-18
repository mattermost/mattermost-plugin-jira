package expvar

import (
	"encoding/json"
	"expvar"
	"time"

	"github.com/mattermost/mattermost-plugin-jira/server/utils"
)

const all = "_all"
const processing = "processing"
const response = "response"
const allResponse = all + "/" + response
const allProcessing = all + "/" + processing

type Service struct {
	Name    string
	IsAsync bool

	// A map of all endpoints, plus 2 for "_allResponse" and "_allProcessing".
	endpoints expvar.Map // *Endpoint

	// Cached pointers to the _all Endpoints.
	allResponse   *Endpoint
	allProcessing *Endpoint

	// set to false for testing to avoid publishing the expvars
	disablePublish bool
}

var _ json.Marshaler = (*Service)(nil)
var _ json.Unmarshaler = (*Service)(nil)

// NewService creates a new Service and registers its expvars.
func NewService(name string, isAsync bool) *Service {
	return newService(name, isAsync, false)
}

func newService(name string, isAsync bool, disablePublish bool) *Service {
	s := &Service{
		Name:           name,
		IsAsync:        isAsync,
		disablePublish: disablePublish,
	}
	s.Init()
	return s
}

func (s *Service) Init() {
	s.allResponse = s.initEndpointVar(allResponse, s.allResponse)
	if s.IsAsync {
		s.allProcessing = s.initEndpointVar(allProcessing, s.allProcessing)
	}
	s.endpoints.Do(func(kv expvar.KeyValue) {
		e := kv.Value.(*Endpoint)
		s.initEndpointVar(kv.Key, e)
	})
}

func (s *Service) Reset() {
	s.endpoints.Do(func(kv expvar.KeyValue) {
		v := kv.Value.(*Endpoint)
		v.Reset()
	})
}

func (s *Service) initEndpointVar(name string, e *Endpoint) *Endpoint {
	e = initEndpoint(s.Name+"/"+name, e, s.disablePublish)
	s.endpoints.Set(name, e)
	return e
}

// Response records a response event.
func (s *Service) Response(eventName string, size utils.ByteSize, dur time.Duration, isError, isIgnored bool) {
	s.allResponse.Record(size, dur, isError, isIgnored)
	if eventName != "" {
		s.initEndpointVar(eventName+"/"+response, nil).Record(size, dur, isError, isIgnored)
	}
}

// Response records a response event.
func (s *Service) Processing(eventName string, dur time.Duration, isError, isIgnored bool) {
	if !s.IsAsync {
		return
	}
	s.allProcessing.Record(0, dur, isError, isIgnored)
	if eventName != "" {
		s.initEndpointVar(eventName+"/"+processing, nil).Record(0, dur, isError, isIgnored)
	}
}

// MarshalJSON implements json.Marshaller.
func (s *Service) MarshalJSON() ([]byte, error) {
	v := struct {
		Name      string
		IsAsync   bool
		Endpoints map[string]*Endpoint
	}{
		Name:      s.Name,
		IsAsync:   s.IsAsync,
		Endpoints: map[string]*Endpoint{},
	}
	s.endpoints.Do(func(kv expvar.KeyValue) {
		v.Endpoints[kv.Key] = kv.Value.(*Endpoint)
	})
	return json.Marshal(v)
}

// UnmarshalJSON implements json.Unmarshaller.
func (s *Service) UnmarshalJSON(data []byte) error {
	v := struct {
		Name      string
		IsAsync   bool
		Endpoints map[string]*Endpoint
	}{}
	err := json.Unmarshal(data, &v)
	if err != nil {
		return err
	}
	s.Name = v.Name
	s.IsAsync = v.IsAsync

	for k, e := range v.Endpoints {
		switch k {
		case allResponse:
			s.allResponse = e
		case allProcessing:
			s.allProcessing = e
		}
		s.endpoints.Set(k, e)
	}
	return nil
}
