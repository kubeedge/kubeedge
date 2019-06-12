package configcenter

import (
	"crypto/tls"
	"errors"
	"fmt"

	"github.com/go-chassis/foundation/httpclient"
	"github.com/go-chassis/go-chassis-config/serializers"
	"github.com/go-mesh/openlogging"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"sync"
)

//MemberDiscovery is a interface
type MemberDiscovery interface {
	ConfigurationInit([]string) error
	GetConfigServer() ([]string, error)
	RefreshMembers() error
	Shuffle() error
	GetWorkingConfigCenterIP([]string) ([]string, error)
}

//MemDiscovery is a struct
type MemDiscovery struct {
	ConfigServerAddresses []string
	//Logger                *log.Entry
	IsInit     bool
	TLSConfig  *tls.Config
	TenantName string
	EnableSSL  bool
	sync.RWMutex
	client *httpclient.URLClient
}

// pullConfigurationsFromServer pulls all the configuration from Config-Server based on dimesionInfo
func (memDis *MemDiscovery) pullConfigurationsFromServer(dimensionInfo string) (map[string]interface{}, error) {
	type GetConfigAPI map[string]map[string]interface{}
	config := make(map[string]interface{})
	configAPIRes := make(GetConfigAPI)
	parsedDimensionInfo := strings.Replace(dimensionInfo, "#", "%23", -1)
	restApi := ConfigPath + "?" + dimensionsInfo + "=" + parsedDimensionInfo
	err := memDis.call(http.MethodGet, restApi, nil, nil, &configAPIRes)
	if err != nil {
		openlogging.GetLogger().Error("Pull config failed:" + err.Error())
		return nil, err
	}
	for _, v := range configAPIRes {
		for key, value := range v {
			config[key] = value

		}
	}

	return config, nil
}

//Shuffle is a method to log error
func (memDis *MemDiscovery) Shuffle() error {
	if memDis.ConfigServerAddresses == nil || len(memDis.ConfigServerAddresses) == 0 {
		err := errors.New(emptyConfigServerConfig)
		openlogging.GetLogger().Error(emptyConfigServerConfig)
		return err
	}

	perm := rand.Perm(len(memDis.ConfigServerAddresses))

	memDis.Lock()
	defer memDis.Unlock()
	openlogging.GetLogger().Debugf("Before Suffled member %s ", memDis.ConfigServerAddresses)
	for i, v := range perm {
		openlogging.GetLogger().Debugf("shuffler %d %d", i, v)
		tmp := memDis.ConfigServerAddresses[v]
		memDis.ConfigServerAddresses[v] = memDis.ConfigServerAddresses[i]
		memDis.ConfigServerAddresses[i] = tmp
	}

	openlogging.GetLogger().Debugf("Suffled member %s", memDis.ConfigServerAddresses)
	return nil
}

//GetWorkingConfigCenterIP is a method which gets working configuration center IP
func (memDis *MemDiscovery) GetWorkingConfigCenterIP(entryPoint []string) ([]string, error) {
	return entryPoint, nil

}

//ConfigurationInit is a method for creating a configuration
func (memDis *MemDiscovery) ConfigurationInit(initConfigServer []string) error {
	if memDis.IsInit == true {
		return nil
	}

	if memDis.ConfigServerAddresses == nil {
		if initConfigServer == nil && len(initConfigServer) == 0 {
			err := errors.New(emptyConfigServerConfig)
			openlogging.GetLogger().Error(emptyConfigServerConfig)
			return err
		}

		memDis.ConfigServerAddresses = make([]string, 0)
		for _, server := range initConfigServer {
			memDis.ConfigServerAddresses = append(memDis.ConfigServerAddresses, server)
		}

		memDis.Shuffle()
	}

	memDis.IsInit = true
	return nil
}

//GetConfigServer is a method used for getting server configuration
func (memDis *MemDiscovery) GetConfigServer() ([]string, error) {
	if memDis.IsInit == false {
		err := errors.New(packageInitError)
		openlogging.GetLogger().Error(packageInitError)
		return nil, err
	}

	if len(memDis.ConfigServerAddresses) == 0 {
		err := errors.New(emptyConfigServerMembers)
		openlogging.GetLogger().Error(emptyConfigServerMembers)
		return nil, err
	}

	if autoDiscoverable {
		err := memDis.RefreshMembers()
		if err != nil {
			openlogging.GetLogger().Error("refresh member is failed: " + err.Error())
			return nil, err
		}
	} else {
		tmpConfigAddrs := memDis.ConfigServerAddresses
		for key := range tmpConfigAddrs {
			if !strings.Contains(memDis.ConfigServerAddresses[key], "https") && memDis.EnableSSL {
				memDis.ConfigServerAddresses[key] = `https://` + memDis.ConfigServerAddresses[key]

			} else if !strings.Contains(memDis.ConfigServerAddresses[key], "http") {
				memDis.ConfigServerAddresses[key] = `http://` + memDis.ConfigServerAddresses[key]
			}
		}
	}

	err := memDis.Shuffle()
	if err != nil {
		openlogging.GetLogger().Error("member shuffle is failed: " + err.Error())
		return nil, err
	}

	memDis.RLock()
	defer memDis.RUnlock()
	openlogging.GetLogger().Debugf("member server return %s", memDis.ConfigServerAddresses[0])
	return memDis.ConfigServerAddresses, nil
}

//RefreshMembers is a method
func (memDis *MemDiscovery) RefreshMembers() error {
	return nil
}

func (memDis *MemDiscovery) call(method string, api string, headers http.Header, body []byte, s interface{}) error {
	hosts, err := memDis.GetConfigServer()
	if err != nil {
		openlogging.GetLogger().Error("Get config server addr failed:" + err.Error())
	}
	index := rand.Int() % len(memDis.ConfigServerAddresses)
	host := hosts[index]
	rawUri := host + api
	errMsgPrefix := fmt.Sprintf("Call %s failed: ", rawUri)
	resp, err := memDis.HTTPDo(method, rawUri, headers, body)
	if err != nil {
		openlogging.GetLogger().Error(errMsgPrefix + err.Error())
		return err

	}
	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		openlogging.GetLogger().Error(errMsgPrefix + err.Error())
		return err
	}
	if !isStatusSuccess(resp.StatusCode) {
		err = fmt.Errorf("statusCode: %d, resp body: %s", resp.StatusCode, body)
		openlogging.GetLogger().Error(errMsgPrefix + err.Error())
		return err
	}
	contentType := resp.Header.Get("Content-Type")
	if len(contentType) > 0 && (len(defaultContentType) > 0 && !strings.Contains(contentType, defaultContentType)) {
		err = fmt.Errorf("content type not %s", defaultContentType)
		openlogging.GetLogger().Error(errMsgPrefix + err.Error())
		return err
	}
	err = serializers.Decode(defaultContentType, body, s)
	if err != nil {
		openlogging.GetLogger().Error("Decode failed:" + err.Error())
		return err
	}
	return nil
}

//HTTPDo Use http-client package for rest communication
func (memDis *MemDiscovery) HTTPDo(method string, rawURL string, headers http.Header, body []byte) (resp *http.Response, err error) {
	if len(headers) == 0 {
		headers = make(http.Header)
	}
	for k, v := range GetDefaultHeaders(memDis.TenantName) {
		headers[k] = v
	}
	return memDis.client.HTTPDo(method, rawURL, headers, body)
}
