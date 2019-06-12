package filesource

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvert2JavaProps(t *testing.T) {
	b := []byte(`
a: 1
b: 2
c:
 d: 3
`)
	m, err := Convert2JavaProps("test.yaml", b)
	assert.NoError(t, err)
	assert.Equal(t, m["c.d"], 3)
}

func TestConvert2ConfigMap(t *testing.T) {
	b := []byte(`
a: 1
b: 2
c:
 d: 3
`)
	m, err := UseFileNameAsKeyContentAsValue("/root/test.yaml", b)
	assert.NoError(t, err)
	assert.Equal(t, b, m["test.yaml"])
}
