package resource

import (
	"kubewatcher/constant"
	"kubewatcher/sender"
	"kubewatcher/util"
	"sync"

	"golang.org/x/exp/slices"
)

type ResourceCache struct {
	cacheTreeLock sync.RWMutex          // 操作父子关系节点树的锁
	key           string                // 资源key 与queue中的key一致 都是租户/资源名
	name          string                // 资源名
	parent        *ResourceCache        // 父节点 一个子只能有一个父 当前设计中仅保存pod和deployment 例如一个pod资源的父节点为一个deployment节点 无父亲设置为nil
	child         []*ResourceCache      // 子节点 一个父可以有多个子 当前设计中仅保存pod和deployment 例如一个deployment资源的子节点为n个pod节点 无子节点设置为空数组 删除子节点时不直接删除 而是设置为nil 等到数组数量达到一定值再进行一次清理操作
	reason        string                // 记录该资源自身的失败原因(如果有) 例如pod为其下属的容器的失败原因之和 deployment为其下的reason字段和message字段
	status        constant.K8sResStatus // 资源状态
	kind          constant.K8sResKind   // 资源种类 目前只有pod和deployment
	meta          interface{}           // 源数据 指未经过任何处理的k8s原生数据
}

func newResourceCache(key, name, reason string, parent *ResourceCache, status constant.K8sResStatus, kind constant.K8sResKind, meta interface{}) *ResourceCache {
	return &ResourceCache{
		cacheTreeLock: sync.RWMutex{},
		key:           key,
		name:          name,
		parent:        parent,
		child:         make([]*ResourceCache, 0),
		reason:        reason,
		status:        status,
		kind:          kind,
		meta:          meta,
	}
}

func (r *ResourceCache) GetParent() *ResourceCache {
	return r.parent
}

func (r *ResourceCache) RemoveChild(childKey string) {
	if r == nil {
		return
	}
	r.cacheTreeLock.Lock()
	defer r.cacheTreeLock.Unlock()
	slices.DeleteFunc(r.child, func(rc *ResourceCache) bool { return rc.key == childKey })
}

func (r *ResourceCache) AddChild(child *ResourceCache) {
	if r == nil {
		return
	}
	r.cacheTreeLock.Lock()
	defer r.cacheTreeLock.Unlock()
	r.child = append(r.child, child)
	if child != r {
		child.parent = r
	}
}

// 遍历Child，不能在range中执行删除操作，会死锁
func (r *ResourceCache) RangeWithoutDelete(fn func(rc *ResourceCache) (stop bool)) {
	r.cacheTreeLock.RLock()
	defer r.cacheTreeLock.RUnlock()
	for _, v := range r.child {
		if stop := fn(v); stop {
			break
		}
	}
	return
}

func (r *ResourceCache) SetStatus(status constant.K8sResStatus) {
	r.status = status
}

func (r *ResourceCache) GetStatus() constant.K8sResStatus {
	return r.status
}

func (r *ResourceCache) SetReason(reason string) {
	r.reason = reason
}

func (r *ResourceCache) GetReason() string {
	return r.reason
}

func (r *ResourceCache) SetMeta(meta interface{}) {
	r.meta = meta
}

func (r *ResourceCache) GetMeta() interface{} {
	return r.meta
}

func (r *ResourceCache) IsNil() bool {
	return r == nil
}

func (r *ResourceCache) GetKey() string {
	return r.key
}

func (r *ResourceCache) GetKind() constant.K8sResKind {
	return r.kind
}

// 是否孤儿
func (r *ResourceCache) IsSingle() bool {
	r.cacheTreeLock.RLock()
	defer r.cacheTreeLock.RUnlock()
	// 有parent--pod
	// 有child--depoy
	return r.parent == nil && len(r.child) == 0
}

// 通过cache生成监控包推送给外界的结构体数据
func (r *ResourceCache) GetSendOut() sender.SendOut {
	r.cacheTreeLock.RLock()
	defer r.cacheTreeLock.RUnlock()
	sendOutKey := util.ParseResourceCacheKey(r.key)
	sendOut := sender.SendOut{
		Key:    sendOutKey,
		Kind:   r.kind,
		Name:   r.name,
		Status: r.status,
		Reason: r.reason,
		Meta:   r.meta,
	}
	if r.parent != nil {
		sendOut.ControllerKey = r.parent.key
	}
	return sendOut
}

type ResourceKeyCache struct {
	kv map[string]*ResourceCache
	sync.RWMutex
}

func (r *ResourceKeyCache) setResourceCacheBYKey(key string, value *ResourceCache) {
	r.Lock()
	defer r.Unlock()
	r.kv[key] = value
}

func (r *ResourceKeyCache) GetResourceCacheBYKey(key string) *ResourceCache {
	r.RLock()
	defer r.RUnlock()
	return r.kv[key]
}

func (r *ResourceKeyCache) DeleteCacheByKey(key string) {
	r.Lock()
	defer r.Unlock()
	delete(r.kv, key)
}

func NewResourceKeyCache() *ResourceKeyCache {
	return &ResourceKeyCache{
		kv: map[string]*ResourceCache{},
	}
}
