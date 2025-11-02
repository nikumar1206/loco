package client

import (
	"fmt"

	"k8s.io/apimachinery/pkg/api/resource"
)

func ValidateResources(cpuStr, memStr string) error {
	cpuQty, err := resource.ParseQuantity(cpuStr)
	if err != nil {
		return fmt.Errorf("invalid CPU quantity: %w", err)
	}

	memQty, err := resource.ParseQuantity(memStr)
	if err != nil {
		return fmt.Errorf("invalid memory quantity: %w", err)
	}

	maxCPU := resource.MustParse("500m")
	maxMem := resource.MustParse("1Gi")

	if cpuQty.Cmp(maxCPU) == 1 {
		return fmt.Errorf("CPU exceeds 500m: got %s", cpuStr)
	}

	if memQty.Cmp(maxMem) == 1 {
		return fmt.Errorf("memory exceeds 1Gi: got %s", memStr)
	}

	return nil
}
