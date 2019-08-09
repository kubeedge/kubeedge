package qpslimiter

import (
	"github.com/go-chassis/go-chassis/core/common"
	"strings"
)

// ConsumerKeys contain consumer keys
type ConsumerKeys struct {
	MicroServiceName       string
	SchemaQualifiedName    string
	OperationQualifiedName string
}

// ProviderKeys contain provider keys
type ProviderKeys struct {
	Global          string
	ServiceOriented string
}

//Prefix is const
const Prefix = "cse.flowcontrol"

// GetConsumerKey get specific key for consumer
func GetConsumerKey(sourceName, serviceName, schemaID, OperationID string) *ConsumerKeys {
	keys := new(ConsumerKeys)
	//for mesher to govern
	if sourceName != "" {
		keys.MicroServiceName = strings.Join([]string{Prefix, sourceName, common.Consumer, "qps.limit", serviceName}, ".")
	} else {
		if serviceName != "" {
			keys.MicroServiceName = strings.Join([]string{Prefix, common.Consumer, "qps.limit", serviceName}, ".")
		}
	}
	if schemaID != "" {
		keys.SchemaQualifiedName = strings.Join([]string{keys.MicroServiceName, schemaID}, ".")
	}
	if OperationID != "" {
		keys.OperationQualifiedName = strings.Join([]string{keys.SchemaQualifiedName, OperationID}, ".")
	}
	return keys
}

// GetProviderKey get specific key for provider
func GetProviderKey(sourceServiceName string) *ProviderKeys {
	keys := &ProviderKeys{}
	if sourceServiceName != "" {
		keys.ServiceOriented = strings.Join([]string{Prefix, common.Provider, "qps.limit", sourceServiceName}, ".")
	}

	keys.Global = strings.Join([]string{Prefix, common.Provider, "qps.global.limit"}, ".")
	return keys
}

// GetSchemaQualifiedName get schema qualified name
func (op *ConsumerKeys) GetSchemaQualifiedName() string {
	return op.SchemaQualifiedName
}

// GetMicroServiceSchemaOpQualifiedName get micro-service schema operation qualified name
func (op *ConsumerKeys) GetMicroServiceSchemaOpQualifiedName() string {
	return op.OperationQualifiedName
}

// GetMicroServiceName get micro-service name
func (op *ConsumerKeys) GetMicroServiceName() string {
	return op.MicroServiceName
}
