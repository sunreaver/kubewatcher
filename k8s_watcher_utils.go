package kubewatcher

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"github.com/sunreaver/kubewatcher/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	informerDefaultResync    = 30 * time.Minute
	waitCacheSyncDoneTimeout = 3 * time.Minute // 等待缓存同步完成时间
)

func MakeRestConfigByBearerToken(host, bearerToken string) (*rest.Config, error) {
	return &rest.Config{
		Host:        host,
		APIPath:     "/apis",
		BearerToken: string(bearerToken),
	}, nil
}

func MakeRestConfigByKubeconfigPath(host, kubeconfigPath string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags(host, kubeconfigPath)
}

/*
构造可以查询k8s标准资源的客户端
*/
func BuildK8sClient(config rest.Config) (*kubernetes.Clientset, error) {
	// 不设置报错： GroupVersion is required when initializing a RESTClient
	config.GroupVersion = &corev1.SchemeGroupVersion
	// 不设置报错： NegotiatedSerializer is required when initializing a RESTClient
	config.NegotiatedSerializer = scheme.Codecs
	// 创建一个RESTClientInterface对象
	restClient, err := rest.RESTClientFor(&config)
	if err != nil {
		util.Errorw("BuildDCEClient", "config", config, "RESTClientFor err", err.Error())
		return nil, err
	}
	// create the clientset
	clientset, err := kubernetes.NewForConfigAndClient(&config, restClient.Client)
	if err != nil {
		util.Errorw("BuildDCEClient", "config", config, "NewForConfigAndClient err", err.Error())
		return nil, err
	}
	return clientset, nil
}

/*
根据clientSet 构造 informer
*/
func BuildResInformerByClientSet(ctx context.Context, clientSet *kubernetes.Clientset) (*K8sWatcherInformer, error) {
	if clientSet == nil {
		return nil, errors.New("clientSet can't be null")
	}
	// informer的关闭清理开关
	informerCtx, informerCancelFn := context.WithCancel(ctx)

	stopCh := informerCtx.Done()

	// sharedInformers可以将多种资源的监听使用共享的cache
	sharedInformers := informers.NewSharedInformerFactory(clientSet, informerDefaultResync)

	// sharedInformers实现各种k8s内置资源的informer
	// informer通过List-Watch机制监听资源的变化，然后将变化后的资源存储在LocalStore中
	depInformer := sharedInformers.Apps().V1().Deployments()
	podInformer := sharedInformers.Core().V1().Pods()
	rsInformer := sharedInformers.Apps().V1().ReplicaSets()

	sharedInformerStartFn := func() error {
		// 启动informer开始缓存数据
		sharedInformers.Start(stopCh)
		// 等待缓存同步完成
		ctx, cancel := context.WithTimeout(context.Background(), waitCacheSyncDoneTimeout)
		go func() {
			// 防止informer异常，一直阻塞在sharedInformers.WaitForCacheSync(stopCh)上，这里对缓存加载逻辑做超时处理
			select {
			case <-ctx.Done(): // 超时了或者缓存完成了
				if ctx.Err() == context.DeadlineExceeded { // 超时还没同步完缓存，认为这个informer有问题，停止这个平台的读取，继续下一个平台
					informerCancelFn()
				}
			}
		}()
		sharedInformers.WaitForCacheSync(stopCh) // 正常缓存完成或者close(stopCh)就不再阻塞执行
		cancel()
		if ctx.Err() == context.DeadlineExceeded {
			return errors.Wrap(ctx.Err(), "informer缓存超时")
		}
		return nil
	}

	k8sInformer := &K8sWatcherInformer{
		DepInformer:      depInformer.Informer(),
		PodInformer:      podInformer.Informer(),
		RSInformer:       rsInformer.Informer(),
		informerStartCtx: informerCtx,
		informerStartFn:  sharedInformerStartFn,
		informerStopFn:   informerCancelFn,
	}

	return k8sInformer, nil
}
