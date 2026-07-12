package contract

const (
	// SecurityGroup identifies the canonical Security module API group.
	SecurityGroup = "/security"
	// OverviewCollection identifies the security overview API route fragment.
	OverviewCollection = "/overview"
	// OverviewMenuPath identifies the canonical Security overview UI route.
	OverviewMenuPath = "/security/overview"
	// OverviewAPIPath identifies the canonical Security overview API path.
	OverviewAPIPath = SecurityGroup + OverviewCollection
)
