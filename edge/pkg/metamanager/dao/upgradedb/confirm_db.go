package upgradedb

import (
	"encoding/json"
	"errors"
	"fmt"

	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	v2 "github.com/kubeedge/kubeedge/edge/pkg/metamanager/dao/v2"
)

const (
	NodeTaskRequestName       = "NodeTaskRequest"
	NodeUpgradeJobRequestName = "NodeUpgradeJobRequest"
)

func checkIfNodeUpgradeJobRequestExists() bool {
	return dbm.DBAccess.QueryTable(v2.NewMetaTableName).Filter("name", NodeUpgradeJobRequestName).Exist()
}

// SaveNodeUpgradeJobRequestToMetaV2 save NodeUpgradeJobRequest to meta_v2
func SaveNodeUpgradeJobRequestToMetaV2(nodeUpgradeJobReq commontypes.NodeUpgradeJobRequest) error {
	nodeUpgradeJobReqJSON, err := json.Marshal(nodeUpgradeJobReq)
	if err != nil {
		return errors.New("failed to marshal NodeUpgradeJobRequest")
	}
	meta := v2.MetaV2{
		Key:   nodeUpgradeJobReq.UpgradeID,
		Name:  NodeUpgradeJobRequestName,
		Value: string(nodeUpgradeJobReqJSON),
	}
	if checkIfNodeUpgradeJobRequestExists() {
		klog.Info("NodeUpgradeJobRequest is existing,begin updating...")
		num, err := dbm.DBAccess.Update(meta, "")
		klog.V(4).Infof("Update affected Num: %d, %v", num, err)
		if err != nil {
			return err
		}
	} else {
		klog.Info("NodeUpgradeJobRequest is not existing,begin insert...")
		num, err := dbm.DBAccess.Insert(meta)
		klog.V(4).Infof("Insert affected Num: %d, %v", num, err)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteNodeUpgradeJobRequestFromMetaV2 delete NodeUpgradeJobRequest from meta_v2
func DeleteNodeUpgradeJobRequestFromMetaV2() error {
	num, err := dbm.DBAccess.QueryTable(v2.NewMetaTableName).Filter("name", NodeUpgradeJobRequestName).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return nil
}

// QueryNodeUpgradeJobRequestFromMetaV2 query NodeUpgradeJobRequest from meta_v2
func QueryNodeUpgradeJobRequestFromMetaV2() (commontypes.NodeUpgradeJobRequest, error) {
	var nodeUpgradeReq commontypes.NodeUpgradeJobRequest
	meta := v2.MetaV2{}
	err := dbm.DBAccess.QueryTable(v2.NewMetaTableName).Filter("name", NodeUpgradeJobRequestName).One(meta)
	if err != nil {
		return nodeUpgradeReq, err
	}
	err = json.Unmarshal([]byte(meta.Value), &nodeUpgradeReq)
	if err != nil {
		return nodeUpgradeReq, fmt.Errorf("failed to unmarshal NodeUpgradeJobRequest: %v", err)
	}
	return nodeUpgradeReq, nil
}
func checkIfNodeTaskRequestExists() bool {
	return dbm.DBAccess.QueryTable(v2.NewMetaTableName).Filter("name", NodeTaskRequestName).Exist()
}

// SaveNodeTaskRequestToMetaV2 save NodeTaskRequest to meta_v2
func SaveNodeTaskRequestToMetaV2(nodeTaskReq types.NodeTaskRequest) error {
	nodeTaskReqJSON, err := json.Marshal(nodeTaskReq)
	if err != nil {
		return errors.New("failed to marshal NodeTaskRequest")
	}
	meta := v2.MetaV2{
		Key:   nodeTaskReq.TaskID,
		Name:  NodeTaskRequestName,
		Value: string(nodeTaskReqJSON),
	}
	if checkIfNodeTaskRequestExists() {
		klog.Info("NodeTaskRequest is existing,begin updating...")
		num, err := dbm.DBAccess.Update(meta, "")
		klog.V(4).Infof("Update affected Num: %d, %v", num, err)
		if err != nil {
			return err
		}
	} else {
		klog.Info("NodeTaskRequest is not existing,begin insert...")
		num, err := dbm.DBAccess.Insert(meta)
		klog.V(4).Infof("Insert affected Num: %d, %v", num, err)
		if err != nil {
			return err
		}
	}
	return nil
}

// DeleteNodeTaskRequestFromMetaV2 delete NodeTaskRequest from meta_v2
func DeleteNodeTaskRequestFromMetaV2() error {
	num, err := dbm.DBAccess.QueryTable(v2.NewMetaTableName).Filter("name", NodeTaskRequestName).Delete()
	klog.V(4).Infof("Delete affected Num: %d, %v", num, err)
	return nil
}

// QueryNodeTaskRequestFromMetaV2 query NodeTaskRequest from meta_v2
func QueryNodeTaskRequestFromMetaV2() (types.NodeTaskRequest, error) {
	var nodeTaskReq types.NodeTaskRequest
	meta := v2.MetaV2{}
	err := dbm.DBAccess.QueryTable(v2.NewMetaTableName).Filter("name", NodeTaskRequestName).One(meta)
	if err != nil {
		return nodeTaskReq, err
	}
	err = json.Unmarshal([]byte(meta.Value), &nodeTaskReq)
	if err != nil {
		return nodeTaskReq, fmt.Errorf("failed to unmarshal NodeUpgradeJobRequest: %v", err)
	}
	return nodeTaskReq, nil
}
