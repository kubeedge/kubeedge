package util

import (
	"errors"
	"strings"
)

var (
	//ErrInvalidPortName happens if your port name is illegal
	ErrInvalidPortName = errors.New("invalid port name, port name must be {protocol}-<suffix>")
	//ErrInvalidURL happens if your utl is illegal
	ErrInvalidURL = errors.New("invalid url, url must be {protocol}://{service-name}:{port-name}")
)

//ParsePortName a port name is composite by protocol-name,like http-admin,http-api,grpc-console,grpc-api
//ParsePortName return two string separately
func ParsePortName(n string) (string, string, error) {
	if n == "" {
		return "", "", ErrInvalidPortName
	}
	tmp := strings.Split(n, "-")
	switch len(tmp) {
	case 2:
		return tmp[0], tmp[1], nil
	case 1:
		return tmp[0], "", nil
	default:
		return "", "", ErrInvalidPortName
	}

}

//ParseServiceAndPort returns service name and port name
func ParseServiceAndPort(n string) (string, string, error) {
	if n == "" {
		return "", "", ErrInvalidURL
	}
	tmp := strings.Split(n, ":")
	switch len(tmp) {
	case 2:
		return tmp[0], tmp[1], nil
	case 1:
		return tmp[0], "", nil
	default:
		return "", "", ErrInvalidURL
	}
}

// GenProtoEndPoint generate proto and port
func GenProtoEndPoint(proto, port string) string {
	if port != "" {
		return proto + "-" + port
	}
	return proto
}
