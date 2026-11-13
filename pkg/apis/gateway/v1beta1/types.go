package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	gatewayv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const GroupName = "gateway.networking.k8s.io"

var GroupVersion = schema.GroupVersion{Group: GroupName, Version: "v1beta1"}

type Group = gatewayv1.Group
type Kind = gatewayv1.Kind
type Namespace = gatewayv1.Namespace
type ObjectName = gatewayv1.ObjectName

type ReferenceGrant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ReferenceGrantSpec `json:"spec,omitempty"`
}

type ReferenceGrantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReferenceGrant `json:"items"`
}

type ReferenceGrantSpec struct {
	From []ReferenceGrantFrom `json:"from"`
	To   []ReferenceGrantTo   `json:"to"`
}

type ReferenceGrantFrom struct {
	Group     Group     `json:"group"`
	Kind      Kind      `json:"kind"`
	Namespace Namespace `json:"namespace"`
}

type ReferenceGrantTo struct {
	Group Group       `json:"group"`
	Kind  Kind        `json:"kind"`
	Name  *ObjectName `json:"name,omitempty"`
}
