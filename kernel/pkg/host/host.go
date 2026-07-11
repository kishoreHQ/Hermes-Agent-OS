// Package host defines the Host Interface (INV-11).
// Mission Control and all UIs are hosts. Kernel never assumes a product UI.
package host

import (
	"context"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/types"
)

type Interface interface {
	SubmitMission(ctx context.Context, m Mission) (types.MissionID, error)
	CancelMission(ctx context.Context, id types.MissionID, reason string) error
	SubscribeEvents(ctx context.Context, id types.MissionID) (<-chan Event, error)
	Health(ctx context.Context) error
}

type Mission struct {
	ID           types.MissionID      `json:"id"`
	Goal         string               `json:"goal"`
	RequiredCaps []types.Capability   `json:"requiredCapabilities"`
	Labels       map[string]string    `json:"labels,omitempty"`
}

type Event struct {
	Seq        int64            `json:"seq"`
	Type       string           `json:"type"`
	MissionID  types.MissionID  `json:"missionId"`
	Data       map[string]any   `json:"data,omitempty"`
}
