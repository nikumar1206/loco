package client

import (
	"k8s.io/apimachinery/pkg/api/resource"
	v1Gateway "sigs.k8s.io/gateway-api/apis/v1"
)

func resourceMustParse(value string) resource.Quantity {
	q, err := resource.ParseQuantity(value)
	if err != nil {
		panic(err)
	}
	return q
}

func ptrToString(s string) *string { return &s }

func ptrToPortNumber(p int) *v1Gateway.PortNumber {
	n := v1Gateway.PortNumber(p)
	return &n
}

// func ptrToGroup(g string) *v1Gateway.Group {
// 	gwG := v1Gateway.Group(g)
// 	return &gwG
// }

func ptrToNamespace(n string) *v1Gateway.Namespace {
	ns := v1Gateway.Namespace(n)
	return &ns
}

func ptrToKind(k string) *v1Gateway.Kind {
	t := v1Gateway.Kind(k)
	return &t
}

func ptrToBool(b bool) *bool { return &b }
