package workflow

import (
	"fmt"
	"strings"

	"github.com/rexleimo/agno-go/pkg/hno/run"
)

func stepEventsKey(stepID string) string {
	return fmt.Sprintf("step_%s_events", stepID)
}

func aggregateEventContent(events run.Events) string {
	if len(events) == 0 {
		return ""
	}
	var builder strings.Builder
	for _, evt := range events {
		switch e := evt.(type) {
		case *run.RunContentEvent:
			if e.Content == "" {
				continue
			}
			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(e.Content)
		}
	}
	return builder.String()
}
