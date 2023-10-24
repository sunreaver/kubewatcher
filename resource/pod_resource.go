package resource

import (
	"kubewatcher/constant"
	"kubewatcher/util"
	"strings"

	"github.com/pkg/errors"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/cache"
)

type MyPod struct {
	*v1.Pod
}

// status, reason
func (m *MyPod) GetStatus() (constant.K8sResStatus, string) {
	podStatus := m.Status.Phase
	switch podStatus {
	case v1.PodPending, v1.PodRunning:
		haveWrong := false
		reasonList := make([]string, 0)
		for _, container := range m.Status.ContainerStatuses {
			if container.State.Waiting != nil || container.State.Terminated != nil {
				haveWrong = true
				if container.State.Waiting != nil {
					if container.State.Waiting.Message == "" && container.State.Waiting.Reason == "" {
						reasonList = append(reasonList, util.ConcatReason("Waiting", "Waiting"))
					} else {
						reasonList = append(reasonList, util.ConcatReason(container.State.Waiting.Message, container.State.Waiting.Reason))
					}
				} else {
					if container.State.Terminated.Message == "" && container.State.Terminated.Reason == "" {
						reasonList = append(reasonList, util.ConcatReason("Terminated", "Terminated"))
					} else {
						reasonList = append(reasonList, util.ConcatReason(container.State.Terminated.Message, container.State.Terminated.Reason))
					}
				}
			}
		}
		if !haveWrong {
			// 走到这里视作无异常 作pod成功处理
			return constant.K8sResStatusSucceed, ""
		} else {
			// 其余统一认作失败
			fullReason := strings.Join(reasonList, "\n")
			return constant.K8sResStatusFail, fullReason
		}
	case v1.PodFailed, v1.PodUnknown:
		// 失败
		reason := util.ConcatReason(m.Status.Message, m.Status.Reason)
		return constant.K8sResStatusFail, reason
	case v1.PodSucceeded:
		return constant.K8sResStatusSucceed, ""
	default:
		return constant.K8sResStatusSucceed, ""
	}
}

func (m *MyPod) GetKind() constant.K8sResKind {
	return constant.PodKind
}

func (m *MyPod) GetParentName(indexerMap map[constant.K8sResKind]cache.Indexer) (name string, err error) {
	nameSpace := m.GetNamespace()
	// 向上处理
	rsIndexer := indexerMap[constant.ReplicaSetKind]
	depIndexer := indexerMap[constant.DeploymentKind]
	// 寻找rs
	rsInter, err := getController(rsIndexer, nameSpace, m.GetOwnerReferences()...)
	if err != nil {
		// 查找不到 代表缓存中还没有pod所属的rs信息 等待下次
		return "", errors.Wrap(err, "孤儿节点，暂不添加")
	}
	rs, _ := rsInter.(*v12.ReplicaSet)
	// 寻找dep
	depInter, err := getController(depIndexer, nameSpace, rs.GetOwnerReferences()...)
	if err != nil {
		// 查找不到 代表缓存中还没有pod所属的dep信息 等待下次
		return "", errors.Wrap(err, "孤儿节点，暂不添加")
	}
	dep, _ := depInter.(*v12.Deployment)
	return dep.GetName(), nil
}

func (m *MyPod) AddRel(keyCache *ResourceKeyCache, indexerMap map[constant.K8sResKind]cache.Indexer) (*ResourceCache, error) {
	// 对于pod来说 如果自身不存在 新建并设置到cacheMap ----> 查找上级rs 如果存在 建立关联关系
	nameSpace := m.GetNamespace()
	dptname, err := m.GetParentName(indexerMap)
	if err != nil {
		return nil, err
	}

	// 建立关联
	depResourceCacheKey := util.ConcatResourceCacheKey(nameSpace, dptname)
	depResourceCache := keyCache.GetResourceCacheBYKey(depResourceCacheKey)
	if depResourceCache.IsNil() {
		// dep存在 建立关联
		// 查找不到 代表缓存中还没有pod所属的dep信息 等待下次
		return nil, errors.New("孤儿节点，暂不添加")
	}
	// 处理自身
	name := m.GetName()
	podResourceCacheKey := util.ConcatResourceCacheKey(nameSpace, name)
	status, reason := m.GetStatus()
	podResourceCache := newResourceCache(podResourceCacheKey, name, reason, depResourceCache, status, constant.PodKind, m.Pod)

	depResourceCache.AddChild(podResourceCache)
	keyCache.setResourceCacheBYKey(podResourceCacheKey, podResourceCache)
	return podResourceCache, nil
}

func (m *MyPod) GetMeta() interface{} {
	return m.Pod
}
