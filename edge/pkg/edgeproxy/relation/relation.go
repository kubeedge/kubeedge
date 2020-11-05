package relation

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/astaxie/beego/orm"
	"github.com/kubeedge/beehive/pkg/core"
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	"github.com/kubeedge/beehive/pkg/core/model"
	"github.com/kubeedge/kubeedge/edge/pkg/edgehub"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/klog"
)

var mgr iRelationManager

func Init() {
	mgr := &relationManager{}
	mgr.init()
}

func InitDBTable(module core.Module) {
	if !module.Enable() {
		klog.Warningf("Module %s is disabled, relation cache for it will not be registered", module.Name())
		return
	}

	orm.RegisterModel(new(Relation))
}

type iRelationManager interface {
	init()
	updateRelation(*Relation) error
	getRelation(string) IRelation
}

type relationManager struct {
	sync.Once
	data sync.Map
}

func (m *relationManager) init() {
	m.Do(func() {
		relations, err := queryAll()
		if err != nil {
			klog.Errorf("init relation manager data failed with error %+v", err)
			return
		}

		for i := 0; i < len(relations); i++ {
			obj := relations[i]
			m.data.Store(obj.GroupResource, obj)
		}

		m.syncRelationFromRemote()
	})
}

func (m *relationManager) updateRelation(relation *Relation) error {
	if err := insertOrUpdate(relation); err != nil {
		return err
	}

	m.data.Store(relation.GroupResource, relation)
	return nil
}

func (m *relationManager) getRelation(gr string) IRelation {
	return m.getRelationFromMemory(gr)
}

func (m *relationManager) getRelationFromMemory(gr string) *Relation {
	obj, ok := m.data.Load(gr)
	if !ok {
		obj := m.getRelationFromDB(gr)
		m.data.Store(gr, obj)
		return obj
	}

	relation, ok := obj.(*Relation)
	if !ok {
		obj := m.getRelationFromDB(gr)
		m.data.Store(gr, obj)
		return obj
	}

	return relation
}

func (m *relationManager) getRelationFromDB(gr string) *Relation {
	r := query(gr)
	if r == nil {
		m.syncRelationFromRemote()
		r = query(gr)
	}

	return r
}

func (m *relationManager) syncRelationFromRemote() {
	resource := fmt.Sprintf("%s/%s/%s", "default", model.ResourceTypeAPIResource, "")
	msg := model.
		NewMessage("").
		BuildRouter("", edgehub.ModuleNameEdgeHub, resource, model.QueryOperation).
		FillBody("edgeproxy sync api resource")

	resp, err := beehiveContext.SendSync(edgehub.ModuleNameEdgeHub, *msg, 1*time.Minute)
	if err != nil {
		klog.Errorf("edgeproxy sync api resource failed with error: %+v. response: %+v", err, resp)
		return
	}

	content := resp.Content.(string)
	resourcelists := make([]*v1.APIResourceList, 0)

	if err = json.Unmarshal([]byte(content), &resourcelists); err != nil {
		klog.Errorf("edgeproxy unmarshal response content failed with error: %+v. content: %s", err, content)
		return
	}

	for _, resourcelist := range resourcelists {
		gv, err := schema.ParseGroupVersion(resourcelist.GroupVersion)
		if err != nil {
			continue
		}

		for _, resource := range resourcelist.APIResources {
			if strings.Contains(resource.Name, "/") {
				continue
			}

			gr := schema.GroupResource{
				Group:    gv.Group,
				Resource: resource.Name,
			}.String()

			obj := &Relation{
				GroupResource: gr,
				Kind:          resource.Kind,
				List:          resource.Kind + "List",
			}

			if err := m.updateRelation(obj); err != nil {
				klog.Errorf("edgeproxy update relation failed with error %+v. object: %+v", err, obj)
				continue
			}
		}
	}
}
