package controller

import (
	"context"
	"kubewatcher/constant"
	"kubewatcher/resource"

	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type K8sController interface {
	GetIndexer() map[constant.K8sResKind]cache.Indexer // 从controller获取查询资源的indexer，key 是资源kind，value是该资源的indexer用于查找该资源数据
	GetKind() constant.K8sResKind                      // 获取controller对应的类型 例如 Deployment或者Pod
	SetHandler(h K8sControllerHandler)                 // 获取controller监听到变化的处理器
	GetQueue() workqueue.RateLimitingInterface         // 从controller上获取其queue
	GetWorkerNum() int                                 // 获取controller 消费queue的协程数量
	KeyConsume(ctx context.Context, key string) error  // controller消费key的处理方法，该方法一般用于判断key对应的对象是否存在，然后调用handler.Handle()进一步处理
	GetCacheMap() *resource.ResourceKeyCache           // 获取缓存map
}

/*
可以处理controller监听对象的处理器
*/
type K8sControllerHandler interface {
	Handle(ctx context.Context, c K8sController, key string, obj interface{}) error
}
