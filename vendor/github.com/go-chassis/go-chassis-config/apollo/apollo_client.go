package apollo

import (
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/go-chassis/foundation/httpclient"
	"github.com/go-chassis/go-chassis-config"
	"github.com/go-chassis/go-chassis-config/serializers"
	"github.com/go-mesh/openlogging"
)

// Client contains the implementation of Client
type Client struct {
	name        string
	client      *httpclient.URLClient
	serviceName string
	cluster     string
	namespace   string
	URI         string
}

const (
	apolloServerAPI    = ":ServerURL/configs/:appID/:clusterName/:nameSpace"
	defaultContentType = "application/json"
	//Name of the Plugin
	Name = "apollo"
)

// NewApolloClient init's the necessary objects needed for seamless communication to apollo Server
func (apolloClient *Client) NewApolloClient() {
	options := &httpclient.URLClientOption{
		SSLEnabled: false,
		TLSConfig:  nil, //TODO Analyse the TLS configuration of Apollo Server
		Compressed: false,
	}
	var err error
	apolloClient.client, err = httpclient.GetURLClient(options)
	if err != nil {
		openlogging.GetLogger().Error("Client Initialization Failed: " + err.Error())
	}
	openlogging.GetLogger().Debugf("Client Initialized successfully")
}

// HTTPDo Use http-client package for rest communication
func (apolloClient *Client) HTTPDo(method string, rawURL string, headers http.Header, body []byte) (resp *http.Response, err error) {
	return apolloClient.client.HTTPDo(method, rawURL, headers, body)
}

// PullConfigs is the implementation of Client and pulls all the configuration for a given serviceName
func (apolloClient *Client) PullConfigs(serviceName, version, app, env string) (map[string]interface{}, error) {
	/*
		1. Compose the URL
		2. Make a Http Request to Apollo Server
		3. Unmarshal the response
		4. Return back the configuration/error
		Note: Currently the input to this function in not used, need to check it's feasibility of using it, as the serviceName/version can be different in Apollo
	*/

	// Compose the URL
	pullConfigurationURL := apolloClient.composeURL()

	// Make a Http Request to Apollo Server
	resp, err := apolloClient.HTTPDo("GET", pullConfigurationURL, nil, nil)
	if err != nil {
		openlogging.GetLogger().Error("Error in Querying the Response from Apollo: " + err.Error())
		return nil, err
	}
	if resp.StatusCode != 200 {
		openlogging.GetLogger().Error("Bad Response : " + "Response from Apollo Server " + resp.Status)
		return nil, errors.New("Bad Response from Apollo Server " + resp.Status)
	}
	/*
		Sample Response from Apollo Server
		{
			"appId": "SampleApp",
			"cluster": "default",
			"namespaceName": "application",
			"configurations": {
				"timeout": "500"
			},
			"releaseKey": "20180327130726-1dc5027439679153"
		}
	*/

	//Unmarshal the response
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	var configurations map[string]interface{}
	error := serializers.Decode(defaultContentType, body, &configurations)
	if error != nil {
		openlogging.GetLogger().Error("Error in Unmarshalling the Response from Apollo: " + error.Error())
		return nil, error
	}

	openlogging.GetLogger().Debugf("The Marshaled response of the body is : ", configurations["configurations"])

	var configValues map[string]interface{}
	configValues = configurations["configurations"].(map[string]interface{})
	return configValues, nil
}

// PullConfig is the implementation of the Client
func (apolloClient *Client) PullConfig(serviceName, version, app, env, key, contentType string) (interface{}, error) {
	/*
		1. Compose the URL
		2. Make a Http Request to Apollo Server
		3. Unmarshal the response
		4. Get the particular key/value
		4. Return back the value/error
		//TODO Use the contentType to send the response
	*/

	// Compose the URL
	pullConfigurationURL := apolloClient.composeURL()

	// Make a Http Request to Apollo Server
	resp, err := apolloClient.HTTPDo("GET", pullConfigurationURL, nil, nil)
	if err != nil {
		openlogging.GetLogger().Error("Error in Querying the Response from Apollo: " + err.Error())
		return nil, err
	}
	if resp.StatusCode != 200 {
		openlogging.GetLogger().Error("Bad Response : " + "Response from Apollo Server " + resp.Status)
		return nil, errors.New("Bad Response from Apollo Server " + resp.Status)
	}

	//Unmarshal the response
	var body []byte
	body, err = ioutil.ReadAll(resp.Body)
	var configurations map[string]interface{}
	error := serializers.Decode(defaultContentType, body, &configurations)
	if error != nil {
		openlogging.GetLogger().Error("Error in Unmarshalling the Response from Apollo: " + err.Error())
		return nil, err
	}

	//Find the particular Key
	configList := configurations["configurations"]
	configurationsValue := ""
	isFound := false

	for configKey, configValue := range configList.(map[string]interface{}) {
		if configKey == key {
			configurationsValue = configValue.(string)
			isFound = true
		}
	}

	if !isFound {
		openlogging.GetLogger().Error("Error in fetching the configurations for particular value" + "No Key found : " + key)
		return nil, errors.New("No Key found : " + key)
	}
	openlogging.GetLogger().Debugf("The Key Value of : ", configurationsValue)
	return configurationsValue, nil
}

// composeURL composes the URL based on the configurations given in chassis.yaml
func (apolloClient *Client) composeURL() string {
	pullConfigurationURL := strings.Replace(apolloServerAPI, ":ServerURL", apolloClient.URI, 1)
	pullConfigurationURL = strings.Replace(pullConfigurationURL, ":appID", apolloClient.serviceName, 1)
	pullConfigurationURL = strings.Replace(pullConfigurationURL, ":clusterName", apolloClient.cluster, 1)
	pullConfigurationURL = strings.Replace(pullConfigurationURL, ":nameSpace", apolloClient.namespace, 1)
	return pullConfigurationURL
}

//PullConfigsByDI returns the configuration for additional Projects in Apollo
func (apolloClient *Client) PullConfigsByDI(dimensionInfo, diInfo string) (map[string]map[string]interface{}, error) {
	// TODO Return the configurations for customized Projects in Apollo Configs
	return nil, nil
}

// PushConfigs   not implemented
func (apolloClient *Client) PushConfigs(data map[string]interface{}, dimensionInfo string) (map[string]interface{}, error) {
	return map[string]interface{}{"Result": "not implemented"}, nil
}

// DeleteConfigsByKeys not implemented
func (apolloClient *Client) DeleteConfigsByKeys(keys []string, dimensionInfo string) (map[string]interface{}, error) {
	return map[string]interface{}{"Result": "not implemented"}, nil
}

//InitConfigApollo initialize the Apollo Client
func InitConfigApollo(options config.Options) config.Client {
	apolloClient := &Client{
		serviceName: options.ApolloServiceName,
		cluster:     options.Cluster,
		URI:         options.ServerURI,
		namespace:   options.Namespace,
	}
	apolloClient.NewApolloClient()
	return apolloClient
}
func (apolloClient *Client) Watch(f func(map[string]interface{}), errHandler func(err error)) error {
	return nil
}
func init() {
	config.InstallConfigClientPlugin(Name, InitConfigApollo)
}
