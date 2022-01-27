package certretrieval

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

var (
	ErrConfig    = fmt.Errorf("configuration error")
	ErrRetrieval = fmt.Errorf("retrieval error")
)

// Config is the configuration for the certrieval
type Config struct {
	Tokenfile              string
	Token                  string
	Vault                  string
	ServerCA               string
	Role                   string
	Name                   string
	ValidityCheckTolerance int64
	Force                  bool
	TTL                    time.Duration

	OutCAfile   string
	OutCertfile string
	OutKeyfile  string
}

// Validate the configuration
func (c Config) Validate() error {
	var errors []error
	if c.Tokenfile == "" && c.Token == "" {
		errors = append(errors, fmt.Errorf("tokenfile not defined"))
	}

	if c.Vault == "" {
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

// UnixTime is a wrapper of time.Time with a suitable JSON marshalling
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
type StringList []string

func (sl StringList) MarshalJSON() ([]byte, error) {
	return []byte(strings.Join(sl, ",")), nil
}

func (sl *StringList) UnmarshalJSON(data []byte) error {
	copy(*sl, strings.Split(string(data), ","))
	return nil
}

// CertificateRequest implements the Vault certificate requests
type CertificateRequest struct {
	Name              string     `json:"name,omitempty"`
	CommonName        string     `json:"common_name,omitempty"`
	AltNames          StringList `json:"alt_names,omitempty"`
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
// Suitable for http request body definition
func marshal(v interface{}) (io.Reader, error) {
	buffer := bytes.Buffer{}
	encoder := json.NewEncoder(&buffer)
	if err := encoder.Encode(v); err != nil {
		return nil, fmt.Errorf("%w: failed to marshal %v: %v", ErrRetrieval, v, err)
	}

	return &buffer, nil
}

// readToken reads the Vault token
func (cr *CertRetrieval) readToken() (string, error) {
	if cr.Token != "" {
		log.Printf("Using token from env variable")
		return cr.Token, nil
	}

	data, err := os.ReadFile(cr.Tokenfile)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))

	return token, nil
}

// retrieveCert executes the http request to retrieve a new certificate from vault
func (cr *CertRetrieval) retrieveCert() (*CertificateResponse, error) {
	token, err := cr.readToken()
	if err != nil {
		return nil, err
	}

	raw := cr.Vault + "/v1/pki/issue/" + cr.Role
	address, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid url %q: %v", ErrRetrieval, raw, err)
	}
	log.Printf("URL: %v", address)
	transport := http.Transport{}
	if address.Scheme == "https" {
		caPool := x509.NewCertPool()
		if cr.ServerCA != "" {
			crPem, err := os.ReadFile(cr.ServerCA)
			if err != nil {
				return nil, fmt.Errorf("%w: failed to read CA certificafte from %q: %v", ErrRetrieval, cr.ServerCA, err)
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
	if cr.TTL > 0 {
		certRequest.TTL = cr.TTL.String()
		log.Printf("Request certificate with TTL %v", certRequest.TTL)
	}
	requestBody, err := marshal(certRequest)
	if err != nil {
		return nil, nil
	}

	request, err := http.NewRequest(http.MethodPost, address.String(), requestBody)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create request: %v", ErrRetrieval, err)
	}

	request.Header.Add("content-type", "application/json")
	request.Header.Add("accept", "application/json")
	request.Header.Add("X-Vault-Token", token)
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
	log.Printf("Wrote certificate to %s", cr.OutCertfile)

	if err := os.Rename(keyFile, cr.OutKeyfile); err != nil {
		return fmt.Errorf("%w: failed to rename keyfile: %v", ErrRetrieval, err)
	}
	log.Printf("Wrote keyfile to %s", cr.OutKeyfile)

	if cr.OutCAfile != "" {
		if err := os.Rename(caFile, cr.OutCAfile); err != nil {
			return fmt.Errorf("%w: failed to rename cafile: %v", ErrRetrieval, err)
		}
		log.Printf("Wrote signing certificate to %s", cr.OutCAfile)
	}

	return nil
}

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
		log.Printf("Error while reading old certificate %q: %v", cr.OutCertfile, err)
		return true
	}

	certData, _ := pem.Decode(pemData)
	if certData == nil {
		log.Printf("No PEM data found in %q", cr.OutCertfile)
		return true
	}

	cert, err := x509.ParseCertificate(certData.Bytes)
	if err != nil {
		log.Printf("Certificate is not parseable: %v", err)
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
		log.Printf("Old certificate in %q is still valid, not retrieving new one", cr.OutCertfile)
		return nil
	}

	log.Printf("Old certificate in %q is stale or does not exist, retrieving new one", cr.OutCertfile)
	certificate, err := cr.retrieveCert()
	if err != nil {
		return err
	}
	cr.storeCertificate(certificate)
	return nil
}
