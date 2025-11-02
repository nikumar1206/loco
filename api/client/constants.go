package client

const (
	// Gateway and namespace
	LocoGatewayName = "eg"
	LocoNS          = "loco-system"

	// Deployment constants
	DefaultReplicas        = int32(1)
	MaxReplicaHistory      = int32(2)
	MaxSurgePercent        = "25%"
	MaxUnavailablePercent  = "25%"
	TerminationGracePeriod = int64(60)

	// Service
	DefaultServicePort     = int32(80)
	SessionAffinityTimeout = int32(10800) // 3 hrs

	// Probe constants
	DefaultStartupGracePeriod = 30
	DefaultTimeout            = 5
	DefaultInterval           = 10
	DefaultFailureThreshold   = 3

	// Default Resource constants
	DefaultCPU    = "100m"
	DefaultMemory = "128Mi"

	// Timeout constants
	DefaultRequestTimeout = "30s"

	// TimeFormat
	DefaultTimeFormat = "2006-01-02T15:04:05-0700"
)
