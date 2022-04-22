package manager

import (
	"sync"

	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"

	"github.com/kubeedge/kubeedge/cloud/pkg/upgradecontroller/config"
)

// UpgradeManager is a manager watch upgrade change event
type UpgradeManager struct {
	// events from watch kubernetes api server
	events chan watch.Event

	// UpgradeMap, key is Upgrade.Name, value is *v1alpha2.Upgrade{}
	UpgradeMap sync.Map
}

// Events return a channel, can receive all Upgrade event
func (dmm *UpgradeManager) Events() chan watch.Event {
	return dmm.events
}

// NewUpgradeManager create UpgradeManager from config
func NewUpgradeManager(si cache.SharedIndexInformer) (*UpgradeManager, error) {
	events := make(chan watch.Event, config.Config.Buffer.UpgradeEvent)
	rh := NewCommonResourceEventHandler(events)
	si.AddEventHandler(rh)

	return &UpgradeManager{events: events}, nil
}
