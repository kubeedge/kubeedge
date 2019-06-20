package security

import (
	"errors"
	"fmt"
	"os"

	"github.com/go-chassis/foundation/security"
	"github.com/go-chassis/go-chassis/pkg/goplugin"
	"github.com/go-mesh/openlogging"
)

const pluginSuffix = ".so"

//CipherPlugins is a map
var cipherPlugins map[string]func() security.Cipher

//InstallCipherPlugin is a function
func InstallCipherPlugin(name string, f func() security.Cipher) {
	cipherPlugins[name] = f
}

//GetCipherNewFunc is a function
func GetCipherNewFunc(name string) (func() security.Cipher, error) {
	if f, ok := cipherPlugins[name]; ok {
		return f, nil
	}
	openlogging.GetLogger().Debugf("try to load cipher [%s] from go plugin", name)
	f, err := loadCipherFromPlugin(name)
	if err == nil {
		cipherPlugins[name] = f
		return f, nil
	}
	if !os.IsNotExist(err) {
		return nil, err
	}
	return nil, fmt.Errorf("unkown cipher plugin [%s]", name)
}

func loadCipherFromPlugin(name string) (func() security.Cipher, error) {
	c, err := goplugin.LookUpSymbolFromPlugin(name+pluginSuffix, "Cipher")
	if err != nil {
		return nil, err
	}
	customCipher, ok := c.(security.Cipher)
	if !ok {
		return nil, errors.New("symbol from plugin is not type Cipher")
	}
	f := func() security.Cipher {
		return customCipher
	}
	return f, nil
}

func init() {
	cipherPlugins = make(map[string]func() security.Cipher)
}
