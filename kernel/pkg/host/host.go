// Package host defines the Host Interface (INV-11).
// Mission Control and all UIs are hosts. Kernel never assumes a product UI.
package host

import (
	"context"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

// MissionState is host-visible lifecycle state.
type MissionState string

const (
	StateQueued           MissionState = "queued"
	StateRunning          MissionState = "running"
	StateSucceeded        MissionState = "succeeded"
	StateFailed           MissionState = "failed"
	StateCancelled        MissionState = "cancelled"
	StateAwaitingApproval MissionState = "awaiting_approval"
)

// Interface is the sole host interaction surface (INV-11).
type Interface interface {
	SubmitMission(ctx context.Context, m Mission) (types.MissionID, error)
	CancelMission(ctx context.Context, id types.MissionID, reason string) error
	GetMission(ctx context.Context, id types.MissionID) (Mission, error)
	ListMissions(ctx context.Context, stateFilter string) ([]Mission, error)
	SubscribeEvents(ctx context.Context, id types.MissionID) (<-chan Event, error)
	EventsSince(ctx context.Context, since int64, missionFilter string) ([]Event, error)
	Replay(ctx context.Context, id types.MissionID) ([]Event, error)
	Health(ctx context.Context) error
}

// Mission is a host-submitted unit of work.
type Mission struct {
	ID           types.MissionID    `json:"id"`
	Name         string             `json:"name,omitempty"`
	Goal         string             `json:"goal"`
	State        MissionState       `json:"state"`
	Mode         types.AgentMode    `json:"mode,omitempty"`
	RequiredCaps []types.Capability `json:"requiredCapabilities"`
	Labels       map[string]string  `json:"labels,omitempty"`
	CostUSD      float64            `json:"costUsd,omitempty"`
	Output       string             `json:"output,omitempty"`
	ProviderID   types.PluginID     `json:"providerId,omitempty"`
	RuntimeID    types.PluginID     `json:"runtimeId,omitempty"`
	ModelID      string             `json:"modelId,omitempty"`
	RouteReason  string             `json:"routeReason,omitempty"`
	SecurityNote string             `json:"securityNote,omitempty"`
	CreatedAt    time.Time          `json:"createdAt"`
	UpdatedAt    time.Time          `json:"updatedAt"`
	CancelReason string             `json:"cancelReason,omitempty"`
}

// Event is a host-visible mission event (seq for reconnect).
type Event struct {
	Seq       int64           `json:"seq"`
	Type      string          `json:"type"`
	MissionID types.MissionID `json:"missionId"`
	TS        time.Time       `json:"ts"`
	Data      map[string]any  `json:"data,omitempty"`
}
