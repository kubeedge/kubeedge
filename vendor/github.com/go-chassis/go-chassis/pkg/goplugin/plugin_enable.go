// +build !go1.10 !debug

package goplugin

import "plugin"

// LoadPlugin load plugin
func LoadPlugin(name string) (*plugin.Plugin, error) {
	path, err := LookupPlugin(name)
	if err != nil {
		return nil, err
	}
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func lookUp(plugName, symName string) (interface{}, error) {
	p, err := LoadPlugin(plugName)
	if err != nil {
		return nil, err
	}
	return p.Lookup(symName)
}
