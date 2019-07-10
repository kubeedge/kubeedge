package common

import (
	"github.com/pkg/errors"
	"os"
	"strconv"
	"strings"

	"github.com/go-chassis/go-chassis/core/common"
)

func SplitServiceKey(key string) (name, namespace string) {
	sets := strings.Split(key, ".")
	if len(sets) >= 2 {
		return sets[0], sets[1]
	}

	ns := os.Getenv("POD_NAMESPACE")
	if ns == "" {
		ns = common.DefaultValue
	}
	if len(sets) == 1 {
		return sets[0], ns
	}
	return key, ns
}

func SplitToGetPort(serviceName string) (port int, err error) {
	splitServiceName := strings.Split(serviceName, ":")
	if len(splitServiceName) != 2 {
		err = errors.New("invalid length after splitting")
		return
	}
	port, err = strconv.Atoi(splitServiceName[1])
	return
}