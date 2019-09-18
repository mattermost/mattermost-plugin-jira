package expvar

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mattermost/mattermost-plugin-jira/server/utils"
	"github.com/stretchr/testify/require"
)

func TestService(t *testing.T) {
	name := fmt.Sprintf("test_%v", uuid.New().String())
	service := newService(name, true, false)
	require.NotNil(t, service)

	for _, s := range sample {
		service.Response("myapi", utils.ByteSize(s.size), time.Duration(s.elapsed)*time.Millisecond, s.isError, s.isIgnored)
	}

	checkSample(t, service.allResponse)
	v := service.endpoints.Get("myapi/response")
	require.NotNil(t, v)
	ep := v.(*Endpoint)
	checkSample(t, ep)

	data, err := service.MarshalJSON()
	require.NoError(t, err)

	service = &Service{}
	err = json.Unmarshal(data, &service)
	require.NoError(t, err)
	service.Init()

	checkSample(t, service.allResponse)
	v = service.endpoints.Get("myapi/response")
	require.NotNil(t, v)
	ep = v.(*Endpoint)
	require.Equal(t, sampleJSON, ep.String())
}
