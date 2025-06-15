package clients

import "k8s.io/apimachinery/pkg/api/resource"

func resourceMustParse(value string) resource.Quantity {
	q, err := resource.ParseQuantity(value)
	if err != nil {
		panic(err)
	}
	return q
}
