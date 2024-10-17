package upgradedb

import (
	"context"
	"fmt"
	"reflect"

	"github.com/beego/beego/v2/client/orm"
	"k8s.io/klog/v2"

	"github.com/kubeedge/kubeedge/common/types"
	commontypes "github.com/kubeedge/kubeedge/common/types"
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
)

// NodeTaskReq Table
const NodeTaskReqTableName = "upgrade_confirm"

// NodeTaskRequestTable struct
type NodeUpgradeConfirmTable struct {
	Key   string `orm:"column(key); size(256);auto;pk"`
	Name  string `orm:"column(name); size(256)"`
	Value string `orm:"column(value); null; type(text)"`
}

func InitNodeUpgradeConfirmTable() orm.Ormer {
	orm.RegisterModel(new(NodeUpgradeConfirmTable))
	obm := orm.NewOrmUsingDB(NodeTaskReqTableName)
	return obm
}

// QueryNodeUpgradeTable query UpgradeConfirm
func queryNodeUpgradeTable(key string, condition string) (NodeUpgradeConfirmTable, error) {
	var table NodeUpgradeConfirmTable
	_, err := dbm.DBAccess.QueryTable(&NodeUpgradeConfirmTable{}).Filter(key, condition).All(table)
	if err != nil {
		return NodeUpgradeConfirmTable{}, err
	}
	return table, nil
}
func GetNodeTaskRequest() (types.NodeTaskRequest, error) {
	var nodeTaskReq types.NodeTaskRequest
	nodeTaskReqType := reflect.TypeOf(nodeTaskReq)
	nodeTaskReqValue := reflect.ValueOf(nodeTaskReq)
	for i := 0; i < nodeTaskReqType.NumField(); i++ {
		fieldType := nodeTaskReqType.Field(i)
		data, err := queryNodeUpgradeTable("Name", fieldType.Name)
		if err != nil {
			return types.NodeTaskRequest{}, err
		}
		nodeTaskReqValue.FieldByName(data.Name).Set(reflect.ValueOf(data.Value))
	}
	return nodeTaskReq, nil
}
func GetNodeUpgradeJobRequest() (commontypes.NodeUpgradeJobRequest, error) {
	var nodeUpgradeReq commontypes.NodeUpgradeJobRequest
	nodeUpgradeReqType := reflect.TypeOf(nodeUpgradeReq)
	nodeUpgradeReqValue := reflect.ValueOf(nodeUpgradeReq)
	for i := 0; i < nodeUpgradeReqType.NumField(); i++ {
		fieldType := nodeUpgradeReqType.Field(i)
		data, err := queryNodeUpgradeTable("Name", fieldType.Name)
		if err != nil {
			return commontypes.NodeUpgradeJobRequest{}, err
		}
		nodeUpgradeReqValue.FieldByName(data.Name).Set(reflect.ValueOf(data.Value))
	}
	return nodeUpgradeReq, nil
}

// SaveDeviceTwin save NodeTaskRequestField
func saveNodeUpgradeConfirm(o orm.Ormer, doc *NodeUpgradeConfirmTable) error {
	err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		// insert data
		// Using txOrm to execute SQL
		_, e := txOrm.Insert(doc)
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})
	if err != nil {
		klog.Errorf("Something wrong when insert NodeTaskRequest data: %v", err)
		return err
	}
	klog.V(4).Info("insert NodeTaskRequest data successfully")
	return nil
}

// SaveNodeTaskRequest save struct of NodeTaskRequest
func SaveNodeTaskRequest(o orm.Ormer, nodetaskreq types.NodeTaskRequest) error {
	var metadata *NodeUpgradeConfirmTable
	nodeTaskReqType := reflect.TypeOf(nodetaskreq)
	nodeTaskReqValue := reflect.ValueOf(nodetaskreq)
	for i := 0; i < nodeTaskReqType.NumField(); i++ {
		fieldType := nodeTaskReqType.Field(i)
		metadata = &NodeUpgradeConfirmTable{
			Name:  fieldType.Name,
			Value: nodeTaskReqValue.FieldByName(fieldType.Name).String(),
		}
		err := saveNodeUpgradeConfirm(o, metadata)
		if err != nil {
			return fmt.Errorf("Save NodeTaskRequest to DB error:%v", err)
		}
	}
	return nil
}

// SaveNodeUpgradeJobRequest save struct of NodeUpgradeJobRequest
func SaveNodeUpgradeJobRequest(o orm.Ormer, nodeUpgradeJobReq commontypes.NodeUpgradeJobRequest) error {
	var metadata *NodeUpgradeConfirmTable
	nodeUpgradeJobReqType := reflect.TypeOf(nodeUpgradeJobReq)
	nodeUpgradeJobReqValue := reflect.ValueOf(nodeUpgradeJobReq)
	for i := 0; i < nodeUpgradeJobReqType.NumField(); i++ {
		fieldType := nodeUpgradeJobReqType.Field(i)
		metadata = &NodeUpgradeConfirmTable{
			Name:  fieldType.Name,
			Value: nodeUpgradeJobReqValue.FieldByName(fieldType.Name).String(),
		}
		err := saveNodeUpgradeConfirm(o, metadata)
		if err != nil {
			return fmt.Errorf("Save NodeUpgradeJobRequest to DB error:%v", err)
		}
	}
	return nil
}

// DeleteNodeUpgradeConfirmTable delete NodeUpgradeConfirmTable
func DeleteNodeUpgradeConfirmTable(o orm.Ormer) error {
	err := o.DoTx(func(ctx context.Context, txOrm orm.TxOrmer) error {
		// Delete data
		// Using txOrm to execute SQL
		_, e := txOrm.QueryTable(&NodeUpgradeConfirmTable{}).Delete()
		// if e != nil the transaction will be rollback
		// or it will be committed
		return e
	})

	if err != nil {
		klog.Errorf("Something wrong when deleting NodeTaskRequest data: %v", err)
		return err
	}
	klog.V(4).Info("Delete NodeTaskRequest data successfully")
	return nil
}
