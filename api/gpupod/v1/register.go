package v1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var SchemeGroupVersion = GroupVersion

func Resource(resource string) schema.GroupResource {
	return SchemeGroupVersion.WithResource(resource).GroupResource()
}

func Kind(kind string) schema.GroupKind {
	return SchemeGroupVersion.WithKind(kind).GroupKind()
}
