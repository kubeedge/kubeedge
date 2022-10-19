package certretrieval

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt"
	vault "github.com/hashicorp/vault/api"
	auth "github.com/hashicorp/vault/api/auth/kubernetes"
	"k8s.io/klog/v2"
	"moul.io/http2curl"
)

const (
	// The canonical path of a service account token in a running k8s pod
	ServiceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

var (
	ErrConfig    = fmt.Errorf("configuration error")
	ErrRetrieval = fmt.Errorf("retrieval error")
)

// Config is the configuration struct for the certrieval
type Config struct {
	// Token is the Vault token that can be passed directly. It is evaluated first.
	// If set, Tokenfile is ignored.
	Token string `json:"token,omitempty"`
	// Tokenfile is the path to the file containing the Vault token. It get's evaluated second only if
	// Token is not set. If Token and Tokenfile are not set, the service account token is used.
	Tokenfile string `json:"tokenfile,omitempty"`
	// Address is the URL of the Vault server, e.g. "https://vault.example.com:8200"
	Address string `json:"vault"`
	// ServerCA is the CA certificate of the Vault server
	ServerCA string `json:"serverca,omitempty"`
	// PKI is the path to the PKI engine in Vault
	PKI string `json:"pki"`
	// Role is the Vault role to use
	Role string `json:"role"`
	// AuthRole is the Vault role to use for authentication
	AuthRole string `json:"authrole"`
	// Name is the name of the certificate to retrieve, e.g. "myservice.example.com"
	Name string `json:"name"`
	// AltNames specifies requested Subject Alternative Names, in a comma-delimited list.
	// These can be host names or email addresses; they will be parsed into their respective fields.
	// If any requested names do not match role policy, the entire request will be denied.
	AltNames string `json:"alt_names,omitempty"`
	// ValidityCheckTolerance is the tolerance in percent for the validity check
	ValidityCheckTolerance int64 `json:"validity_check_tolerance"`
	// Force ignores the validity check and forces retrieval
	Force bool `json:"force"`
	// TTL specifies requested Time To Live for the certificate. Cannot be greater than the role's max_ttl value.
	// If not provided, the role's ttl value will be used.
	TTL time.Duration `json:"ttl,omitempty"`
	// OutCAfile is the path to the file to store the CA certificate
	OutCAfile string `json:"outcafile"`
	// OutCertfile is the path to the file to store the certificate
	OutCertfile string `json:"outcertfile"`
	// OutKeyfile is the path to the file to store the private key
	OutKeyfile string `json:"outkeyfile"`
}

// Validate the configuration to catch problems early.
func (c Config) Validate() error {
	var errors []error
	if c.Tokenfile == "" && c.Token == "" {
		_, err := os.Stat(ServiceAccountPath)
		if err != nil {
			// check for not exist is not required anymore: Even if the file
			// existed, it could not be read anyway
			errors = append(errors, fmt.Errorf("token not found. Checked tokenfile path, env variable and service account path: %v", err))
		}
	}

	if c.AltNames != "" {
		r := regexp.MustCompile(`^([\w+\.\*-]+)(,[\w+\.\*-]+)*$`)
		if !r.MatchString(c.AltNames) {
			errors = append(errors, fmt.Errorf("AltNames must be a comma separated list of DNS names"))
		}
	}

	if c.PKI == "" {
		errors = append(errors, fmt.Errorf("PKI not defined"))
	}

	if c.Address == "" {
		errors = append(errors, fmt.Errorf("vault not defined"))
	}

	if c.Role == "" {
		errors = append(errors, fmt.Errorf("role not defined"))
	}

	if c.Name == "" {
		errors = append(errors, fmt.Errorf("name not defined"))
	}

	if c.OutCAfile == "" {
		errors = append(errors, fmt.Errorf("outCAfile not defined"))
	}

	if c.OutCertfile == "" {
		errors = append(errors, fmt.Errorf("outCertfile not defined"))
	}

	if c.OutKeyfile == "" {
		errors = append(errors, fmt.Errorf("outKeyfile not defined"))
	}

	if c.ValidityCheckTolerance < 0 || 100 < c.ValidityCheckTolerance {
		errors = append(errors, fmt.Errorf("checktolerance must be between 0 and 100"))
	}

	if errors != nil {
		return fmt.Errorf("%w: errors in configuration: %s", ErrConfig, errors)
	}

	return nil
}

// CertRetrieval manages the retrieval and replacement of certificates
type CertRetrieval struct {
	Config
}

// New creates a new CertRetrieval type
func New(config Config) (*CertRetrieval, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	return &CertRetrieval{Config: config}, nil
}

// UnixTime is a wrapper type for time.Time. This allows marshalling and
// unmarshalling JSON representations
type UnixTime time.Time

func (ut UnixTime) MarshalJSON() (data []byte, err error) {
	fmt := strconv.FormatInt(time.Time(ut).Unix(), 10)
	return []byte(fmt), nil
}

func (ut *UnixTime) UnmarshalJSON(data []byte) error {
	n, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		return err
	}
	*ut = UnixTime(time.Unix(n, 0))

	return nil
}

