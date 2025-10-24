package dao

import (
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"k8s.io/klog/v2"
)

const (
	EndpointUrlsName = "endpoint_urls"
)

type EndpointUrls struct {
	Endpoint string `orm:"column(endpoint);type(text);pk"`
	URL      string `orm:"column(url);type(text)"`
}

func InsertEpUrls(endpoint, url string) error {
	_, err := dbm.DBAccess.Raw(
		"INSERT INTO endpoint_urls (endpoint, url) VALUES (?, ?)", endpoint, url).Exec()
	klog.V(4).Infof("INSERT result %v", err)
	return err
}

func DeleteEpUrlsByKey(endpoint string) error {
	num, err := dbm.DBAccess.QueryTable(EndpointUrlsName).Filter("endpoint", endpoint).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return err
}

func IsTableEmpty() bool {
	var count int64
	if count, _ = dbm.DBAccess.QueryTable(EndpointUrlsName).Count(); count > 0 {
		return false
	}
	return true
}

func GetEpUrlsByKey(endpoint string) (*EndpointUrls, error) {
	targetUrls := new(EndpointUrls)
	if err := dbm.DBAccess.QueryTable(EndpointUrlsName).Filter("endpoint", endpoint).One(targetUrls); err != nil {
		return nil, err
	}
	return targetUrls, nil
}

func GetAllEpUrls() ([]*EndpointUrls, error) {
	var EpUrls []*EndpointUrls
	_, err := dbm.DBAccess.QueryTable(EndpointUrlsName).All(&EpUrls)
	if err != nil {
		return nil, err
	}
	return EpUrls, nil
}
