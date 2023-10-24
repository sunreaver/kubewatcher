package constant

type K8sResKind string

const (
	PodKind        K8sResKind = "Pod"
	ReplicaSetKind K8sResKind = "ReplicaSet"
	DeploymentKind K8sResKind = "Deployment"
	ServiceKind    K8sResKind = "Service"
)

type K8sResStatus string

const (
	K8sResStatusDefault K8sResStatus = "default"
	K8sResStatusFail    K8sResStatus = "failed"
	K8sResStatusSucceed K8sResStatus = "succeed"
	K8sResStatusDelete  K8sResStatus = "delete" // 某个资源被删除时推送
)

func (r K8sResStatus) IsDelete() bool {
	return r == K8sResStatusDelete
}
