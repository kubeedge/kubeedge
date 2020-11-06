package relation

import (
	"github.com/kubeedge/kubeedge/edge/pkg/common/dbm"
	"k8s.io/klog"
)

const RelationTableName = "relation"

type IRelation interface {
	GetList() string
	GetKind() string
	GetGroupResource() string
}

type Relation struct {
	GroupResource string `orm:"column(groupresource); size(256); pk"`
	Kind          string `orm:"column(kind); size(256)"`
	List          string `orm:"column(list); size(256)"`
}

func (r *Relation) GetList() string {
	if r == nil {
		return ""
	}

	return r.List
}

func (r *Relation) GetKind() string {
	if r == nil {
		return ""
	}

	return r.Kind
}

func (r *Relation) GetGroupResource() string {
	if r == nil {
		return ""
	}

	return r.GroupResource
}

func insertOrUpdate(relation *Relation) error {
	_, err := dbm.DBAccess.Raw(
		"INSERT OR REPLACE INTO relation (groupresource, kind, list) VALUES (?,?,?)",
		relation.GetGroupResource(),
		relation.GetKind(),
		relation.GetList(),
	).Exec()

	klog.V(4).Infof("update relation. result: %+v\n", err)
	return err
}

func queryAll() ([]*Relation, error) {
	relations := make([]*Relation, 0)
	_, err := dbm.DBAccess.QueryTable(RelationTableName).All(&relations)
	return relations, err
}

func query(gr string) *Relation {
	r := &Relation{GroupResource: gr}
	if err := dbm.DBAccess.Read(r); err != nil {
		klog.Errorf("query relation failed with error: %+v", err)
		return nil
	}

	return r
}
