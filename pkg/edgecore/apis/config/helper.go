package config

import (
	"io/ioutil"

	"k8s.io/klog"
	kyaml "sigs.k8s.io/yaml"
)

func (c *EdgeCoreConfig) Parse(fname string) error {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		klog.Errorf("ReadConfig file %s error %v", fname, err)
		return err
	}
	err = kyaml.Unmarshal(data, c)
	if err != nil {
		klog.Errorf("Unmarshal file %s data error %v", fname, err)
		return err
	}
	return nil
}
