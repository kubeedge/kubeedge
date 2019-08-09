package common

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	security2 "github.com/go-chassis/foundation/security"
	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/pkg/string"
	"github.com/go-chassis/go-chassis/security"
	//this import used for plain cipher
	_ "github.com/go-chassis/go-chassis/security/plugins/plain"
)

//SSLConfig struct stores the necessary info for SSL configuration
type SSLConfig struct {
	CipherPlugin string   `yaml:"cipher_plugin" json:"cipherPlugin"`
	VerifyPeer   bool     `yaml:"verify_peer" json:"verifyPeer"`
	CipherSuites []uint16 `yaml:"cipher_suites" json:"cipherSuits"`
	MinVersion   uint16   `yaml:"min_version" json:"minVersion"`
	MaxVersion   uint16   `yaml:"max_version" json:"maxVersion"`
	CAFile       string   `yaml:"ca_file" json:"caFile"`
	CertFile     string   `yaml:"cert_file" json:"certFile"`
	KeyFile      string   `yaml:"key_file" json:"keyFile"`
	CertPWDFile  string   `yaml:"cert_pwd_file" json:"certPwdFile"`
}

//TLSCipherSuiteMap is a map with key of type string and value of type unsigned integer
var TLSCipherSuiteMap = map[string]uint16{
	"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
}

//TLSVersionMap is a map with key of type string and value of type unsigned integer
var TLSVersionMap = map[string]uint16{
	"TLSv1.0": tls.VersionTLS10,
	"TLSv1.1": tls.VersionTLS11,
	"TLSv1.2": tls.VersionTLS12,
}

//GetX509CACertPool read a certificate file and gets the certificate configuration
func GetX509CACertPool(caCertFile string) (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, fmt.Errorf("read ca cert file %s failed", caCert)
	}

	pool.AppendCertsFromPEM(caCert)
	return pool, nil
}

//LoadTLSCertificate function loads the TLS certificate
func LoadTLSCertificate(certFile, keyFile, passphase string, cipher security2.Cipher) ([]tls.Certificate, error) {
	certContent, err := ioutil.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("read cert file %s failed", certFile)
	}

	keyContent, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return nil, fmt.Errorf("read key file %s failed", keyFile)
	}

	keyBlock, _ := pem.Decode(keyContent)
	if keyBlock == nil {
		return nil, fmt.Errorf("decode key file %s failed", keyFile)
	}

	if x509.IsEncryptedPEMBlock(keyBlock) {
		plainpass, err := cipher.Decrypt(passphase)
		if err != nil {
			return nil, err
		}

		plainPassphaseBytes := stringutil.Str2bytes(plainpass)
		defer stringutil.ClearStringMemory(&plainpass)
		defer stringutil.ClearByteMemory(plainPassphaseBytes)
		keyData, err := x509.DecryptPEMBlock(keyBlock, plainPassphaseBytes)
		if err != nil {
			return nil, fmt.Errorf("decrypt key file %s failed: %s", keyFile, err)
		}

		// 解密成功，重新编码为无加密的PEM格式文件
		plainKeyBlock := &pem.Block{
			Type:  "RSA PRIVATE KEY",
			Bytes: keyData,
		}

		keyContent = pem.EncodeToMemory(plainKeyBlock)
	}

	cert, err := tls.X509KeyPair(certContent, keyContent)
	if err != nil {
		return nil, fmt.Errorf("load X509 key pair from cert file %s with key file %s failed: %s", certFile, keyFile, err)
	}

	var certs []tls.Certificate
	certs = append(certs, cert)

	return certs, nil
}

func getTLSConfig(sslConfig *SSLConfig, role string) (tlsConfig *tls.Config, err error) {
	clientAuthMode := tls.NoClientCert
	var pool *x509.CertPool
	// ca file is needed when veryPeer is true
	if sslConfig.VerifyPeer {
		pool, err = GetX509CACertPool(sslConfig.CAFile)
		if err != nil {
			return nil, err
		}

		clientAuthMode = tls.RequireAndVerifyClientCert
	}

	// if cert pwd file is set, get the pwd
	var keyPassphase []byte
	if sslConfig.CertPWDFile != "" {
		keyPassphase, err = ioutil.ReadFile(sslConfig.CertPWDFile)
		if err != nil {
			return nil, fmt.Errorf("read cert pwd %s failed: %s", sslConfig.CertPWDFile, err)
		}
	}

	// certificate is necessary for server, optional for client
	var certs []tls.Certificate
	if !(role == common.Client && sslConfig.KeyFile == "" && sslConfig.CertFile == "") {
		var cipherPlugin security2.Cipher
		if f, err := security.GetCipherNewFunc(sslConfig.CipherPlugin); err != nil {
			return nil, fmt.Errorf("get cipher plugin [%s] failed, %v", sslConfig.CipherPlugin, err)
		} else if cipherPlugin = f(); cipherPlugin == nil {
			return nil, errors.New("invalid cipher plugin")
		}
		certs, err = LoadTLSCertificate(sslConfig.CertFile, sslConfig.KeyFile, strings.TrimSpace(string(keyPassphase)), cipherPlugin)
		if err != nil {
			return nil, err
		}
	}

	switch role {
	case "server":
		tlsConfig = &tls.Config{
			ClientCAs:                pool,
			Certificates:             certs,
			CipherSuites:             sslConfig.CipherSuites,
			PreferServerCipherSuites: true,
			ClientAuth:               clientAuthMode,
			MinVersion:               sslConfig.MinVersion,
			MaxVersion:               sslConfig.MaxVersion,
		}
	case common.Client:
		tlsConfig = &tls.Config{
			RootCAs:            pool,
			Certificates:       certs,
			CipherSuites:       sslConfig.CipherSuites,
			InsecureSkipVerify: !sslConfig.VerifyPeer,
			MinVersion:         sslConfig.MinVersion,
			MaxVersion:         sslConfig.MaxVersion,
		}
	}

	return tlsConfig, nil
}

//GetClientTLSConfig function gets client side TLS config
func GetClientTLSConfig(sslConfig *SSLConfig) (*tls.Config, error) {
	return getTLSConfig(sslConfig, "client")
}

//GetServerTLSConfig function gets server side TLD config
func GetServerTLSConfig(sslConfig *SSLConfig) (*tls.Config, error) {
	return getTLSConfig(sslConfig, "server")
}

//ParseSSLCipherSuites function parses cipher suites in to a list
func ParseSSLCipherSuites(ciphers string) ([]uint16, error) {
	cipherSuiteList := make([]uint16, 0)
	cipherSuiteNameList := strings.Split(ciphers, ",")
	for _, cipherSuiteName := range cipherSuiteNameList {
		cipherSuiteName = strings.TrimSpace(cipherSuiteName)
		if len(cipherSuiteName) == 0 {
			continue
		}

		if cipherSuite, ok := TLSCipherSuiteMap[cipherSuiteName]; ok {
			cipherSuiteList = append(cipherSuiteList, cipherSuite)
		} else {
			// 配置算法不存在
			return nil, fmt.Errorf("cipher %s not exist", cipherSuiteName)
		}
	}

	return cipherSuiteList, nil
}

//ParseSSLProtocol function parses SSL protocols
func ParseSSLProtocol(sprotocol string) (uint16, error) {
	var result uint16 = tls.VersionTLS12
	if protocol, ok := TLSVersionMap[sprotocol]; ok {
		result = protocol
	} else {
		return result, fmt.Errorf("invalid ssl minimal version invalid(%s)", sprotocol)
	}

	return result, nil
}
