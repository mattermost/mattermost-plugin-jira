package expvar

import (
	"bytes"
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

type Service struct {
	Name    string
	IsAsync bool

	// A map of all endpoints, plus 2 for "_allResponse" and "_allProcessing".
	endpoints expvar.Map // *Endpoint

	// Cached pointers to the _all Endpoints.
	allResponse   *Endpoint
	allProcessing *Endpoint
}

var _ json.Marshaler = (*Service)(nil)
var _ json.Unmarshaler = (*Service)(nil)

// NewService creates a new Service and registers its expvars.
func NewService(name string, isAsync bool) *Service {
	fmt.Println("<><> expvar.NewService: ", name, isAsync)
	s := &Service{
		Name:    name,
		IsAsync: isAsync,
	}
	s.Init()
	return s
}

func (s *Service) Init() {
	fmt.Println("<><> expvar.Service.Init: ", s.Name, s.IsAsync)
	s.allResponse = s.initEndpointVar(allResponse)
	if s.IsAsync {
		s.allProcessing = s.initEndpointVar(allProcessing)
	}
}

func (s *Service) Reset() {
	endpoints.Range(func(key, value interface{}) bool {
		v := value.(*Endpoint)
		v.Reset()
		return true
	})
}

func (s *Service) initEndpointVar(name string) *Endpoint {
	fmt.Println("<><> expvar.Service.initEndpointVar: ", s.Name, name, s.IsAsync)
	e := NewEndpoint(s.Name + "/" + name)
	s.endpoints.Set(name, e)
	return e
}

// Response records a response event.
func (s *Service) Response(eventName string, size utils.ByteSize, dur time.Duration, isError, isIgnored bool) {
	s.allResponse.Record(size, dur, isError, isIgnored)
	if eventName != "" {
		s.initEndpointVar(eventName+"/"+response).Record(size, dur, isError, false)
	}
}

// Response records a response event.
func (s *Service) Processing(eventName string, dur time.Duration, isError, isIgnored bool) {
	if !s.IsAsync {
		return
	}
	s.allProcessing.Record(0, dur, isError, isIgnored)
	if eventName != "" {
		s.initEndpointVar(eventName+"/"+processing).Record(0, dur, isError, false)
	}
}

// MarshalJSON implements json.Marshaller.
func (s *Service) MarshalJSON() ([]byte, error) {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, `{"Name":%q,"IsAsync":%v,"Endpoints":%s}`,
		s.Name, s.IsAsync, s.endpoints.String())
	return b.Bytes(), nil
}

// UnmarshalJSON implements json.Unmarshaller.
func (s *Service) UnmarshalJSON(data []byte) error {
	fmt.Println("<><> Service.UnmarshalJSON: ", string(data))
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
