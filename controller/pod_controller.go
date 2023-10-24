package controller

import (
	"context"
	"fmt"
	"kubewatcher/constant"
	"kubewatcher/resource"
	"kubewatcher/util"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type PodController struct {
	platform   string
	queue      workqueue.RateLimitingInterface
	podIndexer cache.Indexer
	depIndexer cache.Indexer
	rsIndexer  cache.Indexer
	handler    K8sControllerHandler
	workerNum  int
	keyCache   *resource.ResourceKeyCache
}

func NewPodController(queue workqueue.RateLimitingInterface, podIndexer, depIndexer, rsIndexer cache.Indexer, keyCache *resource.ResourceKeyCache) *PodController {
	return &PodController{
		queue:      queue,
		podIndexer: podIndexer,
		depIndexer: depIndexer,
		rsIndexer:  rsIndexer,
		workerNum:  1,
		keyCache:   keyCache,
	}
}

func (c *PodController) SetHandler(handler K8sControllerHandler) {
	c.handler = handler
}

/*
从queue逐个取key去处理
*/
func (c *PodController) GetIndexer() map[constant.K8sResKind]cache.Indexer {
	return map[constant.K8sResKind]cache.Indexer{
		constant.PodKind:        c.podIndexer,
		constant.DeploymentKind: c.depIndexer,
		constant.ReplicaSetKind: c.rsIndexer,
	}
}

func (c *PodController) GetPlatform() string {
	return c.platform
}

func (c *PodController) GetKind() constant.K8sResKind {
	return constant.PodKind
}

func (c *PodController) GetQueue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c *PodController) GetWorkerNum() int {
	return c.workerNum
}

func (c *PodController) SetWorkerNum(workerNum int) {
	c.workerNum = workerNum
}

func (c *PodController) KeyConsume(ctx context.Context, key string) error {
	obj, exists, err := c.podIndexer.GetByKey(key)
	if err != nil {
		return err
	}
	methodKey := BuildWatcherKeyFunc(WatcherKeyPrefixUpdate, key)
	if !exists {
		methodKey = BuildWatcherKeyFunc(WatcherKeyPrefixDelete, key)
		util.Debugw("dealPod", "pod", fmt.Sprintf("%s does not exist\n", key))
	}
	// 处理新增、更新、删除
	return c.handler.Handle(ctx, c, methodKey, obj)
}

func (c *PodController) GetCacheMap() *resource.ResourceKeyCache {
	return c.keyCache
}

/*
这个是pod informer注册的实际处理方法，
*/
func NewPodEventHandlerForQueue(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(newObj)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}
}