// StringList is a wrapper for a string slice with suitable json marshalling
// when the value is not expressed as a JSON array
type StringList []string

func (sl StringList) MarshalJSON() ([]byte, error) {
	return []byte(strings.Join(sl, ",")), nil
}

func (sl *StringList) UnmarshalJSON(data []byte) error {
	copy(*sl, strings.Split(string(data), ","))
	return nil
}

func CommaSeperatedToStringList(s string) StringList {
	return strings.Split(s, ",")
}

// CertificateRequest implements the Vault certificate requests
type CertificateRequest struct {
	Name              string     `json:"name,omitempty"`
	CommonName        string     `json:"common_name,omitempty"`
	AltNames          string     `json:"alt_names,omitempty"`
	IpSans            StringList `json:"ip_sans,omitempty"`
	UriSans           StringList `json:"uri_sans,omitempty"`
	OtherSans         StringList `json:"other_sans,omitempty"`
	TTL               string     `json:"ttl,omitempty"`
	Format            string     `json:"format,omitempty"`
	PrivateKeyFormat  string     `json:"private_key_format,omitempty"`
	ExcludeCnFromSans bool       `json:"exclude_cn_from_sans,omitempty"`
}

// CertificateData is a subtype used in CertificateResponse
type CertificateData struct {
	Certificate    string   `json:"certificate,omitempty"`
	Expiration     UnixTime `json:"expiration,omitempty"`
	IssuingCa      string   `json:"issuing_ca,omitempty"`
	PrivateKey     string   `json:"private_key,omitempty"`
	PrivateKeyType string   `json:"private_key_type,omitempty"`
	SerialNumber   string   `json:"serial_number,omitempty"`
}

// CertificateResponse implementes the Vault response for a certificate request
type CertificateResponse struct {
	RequestId     string          `json:"request_id,omitempty"`
	LeaseId       string          `json:"lease_id,omitempty"`
	LeaseDuration UnixTime        `json:"lease_duration,omitempty"`
	Renewable     bool            `json:"renewable,omitempty"`
	Data          CertificateData `json:"data,omitempty"`
}

// marshal serializes an arbitrary object into json and returns a io.Reader for the result.
// Suitable for http request body definition. Note that this will buffer the body
// in memory
func marshal(v interface{}) (io.Reader, error) {
	buffer := bytes.Buffer{}
	encoder := json.NewEncoder(&buffer)
	if err := encoder.Encode(v); err != nil {
		return nil, fmt.Errorf("%w: failed to marshal %v: %v", ErrRetrieval, v, err)
	}

	return &buffer, nil
}

// loginViaServiceAccount authenticates to Vault using the kubernetes serviceaccount
// engine. The code has taken directly from this example:
// https://www.vaultproject.io/docs/auth/kubernetes#code-example and adapted slightly
func (cr *CertRetrieval) loginViaServiceAccount() (string, error) {
	klog.Info("Authorizing via service account")
	config := vault.DefaultConfig()
	config.Address = cr.Address
	if cr.ServerCA != "" {
		err := config.ConfigureTLS(&vault.TLSConfig{
			CACert: cr.ServerCA,
		})
		if err != nil {
			return "", fmt.Errorf("%w: failed to configure TLS: %v", ErrRetrieval, err)
		}
	}

	client, err := vault.NewClient(config)
	if err != nil {
		return "", fmt.Errorf("%w: unable to initialize Vault client: %v", ErrRetrieval, err)
	}

	// The service-account token will be read from the path where the token's
	// Kubernetes Secret is mounted. By default, Kubernetes will mount it to
	// /var/run/secrets/kubernetes.io/serviceaccount/token, but an administrator
	// may have configured it to be mounted elsewhere.
	// In that case, we'll use the option WithServiceAccountTokenPath to look
	// for the token there.
	k8sAuth, err := auth.NewKubernetesAuth(
		cr.AuthRole,
		auth.WithServiceAccountTokenPath(ServiceAccountPath),
	)
	if err != nil {
		return "", fmt.Errorf("%w: unable to initialize Kubernetes auth method: %v", ErrRetrieval, err)
	}

	authInfo, err := client.Auth().Login(context.TODO(), k8sAuth)
	if err != nil {
		return "", fmt.Errorf("%w: unable to log in with Kubernetes auth: %v", ErrRetrieval, err)
	}
	if authInfo == nil {
		return "", fmt.Errorf("%w: no auth info was returned after login", ErrRetrieval)
	}
	token := authInfo.Auth.ClientToken
	klog.Infof("Resulting token: %v", token)
	return token, nil
}

