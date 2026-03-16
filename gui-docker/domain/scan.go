package domain

import "time"

type ResourceKind string

const (
	ResourceManaged       ResourceKind = "managed"
	ResourceOrphanRuntime ResourceKind = "orphan_runtime"
	ResourceOrphanDir     ResourceKind = "orphan_dir"
	ResourceBrokenApp     ResourceKind = "broken_app"
	ResourceStaleAdmin    ResourceKind = "stale_admin"
	ResourceUnknown       ResourceKind = "unknown"
)

type Confidence string

const (
	ConfidenceHigh   Confidence = "high"
	ConfidenceMedium Confidence = "medium"
	ConfidenceLow    Confidence = "low"
)

type ScanResource struct {
	Kind           ResourceKind
	Confidence     Confidence
	Name           string
	AppID          string
	ContainerNames []string
	ComposeProject string
	Dir            string
	Ports          []string
	Status         string
	Reason         string
	CleanupCmds    []string
	IsCurrentAdmin bool
}

type ScanReport struct {
	ScannedAt time.Time
	Resources []ScanResource
	StacksDir string
}
