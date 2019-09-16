package expvar

import (
	"expvar"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewService(t *testing.T) {
	name := fmt.Sprintf("test_%v", uuid.New().String())
	e := NewService(name, true)
	require.NotNil(t, e)
	require.NotNil(t, expvar.Get(name+"/_all/response"))
	require.NotNil(t, expvar.Get(name+"/_all/processing"))

	name = fmt.Sprintf("test_%v", uuid.New().String())
	e = NewService(name, false)
	require.NotNil(t, e)
	require.NotNil(t, expvar.Get(name+"/_all/response"))
	require.Nil(t, expvar.Get(name+"/_all/processing"))
}

func TestInitEndpointVar(t *testing.T) {
	for _, isAsync := range []bool{true, false} {
		t.Run(fmt.Sprint(isAsync),
			func(t *testing.T) {
				name := fmt.Sprintf("test_%v", uuid.New().String())
				e := NewService(name, isAsync)
				require.NotNil(t, e)
				require.Equal(t, e.IsAsync, isAsync)
				e.Response("myapi", 100, 100, false, false)
				e.Processing("myapi", 100, false, false)
				require.NotNil(t, expvar.Get(name+"/myapi/response"))
				require.Equal(t, isAsync, expvar.Get(name+"/myapi/processing") != nil)
				assert.Nil(t, expvar.Get(name+"/myapi"))
				assert.Nil(t, expvar.Get(name+"/response"))
				assert.Nil(t, expvar.Get(name+"/processing"))

				assert.NotEqual(t, "{}", expvar.Get(name+"/myapi/response").String())
				if isAsync {
					assert.NotEqual(t, "{}", expvar.Get(name+"/myapi/processing").String())
				}
			})
	}
}