// checkIfServiceAccountToken parses the given token and returns true if it is a service account token
func checkIfServiceAccountToken(tokenString string) bool {
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return false
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok {
		if _, ok := claims["kubernetes.io"]; ok {
			kubernetes := claims["kubernetes.io"].(map[string]interface{})
			if _, ok := kubernetes["namespace"]; !ok {
				return false
			}
			if _, ok := kubernetes["pod"]; !ok {
				return false
			}
			if _, ok := kubernetes["serviceaccount"]; !ok {
				return false
			}
		}
	}
	return true
}

func (cr *CertRetrieval) returnVaultTokenByServiceAccount(tokenfile string) (string, error) {
	data, err := os.ReadFile(tokenfile)
	if err != nil {
		return "", err
	}
	tokenRead := strings.TrimSpace(string(data))

	// Check if the token is a service account token and if so, use it to login to vault and return the vault token.
	// Otherwise, return the token as is because it is already a vault token.
	if checkIfServiceAccountToken(tokenRead) {
		token, err := cr.loginViaServiceAccount()
		if err != nil {
			return "", fmt.Errorf("failed to retrieve token via service account: %v", err)
		}
		return token, nil
	} else {
		return tokenRead, nil
	}
}

// readToken evaluates different sources for the token.
// The order of precedence is:
// 1. Setting "token"
// 2. Setting "tokenFile"
// 3. Setting "reading token from service account and ask vault for a token"
func (cr *CertRetrieval) readToken() (string, error) {
	if cr.Token != "" {
		klog.Infof("Using token from env variable")
		return cr.Token, nil
	}
	if cr.Tokenfile != "" {
		return cr.returnVaultTokenByServiceAccount(cr.Tokenfile)
	} else {
		return cr.returnVaultTokenByServiceAccount(ServiceAccountPath)
	}
}

// retrieveCert executes the http request to retrieve a new certificate from vault
func (cr *CertRetrieval) retrieveCert() (*CertificateResponse, error) {
	token, err := cr.readToken()
	if err != nil {
		return nil, err
	}

	raw := cr.Address + "/v1/" + cr.PKI + "/issue/" + cr.Role
	address, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid url %q: %v", ErrRetrieval, raw, err)
	}
	klog.Infof("URL: %v", address)
	transport := http.Transport{}
	if address.Scheme == "https" {
		caPool := x509.NewCertPool()
		if cr.ServerCA != "" {
			crPem, err := os.ReadFile(cr.ServerCA)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to read CA certificate from %q: %v", ErrRetrieval, cr.ServerCA, err)
			}
			block, _ := pem.Decode([]byte(crPem))
			caCert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to parse CA certificate: %v", ErrRetrieval, err)
			}
			caPool.AddCert(caCert)
		}

		transport.TLSClientConfig = &tls.Config{
			Rand:    rand.Reader,
			RootCAs: caPool,
		}
	}
	client := http.Client{Transport: &transport}

	certRequest := CertificateRequest{CommonName: cr.Name}

	if cr.AltNames != "" {
		certRequest.AltNames = cr.AltNames
	}

	if cr.TTL > 0 {
		certRequest.TTL = cr.TTL.String()
		klog.Infof("Request certificate with TTL %v", certRequest.TTL)
	}

	requestBody, err := marshal(certRequest)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, address.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrRetrieval, err)
	}
	request.Method = http.MethodPut
	request.Header.Add("content-type", "application/json")
	request.Header.Add("accept", "application/json")
	request.Header.Add("X-Vault-Token", token)
	request.Header.Add("X-Vault-Request", "true")

	if os.Getenv("VAULT_CURL_DEBUG") != "" {
		fmt.Println(http2curl.GetCurlCommand(request))
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: request failed: %v", ErrRetrieval, err)
	}
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: failed to retrieve: %v", ErrRetrieval, response.Status)
	}

	defer response.Body.Close()
	decoder := json.NewDecoder(response.Body)
	certificate := CertificateResponse{}
	if err := decoder.Decode(&certificate); err != nil {
		return nil, fmt.Errorf("%w: failed to decode body: %v", ErrRetrieval, err)
	}

	return &certificate, nil
}

