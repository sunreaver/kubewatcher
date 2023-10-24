package resource

import (
	"errors"
	"kubewatcher/constant"
	"kubewatcher/util"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

var ErrNoParent = errors.New("no parent")

type ResourceInter interface {
	GetStatus() (constant.K8sResStatus, string) // 各种资源判定自身状态及失败原因方法 返回值依次为 状态-失败原因-自身失败原因 两种失败原因解释见cache.go
	GetKind() constant.K8sResKind
	AddRel(*ResourceKeyCache, map[constant.K8sResKind]cache.Indexer) (*ResourceCache, error) // 各种资源关于加入缓存树的方法
	GetMeta() interface{}
}

// 工具方法 用于从底层资源往上级查询上层资源
func getController(indexer cache.Indexer, nameSpace string, references ...v1.OwnerReference) (interface{}, error) {
	if len(references) == 0 {
		return nil, errors.New("references is nil")
	}
	reference := references[0]
	if !*reference.Controller {
		return nil, errors.New("references0 is not controller")
	}
	realKey := util.ConcatRealKey(nameSpace, references[0].Name)
	inter, exist, _ := indexer.GetByKey(realKey)
	if !exist {
		return nil, errors.New("not exist")
	}
	return inter, nil
}
