package relaycontroller

import (
	beehiveContext "github.com/kubeedge/beehive/pkg/core/context"
	v1 "github.com/kubeedge/kubeedge/pkg/apis/relays/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/klog/v2"
	"reflect"
)

const (
	CloseOperation = "close"
	OpenOperation  = "open"
)

// 方法的具体实现
func (rc *RelayController) checkRelay() {
	for {
		select {
		case <-beehiveContext.Done():
			klog.Info("Stop checkRelay")
			return
		case e := <-rc.relayrcManager.Events():
			relayrc, ok := e.Object.(*v1.Relayrc)

			if !ok {
				klog.Warningf("Object type: %T unsupported", e.Object)
				continue
			}
			switch e.Type {
			case watch.Added:
				rc.relayrcAdded(relayrc)
			case watch.Deleted:
				rc.relayrcDeleted(relayrc)
			case watch.Modified:
				rc.relayrcUpdated(relayrc)
			default:
				klog.Warningf("Device event type: %s unsupported", e.Type)
			}

		}

	}
}

func (rc *RelayController) relayrcAdded(relayrc *v1.Relayrc) {
	rc.relayrcManager.RelayInfo.Store(relayrc.Name, relayrc)
	klog.Warningf("Relay added", relayrc.Spec.RelayID)
	//if relayrc.Spec.Open {
	//	if relayrc.Spec.RelayID != "" {
	//		// 下发
	//		msg := model.NewMessage("")
	//		resource, err := messagelayer.BuildResource(relayrc.Spec.RelayID, "", "", "")
	//		if err != nil {
	//			klog.Warningf("Built message resource failed with error: %s", err)
	//			return
	//		}
	//		msg.BuildRouter(modules.RelayControllerModuleName, constants.RelayGroupName, resource, model.OpenOperation)
	//		content := relayrc.Spec
	//		msg.Content = content
	//		err = rc.messageLayer.Send(*msg)
	//	}
	//}
}

func (rc *RelayController) relayrcDeleted(relayrc *v1.Relayrc) {
	rc.relayrcManager.RelayInfo.Delete(relayrc.Name)
	klog.Warningf("Relay delete")
	// 下发关闭信息
	//msg := model.NewMessage("")
	//resource, err := messagelayer.BuildResource(relayrc.Spec.RelayID, "", "", "")
	//if err != nil {
	//	klog.Warningf("Built message resource failed with error: %s", err)
	//	return
	//}
	//msg.BuildRouter(modules.RelayControllerModuleName, constants.RelayGroupName, resource, model.DeleteOperation)
	//err = rc.messageLayer.Send(*msg)
	//err = rc.messageLayer.Send(*msg)
}

func (rc *RelayController) relayrcUpdated(relayrc *v1.Relayrc) {
	klog.Warningf("Relay updated", relayrc.Spec.RelayID)
	//value, ok := rc.relayrcManager.RelayInfo.Load(relayrc.Name)
	//rc.relayrcManager.RelayInfo.Store(relayrc.Name, relayrc)
	//if ok {
	//	cacheRelayrc := value.(*v1.Relayrc)
	//	if isRelayRCUpdated(cacheRelayrc, relayrc) {
	//		if isSwitchUpdated(cacheRelayrc.Spec.Open, relayrc.Spec.Open) {
	//			if relayrc.Spec.Open {
	//				if isRelayIDExist(relayrc.Spec.RelayID) {
	//					// 下发信息，关掉再打开的情况，必须保证有一个指定的relayID
	//					msg := model.NewMessage("")
	//					resource, err := messagelayer.BuildResource(relayrc.Spec.RelayID, "", "", "")
	//					if err != nil {
	//						klog.Warningf("Built message resource failed with error: %s", err)
	//						return
	//					}
	//					msg.BuildRouter(modules.RelayControllerModuleName, constants.RelayGroupName, resource, model.UpdateOperation)
	//					content := relayrc.Spec
	//					msg.Content = content
	//					err = rc.messageLayer.Send(*msg)
	//				}
	//			} else {
	//				// 下发关闭，检查发送到的目标节点是否为“”，如果为“”取旧值
	//				msg := model.NewMessage("")
	//				var resource string
	//				var err error
	//				if relayrc.Spec.RelayID == "" {
	//					resource, err = messagelayer.BuildResource(cacheRelayrc.Spec.RelayID, "", "", "")
	//				} else {
	//					resource, err = messagelayer.BuildResource(relayrc.Spec.RelayID, "", "", "")
	//				}
	//
	//				if err != nil {
	//					klog.Warningf("Built message resource failed with error: %s", err)
	//					return
	//				}
	//				msg.BuildRouter(modules.RelayControllerModuleName, constants.RelayGroupName, resource, model.CloseOperation)
	//				content := relayrc.Spec
	//				msg.Content = content
	//				err = rc.messageLayer.Send(*msg)
	//			}
	//		} else if isRelayIDUpdated(cacheRelayrc.Spec.RelayID, relayrc.Spec.RelayID) {
	//			if relayrc.Spec.RelayID != "" {
	//				// 下发信息
	//				// 下发关闭
	//				msg := model.NewMessage("")
	//				resource, err := messagelayer.BuildResource(cacheRelayrc.Spec.RelayID, "", "", "")
	//				if err != nil {
	//					klog.Warningf("Built message resource failed with error: %s", err)
	//					return
	//				}
	//				msg.BuildRouter(modules.RelayControllerModuleName, constants.RelayGroupName, resource, model.UpdateOperation)
	//				content := relayrc.Spec
	//				msg.Content = content
	//				err = rc.messageLayer.Send(*msg)
	//			}
	//		}
	//	}
	//}
}

func isRelayIDExist(id string) bool {
	if id != "" {
		return true
	}
	return false
}

func isSwitchUpdated(old bool, new bool) bool {
	return old != new
}

func isRelayIDUpdated(old string, new string) bool {
	return old != new
}

func isRelayRCUpdated(old *v1.Relayrc, new *v1.Relayrc) bool {
	return !reflect.DeepEqual(old.ObjectMeta, new.ObjectMeta) || !reflect.DeepEqual(old.Spec, new.Spec) || !reflect.DeepEqual(old.Status, new.Status)
}