// storeFile writes the passed data to a _temporary_ file in the same directory
// as the target file. The targetfile is _not_ modified
func (cr *CertRetrieval) storeFile(data []byte, targetFile string) (string, error) {
	dir := filepath.Dir(targetFile)
	name := filepath.Base(targetFile)
	file, err := os.CreateTemp(dir, "."+name)
	if err != nil {
		return "", fmt.Errorf("%w: failed to create tempfile: %v", ErrRetrieval, err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	defer writer.Flush()

	if _, err := io.Copy(writer, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("%w: failed to write data to %q: %v", ErrRetrieval, file.Name(), err)
	}

	return file.Name(), nil
}

// storeCertificate stores the certificate data into the target files
func (cr *CertRetrieval) storeCertificate(certificate *CertificateResponse) error {
	var certFile, keyFile, caFile string
	var err error
	certFile, err = cr.storeFile([]byte(certificate.Data.Certificate), cr.OutCertfile)
	if err != nil {
		return err
	}

	keyFile, err = cr.storeFile([]byte(certificate.Data.PrivateKey), cr.OutKeyfile)
	if err != nil {
		return err
	}

	if cr.OutCAfile != "" {
		caFile, err = cr.storeFile([]byte(certificate.Data.IssuingCa), cr.OutCAfile)
		if err != nil {
			return err
		}
	}

	if err := os.Rename(certFile, cr.OutCertfile); err != nil {
		return fmt.Errorf("%w: failed to rename certfile: %v", ErrRetrieval, err)
	}
	klog.Infof("Wrote certificate to %s", cr.OutCertfile)

	if err := os.Rename(keyFile, cr.OutKeyfile); err != nil {
		return fmt.Errorf("%w: failed to rename keyfile: %v", ErrRetrieval, err)
	}
	klog.Infof("Wrote keyfile to %s", cr.OutKeyfile)

	if cr.OutCAfile != "" {
		if err := os.Rename(caFile, cr.OutCAfile); err != nil {
			return fmt.Errorf("%w: failed to rename cafile: %v", ErrRetrieval, err)
		}
		klog.Infof("Wrote signing certificate to %s", cr.OutCAfile)
	}

	return nil
}

// oldCertIsStale determines, if the validity period of the current certificate
// is nearing end of life (or is already expired). The tolerance is used
// to retrieve a certificate early.
func (cr *CertRetrieval) oldCertIsStale() bool {
	if cr.Force {
		return true
	}
	_, err := os.Stat(cr.OutCertfile)
	if os.IsNotExist(err) {
		// Certfile does not exist => retrieve it anyways
		return true
	}
	pemData, err := os.ReadFile(cr.OutCertfile)
	if err != nil {
		klog.Errorf("Error while reading old certificate %q: %v", cr.OutCertfile, err)
		return true
	}

	certData, _ := pem.Decode(pemData)
	if certData == nil {
		klog.Errorf("No PEM data found in %q", cr.OutCertfile)
		return true
	}

	cert, err := x509.ParseCertificate(certData.Bytes)
	if err != nil {
		klog.Errorf("Certificate is not parseable: %v", err)
		return true
	}
	remainingValidity := time.Until(cert.NotAfter)
	if remainingValidity < 0 {
		// expired in the past => is stale
		return true
	}

	if cr.ValidityCheckTolerance == 0 {
		// no tolerance check defined, so this certificate is still valid
		return false
	}

	// calculate the percentage of the total lifetime of the cert
	lifetime := cert.NotAfter.Sub(cert.NotBefore) * time.Duration(cr.ValidityCheckTolerance) / 100
	// convert the lifetime into an absolute point in time
	limit := time.Now().Add(lifetime)

	// return true  we are not in the acceptable range of the validity period anymore
	return limit.Before(time.Now())
}

// Retrieve performs the certificate retrieval
func (cr *CertRetrieval) Retrieve() error {
	if !cr.oldCertIsStale() {
		klog.Infof("Old certificate in %q is still valid, not retrieving new one", cr.OutCertfile)
		return nil
	}

	klog.Infof("Old certificate in %q is stale or does not exist, retrieving new one", cr.OutCertfile)
	certificate, err := cr.retrieveCert()
	if err != nil {
		return err
	}
	klog.Info("Retrieved certificates successfully, storing to file")
	if err := cr.storeCertificate(certificate); err != nil {
		klog.Errorf("Failed to store certificates: %v", err)
	}
	return nil
}
