package cache

import (
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/klog"
)

var cacheDataPathPrefix = "/var/lib/kubeedge/cache"

func InitCacheDataPathPrefix(path string) {
	cacheDataPathPrefix = path
}

type Cache struct {
	ID        int64
	UA        string
	Resource  string
	Namespace string
	Name      string
	Value     []byte
}

func buildDirPath(ua, gr, namespace string) string {
	return fmt.Sprintf("%s/%s/%s/%s/", cacheDataPathPrefix, ua, gr, namespace)
}

func writeFile(path, filename string, content []byte) error {
	var err error

	if err = os.MkdirAll(path, 0755); err != nil {
		klog.Errorf("mkdir %s failed with error %+v", path, err)
		return err
	}

	filename = path + filename

	if err = ioutil.WriteFile(filename, content, 0644); err != nil {
		klog.Errorf("writle file %s failed with error %+v", filename, err)
		return err
	}

	return nil
}

func readFileRecursion(dir string) ([]string, error) {
	var (
		err     error
		content = make([]string, 0)
	)

	filelist, err := ioutil.ReadDir(dir)
	if err != nil {
		// no resource there
		if os.IsNotExist(err) {
			return content, nil
		}

		klog.Errorf("read cache data from %s failed with error %+v", dir, err)
		return content, err
	}

	for i := 0; i < len(filelist); i++ {
		file := filelist[i]

		path := dir + "/" + file.Name()
		if file.IsDir() {
			subcontent, _ := readFileRecursion(path)
			content = append(content, subcontent...)
			continue
		}

		filecontent, err := readFile(path)
		if err != nil {
			klog.Errorf("read file %s failed with error %+v", path, err)
			continue
		}

		content = append(content, filecontent)
	}

	return content, nil
}

func readFile(path string) (string, error) {
	bs, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}

	return string(bs), err
}

func deleteCache(ua, resource, namespace, name string) {
	path := buildDirPath(ua, resource, namespace)
	if err := os.RemoveAll(path); err != nil {
		klog.Errorf("delete cache data failed with error %+v", err)
		return
	}
	klog.Infof("delete cache data from %s successful", path)
	return
}

func insertOrUpdateCache(cache *Cache) error {
	path := buildDirPath(cache.UA, cache.Resource, cache.Namespace)
	if err := writeFile(path, cache.Name, cache.Value); err != nil {
		klog.Errorf(
			"insertOrUpdate %s/%s/s/%s failed.",
			cache.UA,
			cache.Resource,
			cache.Namespace,
			cache.Name,
		)
		return err
	}

	return nil
}

func queryCacheList(ua, resource, namespace string) ([]string, error) {
	return readFileRecursion(buildDirPath(ua, resource, namespace))
}

func queryCache(ua, resource, namespace, name string) (string, error) {
	return readFile(buildDirPath(ua, resource, namespace) + name)
}

func IsUAExisted(ua string) bool {
	path := buildDirPath(ua, "", "")
	info, err := os.Stat(path)
	if err == nil {
		return info.IsDir()
	}

	return false
}
