package run

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"
)

// MarshalJSON renders the event slice as JSON by delegating to each concrete event.
func (events Events) MarshalJSON() ([]byte, error) {
	if events == nil {
		return []byte("[]"), nil
	}
	payload := make([]json.RawMessage, len(events))
	for i, evt := range events {
		if evt == nil {
			payload[i] = json.RawMessage("null")
			continue
		}
		data, err := json.Marshal(evt)
		if err != nil {
			return nil, err
		}
		payload[i] = data
	}
	return json.Marshal(payload)
}

// UnmarshalJSON reconstructs concrete event types from their serialized form.
func (events *Events) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*events = nil
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	decoded := make([]BaseRunOutputEvent, 0, len(raw))
	for _, item := range raw {
		if len(item) == 0 || string(item) == "null" {
			continue
		}
		evt, err := decodeEvent(item)
		if err != nil {
			return err
		}
		decoded = append(decoded, evt)
	}
	*events = decoded
	return nil
}

func decodeEvent(raw json.RawMessage) (BaseRunOutputEvent, error) {
	var meta struct {
		Event     string `json:"event"`
		EventType string `json:"event_type"`
	}
	_ = json.Unmarshal(raw, &meta)
	kind := strings.ToLower(strings.TrimSpace(meta.Event))
	if kind == "" {
		kind = strings.ToLower(strings.TrimSpace(meta.EventType))
	}
	switch {
	case kind == "" || strings.Contains(kind, "content"):
		var evt RunContentEvent
		if err := json.Unmarshal(raw, &evt); err != nil {
			return nil, err
		}
		if evt.eventBase.eventType == "" {
			evt.eventBase.eventType = EventTypeRunContent
		}
		return &evt, nil
	case strings.Contains(kind, "completed"):
		var evt RunCompletedEvent
		if err := json.Unmarshal(raw, &evt); err != nil {
			return nil, err
		}
		if evt.eventBase.eventType == "" {
			evt.eventBase.eventType = EventTypeRunCompleted
		}
		return &evt, nil
	default:
		var evt GenericRunEvent
		if err := json.Unmarshal(raw, &evt); err != nil {
			return nil, err
		}
		if evt.eventBase.eventType == "" {
			evt.eventBase.eventType = kind
		}
		return &evt, nil
	}
}

// MarshalJSON serializes RunContentEvent with canonical metadata fields.
func (e *RunContentEvent) MarshalJSON() ([]byte, error) {
	type alias RunContentEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeRunContent),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(e),
	}
	return json.Marshal(payload)
}

// UnmarshalJSON hydrates RunContentEvent from the serialized payload.
func (e *RunContentEvent) UnmarshalJSON(data []byte) error {
	type alias RunContentEvent
	aux := struct {
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	meta := map[string]interface{}{}
	if err := json.Unmarshal(data, &meta); err == nil {
		e.eventBase.eventType = extractEventKind(meta, EventTypeRunContent)
		e.eventBase.timestamp = extractTimestamp(meta)
	}
	return nil
}

// MarshalJSON serializes RunCompletedEvent with canonical metadata fields.
func (e *RunCompletedEvent) MarshalJSON() ([]byte, error) {
	type alias RunCompletedEvent
	payload := struct {
		Event     string `json:"event"`
		CreatedAt int64  `json:"created_at"`
		*alias
	}{
		Event:     canonicalEventType(e.eventBase.eventType, EventTypeRunCompleted),
		CreatedAt: unixSeconds(e.timestamp),
		alias:     (*alias)(e),
	}
	return json.Marshal(payload)
}

// UnmarshalJSON hydrates RunCompletedEvent from JSON.
func (e *RunCompletedEvent) UnmarshalJSON(data []byte) error {
	type alias RunCompletedEvent
	aux := struct {
		*alias
	}{alias: (*alias)(e)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	meta := map[string]interface{}{}
	if err := json.Unmarshal(data, &meta); err == nil {
		e.eventBase.eventType = extractEventKind(meta, EventTypeRunCompleted)
		e.eventBase.timestamp = extractTimestamp(meta)
	}
	return nil
}

// GenericRunEvent preserves unknown event payloads while implementing BaseRunOutputEvent.
type GenericRunEvent struct {
	eventBase
	fields map[string]interface{}
}

// MarshalJSON re-emits the original payload enriched with canonical metadata.
func (e *GenericRunEvent) MarshalJSON() ([]byte, error) {
	if e.fields == nil {
		e.fields = make(map[string]interface{})
	}
	if _, ok := e.fields["event"]; !ok && e.eventBase.eventType != "" {
		e.fields["event"] = e.eventBase.eventType
	}
	if _, ok := e.fields["created_at"]; !ok && !e.timestamp.IsZero() {
		e.fields["created_at"] = unixSeconds(e.timestamp)
	}
	return json.Marshal(e.fields)
}

// UnmarshalJSON stores the raw payload and extracts basic metadata.
func (e *GenericRunEvent) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &e.fields); err != nil {
		return err
	}
	e.eventBase.eventType = extractEventKind(e.fields, "")
	e.eventBase.timestamp = extractTimestamp(e.fields)
	return nil
}

func canonicalEventType(value, fallback string) string {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return fallback
	}
	return v
}

func unixSeconds(ts time.Time) int64 {
	if ts.IsZero() {
		return 0
	}
	return ts.UTC().Unix()
}

func extractEventKind(meta map[string]interface{}, fallback string) string {
	if meta == nil {
		return fallback
	}
	if evt, ok := meta["event"].(string); ok && strings.TrimSpace(evt) != "" {
		return strings.ToLower(strings.TrimSpace(evt))
	}
	if evt, ok := meta["event_type"].(string); ok && strings.TrimSpace(evt) != "" {
		return strings.ToLower(strings.TrimSpace(evt))
	}
	return fallback
}

func extractTimestamp(meta map[string]interface{}) time.Time {
	if meta == nil {
		return time.Time{}
	}
	if raw, ok := meta["created_at"]; ok {
		switch v := raw.(type) {
		case float64:
			return time.Unix(int64(v), 0).UTC()
		case string:
			if v == "" {
				return time.Time{}
			}
			if unix, err := strconv.ParseInt(v, 10, 64); err == nil {
				return time.Unix(unix, 0).UTC()
			}
			if ts, err := time.Parse(time.RFC3339, v); err == nil {
				return ts.UTC()
			}
		}
	}
	return time.Time{}
}
