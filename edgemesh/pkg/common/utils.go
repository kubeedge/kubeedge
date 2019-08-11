package common

import (
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

//keep service name format the same as k8s: ${service_name}.${namespace}.svc.${cluster}:${port}
func ParseServiceName(serviceName string) (service, namespace, serviceType, cluster string, port int, err error) {
	serviceNameSets := strings.Split(serviceName, ".")
	if len(serviceNameSets) != 4 {
		err = errors.New("invalid length after splitting service name")
		return
	}
	splitToGetPort := strings.Split(serviceNameSets[3], ":")
	if len(splitToGetPort) != 2 {
		err = errors.New("invalid length when splitting to get port")
		return
	}
	service = serviceNameSets[0]
	namespace = serviceNameSets[1]
	serviceType = "svc"
	cluster = splitToGetPort[0]
	port, err = strconv.Atoi(splitToGetPort[1])
	return
}
