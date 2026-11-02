package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/farrellm/nisaba/internal/auth"
	"github.com/farrellm/nisaba/internal/model"
	"github.com/farrellm/nisaba/internal/push"
)

// PushStore is the subscription storage the push handlers need.
type PushStore interface {
	UpsertPushSubscription(ctx context.Context, userID int64, sub model.PushSubscription) error
	DeletePushSubscription(ctx context.Context, userID int64, endpoint string) error
}

const pushDisabledMsg = "Push notifications are not configured"

// PushPublicKey returns the VAPID application server key browsers subscribe
// with; 503 when push isn't configured.
func PushPublicKey(n *push.Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if !n.Enabled() {
			writeError(w, http.StatusServiceUnavailable, pushDisabledMsg)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"publicKey": n.PublicKey()})
	}
}

// SubscribePush registers the caller's browser (a PushSubscription.toJSON()
// body) to receive run notifications.
func SubscribePush(st PushStore, n *push.Notifier) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFrom(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		if !n.Enabled() {
			writeError(w, http.StatusServiceUnavailable, pushDisabledMsg)
			return
		}
		var sub model.PushSubscription
		if err := decodeJSON(r, &sub); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}
		// The server POSTs to this URL, so accept only an https push endpoint.
		if !strings.HasPrefix(sub.Endpoint, "https://") || sub.Keys.P256dh == "" || sub.Keys.Auth == "" {
			writeError(w, http.StatusBadRequest, "Invalid push subscription")
			return
		}
		if err := st.UpsertPushSubscription(r.Context(), userID, sub); err != nil {
			internalError(w, r, "Failed to save push subscription", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

// UnsubscribePush removes the caller's subscription for a {endpoint} body.
func UnsubscribePush(st PushStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID, ok := auth.UserIDFrom(r.Context())
		if !ok {
			writeError(w, http.StatusUnauthorized, "Not logged in")
			return
		}
		var body struct {
			Endpoint string `json:"endpoint"`
		}
		if err := decodeJSON(r, &body); err != nil || body.Endpoint == "" {
			writeError(w, http.StatusBadRequest, "endpoint is required")
			return
		}
		if err := st.DeletePushSubscription(r.Context(), userID, body.Endpoint); err != nil {
			internalError(w, r, "Failed to delete push subscription", err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}
