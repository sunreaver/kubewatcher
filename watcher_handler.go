package kubewatcher

import (
	"context"
	"kubewatcher/constant"
	"kubewatcher/controller"
	"kubewatcher/resource"
	sender2 "kubewatcher/sender"
	"kubewatcher/util"

	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

// 用于同步处理pod或者deployment推送过来的更新
type HandAndSender struct {
	sender *sender2.Sender
}

func NewHandAndSender(sender *sender2.Sender) *HandAndSender {
	return &HandAndSender{
		sender: sender,
	}
}

func (hs *HandAndSender) GetSender() *sender2.Sender {
	return hs.sender
}

func (hs *HandAndSender) Handle(ctx context.Context, c controller.K8sController, key string, obj interface{}) error {
	t := c.GetKind()
	var value resource.ResourceInter
	switch t {
	case constant.DeploymentKind:
		var d *appv1.Deployment
		if obj != nil {
			d = obj.(*appv1.Deployment).DeepCopy() // 避免修改到缓存数据
			util.Debugw("Deployment Handle", "current", d.Status)
		}
		value = &resource.MyDep{Deployment: d}
	case constant.PodKind:
		var p *corev1.Pod
		if obj != nil {
			p = obj.(*corev1.Pod).DeepCopy() // 避免修改到缓存数据
			util.Debugw("Pod Handle", "current", p.Status)
		}
		value = &resource.MyPod{Pod: p}
	}

	// 处理新增\更新\删除
	return handler(key, value, c, hs)
}
