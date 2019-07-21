package tls

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/config"
	secCommon "github.com/go-chassis/go-chassis/security/common"

	"github.com/go-chassis/go-chassis/third_party/forked/k8s.io/apimachinery/pkg/util/sets"
)

var errSSLConfigNotExist = errors.New("No SSL config")
var useDefaultSslTag = sets.NewString(
	"registry.Consumer.",
	"configcenter.Consumer.",
	"monitor.Consumer.",
	"serviceDiscovery.Consumer.",
	"registrator.Consumer.",
	"contractDiscovery.Consumer.",
	"router.Consumer",
)

func hasDefaultSslTag(tag string) bool {
	if len(tag) == 0 {
		return false
	}

	if useDefaultSslTag.Has(tag) {
		return true
	}
	return false
}

func getDefaultSslConfigMap() map[string]string {
	cipherSuits := []string{}
	for k := range secCommon.TLSCipherSuiteMap {
		cipherSuits = append(cipherSuits, k)
	}

	cipherSuitesKey := strings.Join(cipherSuits, ",")
	defaultSslConfigMap := map[string]string{
		common.SslCipherPluginKey: "default",
		common.SslVerifyPeerKey:   common.FALSE,
		common.SslCipherSuitsKey:  cipherSuitesKey,
		common.SslProtocolKey:     "TLSv1.2",
		common.SslCaFileKey:       "",
		common.SslCertFileKey:     "",
		common.SslKeyFileKey:      "",
		common.SslCertPwdFileKey:  "",
	}
	return defaultSslConfigMap
}

func getSSLConfigMap(tag string) map[string]string {
	sslConfigMap := config.GlobalDefinition.Ssl
	defaultSslConfigMap := getDefaultSslConfigMap()
	result := make(map[string]string)

	sslSet := false
	if tag != "" {
		tag = tag + `.`
	}

	for k, v := range defaultSslConfigMap {
		// 使用默认配置
		result[k] = v
		// 若配置了全局配置项，则覆盖默认配置
		if r, exist := sslConfigMap[k]; exist && r != "" {
			result[k] = r
			sslSet = true
		}
		// 若配置了指定交互方的配置项，则覆盖全局配置
		keyWithTag := tag + k
		if v, exist := sslConfigMap[keyWithTag]; exist && v != "" {
			result[k] = v
			sslSet = true
		}
	}
	// 未设置ssl 且不提供内部默认ss配置 返回空字典
	if !sslSet && !hasDefaultSslTag(tag) {
		return make(map[string]string)
	}

	return result
}

func parseSSLConfig(sslConfigMap map[string]string) (*secCommon.SSLConfig, error) {
	sslConfig := &secCommon.SSLConfig{}
	var err error

	sslConfig.CipherPlugin = sslConfigMap[common.SslCipherPluginKey]

	sslConfig.VerifyPeer, err = strconv.ParseBool(sslConfigMap[common.SslVerifyPeerKey])
	if err != nil {
		return nil, err
	}

	sslConfig.CipherSuites, err = secCommon.ParseSSLCipherSuites(sslConfigMap[common.SslCipherSuitsKey])
	if err != nil {
		return nil, err
	}
	if len(sslConfig.CipherSuites) == 0 {
		return nil, fmt.Errorf("No valid cipher")
	}

	sslConfig.MinVersion, err = secCommon.ParseSSLProtocol(sslConfigMap[common.SslProtocolKey])
	if err != nil {
		return nil, err
	}
	sslConfig.MaxVersion = secCommon.TLSVersionMap["TLSv1.2"]
	sslConfig.CAFile = sslConfigMap[common.SslCaFileKey]
	sslConfig.CertFile = sslConfigMap[common.SslCertFileKey]
	sslConfig.KeyFile = sslConfigMap[common.SslKeyFileKey]
	sslConfig.CertPWDFile = sslConfigMap[common.SslCertPwdFileKey]

	return sslConfig, nil
}

// GetSSLConfigByService get ssl configurations based on service
func GetSSLConfigByService(svcName, protocol, svcType string) (*secCommon.SSLConfig, error) {
	tag, err := generateSSLTag(svcName, protocol, svcType)
	if err != nil {
		return nil, err
	}

	sslConfigMap := getSSLConfigMap(tag)
	if len(sslConfigMap) == 0 {
		return nil, errSSLConfigNotExist
	}

	sslConfig, err := parseSSLConfig(sslConfigMap)
	if err != nil {
		return nil, err
	}
	return sslConfig, nil
}

// GetDefaultSSLConfig get default ssl configurations
func GetDefaultSSLConfig() *secCommon.SSLConfig {
	sslConfigMap := getDefaultSslConfigMap()
	sslConfig, _ := parseSSLConfig(sslConfigMap)
	return sslConfig
}

// generateSSLTag generate ssl tag
func generateSSLTag(svcName, protocol, svcType string) (string, error) {
	var tag string
	if svcName != "" {
		tag = tag + "." + svcName
	}
	if protocol != "" {
		tag = tag + "." + protocol
	}
	if tag == "" {
		return "", errors.New("Service name and protocol can't be empty both")
	}

	switch svcType {
	case common.Consumer, common.Provider:
		tag = tag + "." + svcType
	default:
		return "", fmt.Errorf("Service type not support: %s, must be: %s|%s",
			svcType, common.Provider, common.Consumer)
	}

	return tag[1:], nil
}

// GetTLSConfigByService get tls configurations based on service
func GetTLSConfigByService(svcName, protocol, svcType string) (*tls.Config, *secCommon.SSLConfig, error) {
	sslConfig, err := GetSSLConfigByService(svcName, protocol, svcType)
	if err != nil {
		return nil, nil, err
	}

	var tlsConfig *tls.Config
	switch svcType {
	case common.Provider:
		tlsConfig, err = secCommon.GetServerTLSConfig(sslConfig)
	case common.Consumer:
		tlsConfig, err = secCommon.GetClientTLSConfig(sslConfig)
	default:
		err = fmt.Errorf("service type not support: %s, must be: %s|%s",
			svcType, common.Provider, common.Consumer)
	}
	if err != nil {
		return nil, sslConfig, err
	}

	return tlsConfig, sslConfig, nil
}

// IsSSLConfigNotExist check the status of ssl configurations
func IsSSLConfigNotExist(e error) bool {
	return e == errSSLConfigNotExist
}

// GetTLSConfig returns tls config from scheme and type
func GetTLSConfig(scheme, t string) (*tls.Config, error) {
	var tlsConfig *tls.Config
	secure := scheme == common.HTTPS
	if secure {
		sslTag := t + "." + common.Consumer
		tmpTLSConfig, _, err := GetTLSConfigByService(t, "", common.Consumer)
		if err != nil {
			if IsSSLConfigNotExist(err) {
				return nil, fmt.Errorf("%s tls mode, but no ssl config", sslTag)
			}
			return nil, fmt.Errorf("Load %s TLS config failed", sslTag)
		}
		tlsConfig = tmpTLSConfig
	}
	return tlsConfig, nil
}
