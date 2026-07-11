package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/kishoreHQ/Hermes-Agent-OS/kernel/pkg/eventbus"
	"nhooyr.io/websocket"
)

// apiEvents serves:
//   - GET with Accept: application/json or ?format=json → catch-up journal (since, mission)
//   - GET Upgrade: websocket → live stream with catch-up
func (s *Server) apiEvents(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeErr(w, 405, "method", "GET", "GET /api/v1/events")
		return
	}
	since, _ := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
	mission := r.URL.Query().Get("mission")

	// JSON catch-up (hosts that poll or integration tests)
	if r.URL.Query().Get("format") == "json" ||
		strings.Contains(r.Header.Get("Accept"), "application/json") &&
			!strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		evs, err := s.k.EventsSince(r.Context(), since, mission)
		if err != nil {
			writeErr(w, 500, "events_failed", err.Error(), "Retry.")
			return
		}
		writeOK(w, eventsJSON(evs))
		return
	}

	// WebSocket live + catch-up
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		return
	}
	defer c.Close(websocket.StatusNormalClosure, "")

	ctx := r.Context()

	// Catch-up from journal
	evs, err := s.k.EventsSince(ctx, since, mission)
	if err == nil {
		for _, e := range evs {
			if err := writeWSHostEvent(ctx, c, e.Seq, e.Type, string(e.MissionID), e.TS, e.Data); err != nil {
				return
			}
		}
	}

	// Live subscribe
	ch, err := s.k.Bus().Subscribe(ctx, mission)
	if err != nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-ch:
			if !ok {
				return
			}
			if err := writeWSBusEvent(ctx, c, e); err != nil {
				return
			}
		}
	}
}

func writeWSBusEvent(ctx context.Context, c *websocket.Conn, e eventbus.Event) error {
	return writeWSHostEvent(ctx, c, e.Seq, e.Type, string(e.MissionID), e.Time, e.Data)
}

func writeWSHostEvent(ctx context.Context, c *websocket.Conn, seq int64, typ, missionID string, ts time.Time, data map[string]any) error {
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	payload := map[string]any{
		"seq":       seq,
		"type":      typ,
		"ts":        ts.UTC().Format(time.RFC3339Nano),
		"missionId": missionID,
		"data":      data,
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, b)
}
