package apollo

import (
	"encoding/json"
	"fmt"
	"github.com/go-chassis/paas-lager"
	"github.com/go-mesh/openlogging"
	"github.com/stretchr/testify/assert"
	"net/http"
	"os"
	"testing"
)

func init() {
	log.Init(log.Config{
		LoggerLevel:   "DEBUG",
		EnableRsyslog: false,
		LogFormatText: true,
		Writers:       []string{"stdout"},
	})
	l := log.NewLogger("test")
	openlogging.SetLogger(l)
}
func TestApolloClient_NewApolloClient(t *testing.T) {

}

func TestApolloClient_HTTPDo(t *testing.T) {
	keepAlive := map[string]interface{}{
		"timeout": "500",
	}
	helper := startHttpServer(":9876", "/test", keepAlive)

	gopath := os.Getenv("GOPATH")
	os.Setenv("CHASSIS_HOME", gopath+"src/github.com/go-chassis/go-chassis/examples/discovery/server/")

	apolloClient := &Client{}
	apolloClient.NewApolloClient()

	// Test existing API 's
	resp, err := apolloClient.HTTPDo("GET", "http://127.0.0.1:9876/test", nil, nil)
	assert.NotEqual(t, resp, nil)
	assert.Equal(t, err, nil)

	// Test Non-existing API's
	resp, err = apolloClient.HTTPDo("GET", "http://127.0.0.1:9876/testUN", nil, nil)
	assert.Equal(t, resp.StatusCode, 404)
	assert.Equal(t, err, nil)

	// Shutdown the helper server gracefully
	if err := helper.Shutdown(nil); err != nil {
		panic(err)
	}
}

func TestApolloClient_PullConfig(t *testing.T) {
	configurations := map[string]interface{}{
		"timeout": "500",
	}
	configBody := map[string]interface{}{
		"appId":          "TestApp",
		"cluster":        "default",
		"namespaceName":  "application",
		"configurations": configurations,
		"releaseKey":     "20180327130726-1dc5027439679153",
	}

	helper := startHttpServer(":9875", "/configs/TestApp/Default/application", configBody)

	gopath := os.Getenv("GOPATH")
	os.Setenv("CHASSIS_HOME", gopath+"src/github.com/go-chassis/go-chassis/examples/discovery/server/")

	apolloClient := &Client{}
	apolloClient.NewApolloClient()

	//Test existing Services
	configResponse, error := apolloClient.PullConfig("TestApp", "1.0", "SampleApp", "Default", "timeout", "")
	assert.NotEqual(t, configResponse, nil)
	assert.Equal(t, error, nil)

	//Test the non-existing Key
	configResponse, error = apolloClient.PullConfig("TestApp", "1.0", "SampleApp", "Default", "non-exsiting", "")
	assert.Contains(t, error.Error(), "No Key found")

	// Test the non-exsisting Service
	configResponse, error = apolloClient.PullConfig("TestApp", "1.0", "SampleApp", "Default", "non-exsiting", "")
	assert.Contains(t, error.Error(), "Bad Response")

	// Shutdown the helper server gracefully
	if err := helper.Shutdown(nil); err != nil {
		panic(err)
	}

}

func TestApolloClient_PullConfigs(t *testing.T) {
	configurations := map[string]interface{}{
		"timeout": "500",
	}
	configBody := map[string]interface{}{
		"appId":          "SampleApp",
		"cluster":        "default",
		"namespaceName":  "application",
		"configurations": configurations,
		"releaseKey":     "20180327130726-1dc5027439679153",
	}

	helper := startHttpServer(":9874", "/configs/SampleApp/Default/application", configBody)

	gopath := os.Getenv("GOPATH")
	os.Setenv("CHASSIS_HOME", gopath+"src/github.com/go-chassis/go-chassis/examples/discovery/server/")

	apolloClient := &Client{}
	apolloClient.NewApolloClient()

	//Test existing Services
	configResponse, error := apolloClient.PullConfigs("SampleApp", "1.0", "SampleApp", "Default")
	assert.NotEqual(t, configResponse, nil)
	assert.Equal(t, error, nil)

	//Test the non-existing Services
	configResponse, error = apolloClient.PullConfigs("SampleApp", "1.0", "SampleApp", "Default")
	assert.Contains(t, error.Error(), "Bad Response")

	// Shutdown the helper server gracefully
	if err := helper.Shutdown(nil); err != nil {
		panic(err)
	}
}

func startHttpServer(port string, pattern string, responseBody map[string]interface{}) *http.Server {
	helper := &http.Server{Addr: port}
	http.HandleFunc(pattern, func(w http.ResponseWriter, r *http.Request) {

		body, _ := json.Marshal(responseBody)
		w.Write(body)
	})

	go func() {
		if err := helper.ListenAndServe(); err != nil {
			fmt.Printf("Httpserver: ListenAndServe() error: %s", err)
		}
	}()
	return helper
}
