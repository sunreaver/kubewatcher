package controller

import (
	"context"
	"fmt"

	"github.com/sunreaver/kubewatcher/constant"
	"github.com/sunreaver/kubewatcher/resource"
	"github.com/sunreaver/kubewatcher/util"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type DeploymentController struct {
	queue      workqueue.RateLimitingInterface
	depIndexer cache.Indexer
	handler    K8sControllerHandler
	workerNum  int
	keyCache   *resource.ResourceKeyCache
}

func NewDeploymentController(queue workqueue.RateLimitingInterface, depIndexer cache.Indexer, keyCache *resource.ResourceKeyCache) *DeploymentController {
	return &DeploymentController{
		queue:      queue,
		depIndexer: depIndexer,
		workerNum:  1, // 默认一个queue消费协程
		keyCache:   keyCache,
	}
}

func (c *DeploymentController) SetWorkerNum(workerNum int) {
	c.workerNum = workerNum
}

func (c *DeploymentController) SetHandler(handler K8sControllerHandler) {
	c.handler = handler
}

func (c *DeploymentController) GetIndexer() map[constant.K8sResKind]cache.Indexer {
	return map[constant.K8sResKind]cache.Indexer{constant.DeploymentKind: c.depIndexer}
}

func (c *DeploymentController) GetKind() constant.K8sResKind {
	return constant.DeploymentKind
}

func (c *DeploymentController) GetWorkerNum() int {
	return c.workerNum
}

func (c *DeploymentController) GetQueue() workqueue.RateLimitingInterface {
	return c.queue
}

func (c *DeploymentController) KeyConsume(ctx context.Context, key string) error {
	if c.handler == nil {
		util.Errorw("dealDeployment", "Deployment", "No handler")
		return nil
	}
	obj, exists, err := c.depIndexer.GetByKey(key)
	if err != nil {
		return err
	}
	methodKey := BuildWatcherKeyFunc(WatcherKeyPrefixUpdate, key)
	if !exists {
		methodKey = BuildWatcherKeyFunc(WatcherKeyPrefixDelete, key)
		util.Infow("dealDeployment", "Deployment", fmt.Sprintf("Deployment %s does not exist\n", key))
	}
	// 处理新增、更新、删除
	return c.handler.Handle(ctx, c, methodKey, obj)
}

func (c *DeploymentController) GetCacheMap() *resource.ResourceKeyCache {
	return c.keyCache
}

/*
这个是deployment informer注册的实际处理方法，
*/
func NewDeploymentEventHandlerForQueue(queue workqueue.RateLimitingInterface) cache.ResourceEventHandler {
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
