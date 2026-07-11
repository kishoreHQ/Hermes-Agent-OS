// Package types defines Hermes domain identifiers.
// Vendor-neutral. Protocol-aligned with AESP where applicable.
package types

type TenantID string
type PrincipalID string
type MissionID string
type SessionID string
type WorkUnitID string
type TraceID string
type ArtifactDigest string
type PluginID string
type Capability string

// TrustLabel for memory and tool results (AESP-aligned).
type TrustLabel string

const (
	TrustSystem        TrustLabel = "system"
	TrustVerified      TrustLabel = "verified"
	TrustAgent         TrustLabel = "agent"
	TrustRetrieved     TrustLabel = "retrieved"
	TrustUntrusted     TrustLabel = "untrusted"
	TrustPoisonSuspect TrustLabel = "poison-suspect"
)

// CostTier for capability routing (never vendor names).
type CostTier string

const (
	TierFreeLocal  CostTier = "free-local"
	TierFreeHosted CostTier = "free-hosted"
	TierBudget     CostTier = "budget"
	TierStandard   CostTier = "standard"
	TierPremium    CostTier = "premium"
)

// AgentMode controls autonomy.
type AgentMode string

const (
	ModeFull    AgentMode = "full"
	ModeAssist  AgentMode = "assist"
	ModeObserve AgentMode = "observe"
)
