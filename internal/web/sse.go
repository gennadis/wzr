package web

import (
	"encoding/json"
	"fmt"
	"net/http"

	"wzr/internal/notify"
)

// writeSSEEvent serializes event as a JSON SSE data frame and flushes.
func writeSSEEvent(w http.ResponseWriter, flusher http.Flusher, event notify.StepEvent) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "data: %s\n\n", data)
	flusher.Flush()
}

// sseHeaders sets the required headers for an SSE response.
func sseHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}
