package kubewatcher

import (
	"strings"

	"github.com/sunreaver/kubewatcher/constant"
	cpkg "github.com/sunreaver/kubewatcher/controller"
	"github.com/sunreaver/kubewatcher/resource"
	"github.com/sunreaver/kubewatcher/sender"
	"github.com/sunreaver/kubewatcher/util"
)

func handler(queueKey string, value resource.ResourceInter, controller cpkg.K8sController, sdGetter sender.SenderGetter) error {
	resourceCacheMap := controller.GetCacheMap()                 // queueKey为操作/资源key格式 需要拆分开来解析
	eventType, resourceKey := cpkg.SplitWatcherKeyFunc(queueKey) // level为每种资源自定义的一个在层级结构中的层级 通过level可以灵活设置哪些资源的状态和reason通过哪些途径修改
	resourceCacheItem := resourceCacheMap.GetResourceCacheBYKey(resourceKey)
	// 开启状态机更新自身状态以及向上推理更新上层状态
	if eventType.IsUpdate() {
		if resourceCacheItem.IsNil() || resourceCacheItem.IsSingle() {
			// 缓存值为空 或者 不为空但是是孤儿节点 先尝试建立关联
			// 该步骤之后 resourceCacheItem 必不为空
			item, err := value.AddRel(controller.GetCacheMap(), controller.GetIndexer())
			if err != nil {
				return err
			}
			resourceCacheItem = item
		}
		nowStatus, reason := value.GetStatus()
		checkStatus(resourceCacheItem, sdGetter.GetSender(), controller.GetCacheMap(), nowStatus, reason, value.GetMeta())
	} else {
		if resourceCacheItem.IsNil() {
			// 上来第一个状态就是delete的资源不作处理
			return nil
		}
		checkStatus(resourceCacheItem, sdGetter.GetSender(), controller.GetCacheMap(), constant.K8sResStatusDelete, "delete", value.GetMeta())
	}
	return nil
}

/*
level代表当前递归层级 根据当前层级判断下一层级的changeStatus, changeReason可以达到控制哪些父级资源可以被修改哪些字段的功能
nowStatus为delete时 还要根据changeStatus判断 例如:经过推理 某个dep应该被删除 但是实际策略上dep的状态不靠推理来管 所以即使nowStatus为delete 最终也不会删除
firstIn为true 代表该次修改状态为自身触发 非推理触发
*/
func checkStatus(resource *resource.ResourceCache, sender *sender.Sender, keyCatch *resource.ResourceKeyCache, nowStatus constant.K8sResStatus, reason string, meta interface{}) {
	// 处理自身
	dealSelf(resource, sender, keyCatch, nowStatus, reason, meta)
	// 向上处理
	dealUp(resource, sender, keyCatch)
}

func dealSelf(resource *resource.ResourceCache, sender *sender.Sender, keyCatch *resource.ResourceKeyCache, nowStatus constant.K8sResStatus, reason string, meta interface{}) {
	oldStatus := resource.GetStatus()
	oldFailReason := resource.GetReason()
	needSend := false
	if meta != nil {
		resource.SetMeta(meta)
	}
	if len(reason) > 0 && reason != oldFailReason {
		util.Debugw("k8s_watcher_reason_change", "kind", resource.GetKind(), "key", resource.GetKey(), "reason", oldFailReason, "newReason", reason)
		// needSend = true
		resource.SetReason(reason)
	}
	// 当前状态与旧状态不一致 或者 状态一致但错误原因变动
	if nowStatus != oldStatus {
		util.Infow("k8s_watcher_status_change", "kind", resource.GetKind(), "key", resource.GetKey(), "status", oldStatus, "newStatus", nowStatus)
		needSend = true
		resource.SetStatus(nowStatus)
	}
	if nowStatus.IsDelete() {
		util.Infow("k8s_watcher_status_delete", "kind", resource.GetKind(), "key", resource.GetKey())
		needSend = true
		// 要执行删除动作 将当前资源从父亲的儿子列表中移除
		resource.GetParent().RemoveChild(resource.GetKey())
		// 删除当前节点
		keyCatch.DeleteCacheByKey(resource.GetKey())
	}
	if needSend {
		// 向外推送
		sender.AddSendOut(resource.GetSendOut())
	}
}

func dealUp(r *resource.ResourceCache, sender *sender.Sender, keyCatch *resource.ResourceKeyCache) {
	parent := r.GetParent()
	if parent != nil && parent.GetStatus() != constant.K8sResStatusDelete { // 如果parent被删除，则不能通过此方法更新
		fullReasonList := make([]string, 0)
		fullStatus := constant.K8sResStatusSucceed
		shouldDelete := true // 父节点没有任何子节点后 理应删除 但是是否真的删除 要视策略而定
		parent.RangeWithoutDelete(func(brother *resource.ResourceCache) (stop bool) {
			util.Debugw("child", "key", brother.GetKey(), "status", brother.GetStatus())
			shouldDelete = false // 有至少一个非空子节点就不删除
			if brother.GetStatus() == constant.K8sResStatusFail {
				fullStatus = constant.K8sResStatusFail
				fullReasonList = append(fullReasonList, brother.GetReason())
			}
			return false
		})
		fullReason := strings.Join(fullReasonList, "\n")
		if shouldDelete {
			checkStatus(parent, sender, keyCatch, constant.K8sResStatusDelete, fullReason, nil)
		} else {
			checkStatus(parent, sender, keyCatch, fullStatus, fullReason, nil)
		}
	}
}
