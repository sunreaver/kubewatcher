package sender

import (
	"context"
	"kubewatcher/constant"
)

// 各字段与cache.go中基本一致
type SendOut struct {
	Key           string
	Kind          constant.K8sResKind
	Name          string
	Status        constant.K8sResStatus
	Reason        string
	ControllerKey string
	Meta          interface{}
}

type Sender struct {
	podCallback []func(SendOut) // 存储pod类型回调方法
	depCallback []func(SendOut) // 存储deployment类型回调方法
	ch          chan SendOut    // 存储消息
}

func NewSender() *Sender {
	return &Sender{
		podCallback: []func(SendOut){},
		depCallback: []func(SendOut){},
		ch:          make(chan SendOut, 10),
	}
}

func NewSenderWithCBFn(podCallback, depCallback []func(SendOut)) *Sender {
	return &Sender{
		podCallback: podCallback,
		depCallback: depCallback,
		ch:          make(chan SendOut, 10),
	}
}

func (s *Sender) AddPodCallback(fn ...func(out SendOut)) {
	s.podCallback = append(s.podCallback, fn...)
}

func (s *Sender) AddDepCallback(fn ...func(out SendOut)) {
	s.depCallback = append(s.depCallback, fn...)
}

func (s *Sender) AddSendOut(cache SendOut) {
	s.ch <- cache
}

func (s *Sender) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case sendOut := <-s.ch:
				var funcList []func(SendOut)
				if sendOut.Kind == constant.PodKind {
					funcList = s.podCallback
				} else if sendOut.Kind == constant.DeploymentKind {
					funcList = s.depCallback
				}
				for _, fn := range funcList {
					fn(sendOut)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}
