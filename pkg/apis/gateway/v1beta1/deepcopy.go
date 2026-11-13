package v1beta1

import (
	runtime "k8s.io/apimachinery/pkg/runtime"
)

func (in *ReferenceGrant) DeepCopyInto(out *ReferenceGrant) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
}

func (in *ReferenceGrant) DeepCopy() *ReferenceGrant {
	if in == nil {
		return nil
	}
	out := new(ReferenceGrant)
	in.DeepCopyInto(out)
	return out
}

func (in *ReferenceGrant) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ReferenceGrantList) DeepCopyInto(out *ReferenceGrantList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		out.Items = make([]ReferenceGrant, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}
}

func (in *ReferenceGrantList) DeepCopy() *ReferenceGrantList {
	if in == nil {
		return nil
	}
	out := new(ReferenceGrantList)
	in.DeepCopyInto(out)
	return out
}

func (in *ReferenceGrantList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

func (in *ReferenceGrantSpec) DeepCopyInto(out *ReferenceGrantSpec) {
	*out = *in
	if in.From != nil {
		out.From = append([]ReferenceGrantFrom(nil), in.From...)
	}
	if in.To != nil {
		out.To = make([]ReferenceGrantTo, len(in.To))
		for i := range in.To {
			in.To[i].DeepCopyInto(&out.To[i])
		}
	}
}

func (in *ReferenceGrantSpec) DeepCopy() *ReferenceGrantSpec {
	if in == nil {
		return nil
	}
	out := new(ReferenceGrantSpec)
	in.DeepCopyInto(out)
	return out
}

func (in *ReferenceGrantTo) DeepCopyInto(out *ReferenceGrantTo) {
	*out = *in
	if in.Name != nil {
		out.Name = new(ObjectName)
		*out.Name = *in.Name
	}
}

func (in *ReferenceGrantTo) DeepCopy() *ReferenceGrantTo {
	if in == nil {
		return nil
	}
	out := new(ReferenceGrantTo)
	in.DeepCopyInto(out)
	return out
}
