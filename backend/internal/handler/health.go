package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"
)

// pinger is the health check's view of the database pool.
type pinger interface {
	Ping(ctx context.Context) error
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
}

// Health reports liveness plus whether the database answers a ping.
func Health(pool pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dbStatus := "ok"
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			dbStatus = "error: " + err.Error()
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(healthResponse{ //nolint:errcheck
			Status: "ok",
			DB:     dbStatus,
		})
	}
}
