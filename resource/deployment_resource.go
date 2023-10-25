package resource

import (
	"github.com/sunreaver/kubewatcher/constant"
	"github.com/sunreaver/kubewatcher/util"
	appv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/cache"
)

type MyDep struct {
	*appv1.Deployment
}

func (m *MyDep) GetStatus() (constant.K8sResStatus, string) {
	reason := ""
	for _, condition := range m.Status.Conditions {
		if condition.Type == appv1.DeploymentProgressing {
			reason = util.ConcatReason(condition.Message, condition.Reason)
			break
		}
	}
	status := m.Status
	if status.UpdatedReplicas == *(m.Spec.Replicas) && status.Replicas == *(m.Spec.Replicas) && status.AvailableReplicas == *(m.Spec.Replicas) && status.ObservedGeneration >= m.Generation {
		// 仅此一种情况视为成功
		return constant.K8sResStatusSucceed, reason
	}

	return constant.K8sResStatusFail, reason
}

func (m *MyDep) GetKind() constant.K8sResKind {
	return constant.DeploymentKind
}

func (m *MyDep) GetParent(indexerMap map[constant.K8sResKind]cache.Indexer) (name string, err error) {
	return "", ErrNoParent
}

func (m *MyDep) AddRel(keyCache *ResourceKeyCache, indexMap map[constant.K8sResKind]cache.Indexer) (*ResourceCache, error) {
	name := m.GetName()
	nameSpace := m.GetNamespace()
	// 处理自身
	depResourceCacheKey := util.ConcatResourceCacheKey(nameSpace, name)
	status, reason := m.GetStatus()
	depResourceCache := newResourceCache(depResourceCacheKey, name, reason, nil, status, constant.DeploymentKind, m.Deployment)
	keyCache.setResourceCacheBYKey(depResourceCacheKey, depResourceCache)
	return depResourceCache, nil
}

func (m *MyDep) GetMeta() interface{} {
	return m.Deployment
}
