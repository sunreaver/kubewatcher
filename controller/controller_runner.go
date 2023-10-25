package controller

import (
	"context"
	"time"

	"github.com/sunreaver/kubewatcher/resource"
	"github.com/sunreaver/kubewatcher/util"
	k8sruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

// controller的运行器，按照启动多个worker去消费controller的queue数据的流程去运行
type ControllerRunner struct {
	Controller K8sController
}

func NewControllerRunner(c K8sController) *ControllerRunner {
	return &ControllerRunner{
		Controller: c,
	}
}

/*
从queue逐个取key去处理
*/
func (cr *ControllerRunner) processNextItem(ctx context.Context) bool {
	var key interface{}
	quit := false
	defer util.Recover()
	c := cr.Controller
	cqueue := c.GetQueue()
	key, quit = cqueue.Get()
	if quit {
		return false
	}
	defer cqueue.Done(key)
	err := c.KeyConsume(ctx, key.(string))
	cr.handleErr(err, key)
	return true
}

/*
当处理key发生错误时重试
*/
func (cr *ControllerRunner) handleErr(err error, key interface{}) {
	c := cr.Controller
	cqueue := c.GetQueue()
	if err == nil {
		cqueue.Forget(key)
		return
	}
	if cqueue.NumRequeues(key) < 5 { // 允许重试5次
		cqueue.AddRateLimited(key)
		return
	}
	cqueue.Forget(key)
	k8sruntime.HandleError(err)
}

func (cr *ControllerRunner) runWorker(ctx context.Context) {
	for cr.processNextItem(ctx) {
	}
}

func (cr *ControllerRunner) RunController(ctx context.Context) {
	c := cr.Controller
	defer c.GetQueue().ShutDown()

	for i := 0; i < c.GetWorkerNum(); i++ {
		go wait.UntilWithContext(ctx, cr.runWorker, time.Second)
	}
	<-ctx.Done()
}

/*
创建deployment类型的监听controller
不直接使用informer.EventHandler直接处理数据，而是使用workqueue方式处理的原因是：
1. informer.EventHandler是同步调用的，即下一个事件必须等待上一事件完成，可能导致长时间阻塞
2. 官方推荐使用workqueue controller这种方式处理对象变化（Add/Update/Del->queue->runWorker->syncHandler模式）
3. workqueue中只存放key(namespace/meta.name)，多条事件过来其实都是同一个key，直到处理的那一刻取最新数据处理，达到合并处理的效果
4. workqueue 有失败重试机制，可以避免一个event处理失败了丢失处理问题
5. workqueue 作为缓冲机制，可以启用多个协程处理queue数据
*/
func BuildDeploymentController(ctx context.Context, depInformer cache.SharedIndexInformer, handler K8sControllerHandler, keyCache *resource.ResourceKeyCache) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	depInformer.AddEventHandler(NewDeploymentEventHandlerForQueue(queue)) // 为deployment informer注册事件入queue方法
	// 构造deployment controller
	depController := NewDeploymentController(queue, depInformer.GetIndexer(), keyCache)
	depController.SetHandler(handler)
	// depController.SetWorkerNum(10)
	runner := NewControllerRunner(depController)
	go runner.RunController(ctx)
}

func BuildPodController(ctx context.Context, podInformer, depInformer, rsInformer cache.SharedIndexInformer, handler K8sControllerHandler, keyCache *resource.ResourceKeyCache) {
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	podInformer.AddEventHandler(NewPodEventHandlerForQueue(queue)) // 为pod informer注册事件入queue方法
	// 构造pod controller
	podController := NewPodController(queue, podInformer.GetIndexer(), depInformer.GetIndexer(), rsInformer.GetIndexer(), keyCache)
	podController.SetHandler(handler)
	// podController.SetWorkerNum(10)
	runner := NewControllerRunner(podController)
	go runner.RunController(ctx)
}
