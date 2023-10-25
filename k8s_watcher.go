package kubewatcher

import (
	"context"

	"github.com/pkg/errors"
	"github.com/sunreaver/kubewatcher/controller"
	"github.com/sunreaver/kubewatcher/resource"
	"github.com/sunreaver/kubewatcher/sender"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

/*
一个watcher监控一个集群的内容，并根据回调通知推送对象给使用方
支持从配置文件、clientSet、已创建好的informer三种方式启动watcher
*/
type K8sWatcher struct {
	ctx       context.Context
	clientSet *kubernetes.Clientset      // watcher构造informer的clientset，当使用FromClientSet().start()时传入这个
	informer  *K8sWatcherInformer        // watcher使用的informer，当使用FromInformer().start()时传入这个，且同时传入informer.InformerStartCtx
	err       error                      // watcher启动过程中的错误
	sender    *sender.Sender             // 负责资源事件的向外发送
	keyCache  *resource.ResourceKeyCache // 存储资源缓存的相关信息 key为资源key value为资源信息
}

/*
使用外部的clientSet对象启动一个watcher
*/
func AsyncStartWatcherByClientSet(ctx context.Context, clientSet *kubernetes.Clientset) (*K8sWatcher, error) {
	sd := sender.NewSender()
	watcher := &K8sWatcher{
		ctx:       ctx,
		clientSet: clientSet,
		sender:    sd,
		keyCache:  resource.NewResourceKeyCache(),
	}
	return watcher, watcher.fromClientSet().start()
}

/*
使用外部的informer对象启动一个watcher
当使用这个方法时，ctx应该是informer的启动ctx
*/
func AsyncStartWatcherByInformer(ctx context.Context, platform string, informer *K8sWatcherInformer) (*K8sWatcher, error) {
	sd := sender.NewSender()
	watcher := &K8sWatcher{
		ctx:      ctx,
		informer: informer,
		sender:   sd,
		keyCache: resource.NewResourceKeyCache(),
	}
	return watcher, watcher.fromInformer().start()
}

type K8sWatcherInformer struct {
	DepInformer      cache.SharedIndexInformer
	PodInformer      cache.SharedIndexInformer
	RSInformer       cache.SharedIndexInformer
	informerStartCtx context.Context // 如果是通过informer类型启动，这个ctx是外部informer的ctx，cfg、clientSet启动会从父ctx来自动设置这个ctx
	informerStartFn  func() error    // 通过cfg或者clientset创建的informer启动方法
	informerStopFn   func()          // 通过cfg或者clientset创建的informer的关闭方法，是context的cancel，用来关闭informer和controller
}

/*
适用于传入cfg或者clientSet方式启动watcher对象的关闭
外部传入informer方式启动的watcher对象关闭受外部informer的控制
*/
func (w *K8sWatcher) Close() {
	if w.informer != nil && w.informer.informerStopFn != nil { // 这个是context 的cancel方法，重复调用没问题
		w.informer.informerStopFn() // 关闭informer和controller
	}
}

func (w *K8sWatcher) Check() error {
	if w == nil {
		return errors.New("nil")
	}
	if w.err != nil {
		return w.err
	}
	if w.ctx == nil {
		return errors.New("K8sWatcher ctx can't be null")
	}
	if w.sender == nil {
		return errors.New("sender can't be null")
	}
	if w.informer == nil {
		return errors.New("informer can't be null")
	}
	if w.informer.informerStartCtx == nil {
		return errors.New("Informer InformerStartCtx can't be null")
	}
	if w.informer.DepInformer == nil {
		return errors.New("DepInformer can't be null")
	}
	if w.informer.PodInformer == nil {
		return errors.New("PodInformer can't be null")
	}
	if w.informer.RSInformer == nil {
		return errors.New("RSInformer can't be null")
	}
	return nil
}

func (w *K8sWatcher) fromClientSet() *K8sWatcher {
	if w.err != nil {
		return w
	}
	informer, err := BuildResInformerByClientSet(w.ctx, w.clientSet)
	if err != nil {
		w.err = err
		return w
	}
	w.informer = informer
	return w

}

func (w *K8sWatcher) fromInformer() *K8sWatcher {
	w.informer.informerStartCtx = w.ctx
	return w
}

func (w *K8sWatcher) AddPodCallback(fnList ...func(out sender.SendOut)) {
	if w.sender != nil {
		w.sender.AddPodCallback(fnList...)
	}
}

func (w *K8sWatcher) AddDepCallback(fnList ...func(out sender.SendOut)) {
	if w.sender != nil {
		w.sender.AddDepCallback(fnList...)
	}
}

/*
使用示例：
c.fromDCECfg().start()  或者 c.fromInformer().start()
*/
func (w *K8sWatcher) start() (err error) {
	defer func() {
		if err != nil {
			w.Close()
		}
	}()
	if err = w.Check(); err != nil {
		return err
	}
	w.startSender()          // 启动监听器的sender
	w.startController()      // 启动controller
	return w.startInformer() // 启动informer开始同步数据
}

/*
启动监听器的sender
*/
func (w *K8sWatcher) startSender() {
	w.sender.Start(w.informer.informerStartCtx) // 启动监听器的sender
}

/*
启动watcher的使用的controller
*/
func (w *K8sWatcher) startController() {
	ctx := w.informer.informerStartCtx
	podInformer := w.informer.PodInformer
	depInformer := w.informer.DepInformer
	rsInformer := w.informer.RSInformer

	handAndSender := NewHandAndSender(w.sender)
	controller.BuildDeploymentController(ctx, depInformer, handAndSender, w.keyCache)
	controller.BuildPodController(ctx, podInformer, depInformer, rsInformer, handAndSender, w.keyCache)
}

/*
在controller运行后，启动informer开始缓存数据
*/
func (w *K8sWatcher) startInformer() error {
	if w.informer.informerStartFn != nil {
		return w.informer.informerStartFn()
	}
	return nil
}
