// Package push sends Web Push notifications (VAPID) to a user's registered
// browsers — on iOS, to Nisaba installed on the Home Screen.
package push

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/farrellm/nisaba/internal/model"
)

// ttlSeconds is how long the push service holds an undelivered notification
// (e.g. while the phone is offline) before dropping it.
const ttlSeconds = 60 * 60

// ErrDisabled means the VAPID keys aren't configured.
var ErrDisabled = errors.New("push: not configured")

// Message is the notification payload the service worker (frontend/public/sw.js)
// shows; URL is the in-app path opened when it's tapped.
type Message struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	URL   string `json:"url"`
}

// Store is the subscription storage the notifier reads and prunes.
type Store interface {
	ListPushSubscriptions(ctx context.Context, userID int64) ([]model.PushSubscription, error)
	DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) error
}

// Notifier sends Messages to every subscription a user has registered. The
// zero value / nil is disabled: Notify is a no-op and Send reports ErrDisabled.
type Notifier struct {
	store      Store
	publicKey  string
	privateKey string
	subject    string
}

// NewNotifier returns a Notifier, or nil (disabled) unless both keys and the
// subject are set. subject is the VAPID contact: an https: URL or an e-mail
// address (with or without "mailto:"). Apple's push service rejects tokens
// whose subject isn't a real contact (403 BadJwtToken), so there's no default.
func NewNotifier(store Store, publicKey, privateKey, subject string) *Notifier {
	if publicKey == "" || privateKey == "" || subject == "" {
		return nil
	}
	// webpush-go prepends "mailto:" to anything that isn't an https: URL, so
	// pass a bare address to avoid sending "mailto:mailto:…".
	subject = strings.TrimPrefix(subject, "mailto:")
	return &Notifier{store: store, publicKey: publicKey, privateKey: privateKey, subject: subject}
}

// Enabled reports whether push is configured.
func (n *Notifier) Enabled() bool { return n != nil }

// PublicKey is the VAPID application server key browsers subscribe with.
func (n *Notifier) PublicKey() string {
	if n == nil {
		return ""
	}
	return n.publicKey
}

// Notify sends msg to all of userID's subscriptions, logging (not returning)
// failures: a notification must never fail the work that triggered it.
func (n *Notifier) Notify(ctx context.Context, userID int64, msg Message) {
	if n == nil {
		return
	}
	if _, err := n.Send(ctx, userID, msg); err != nil {
		slog.Error("push notify failed", "err", err, "user", userID)
	}
}

// Send delivers msg to all of userID's subscriptions and returns how many
// accepted it. Subscriptions the push service reports as gone (404/410) are
// deleted; other per-subscription failures are joined into the error.
func (n *Notifier) Send(ctx context.Context, userID int64, msg Message) (int, error) {
	if n == nil {
		return 0, ErrDisabled
	}
	subs, err := n.store.ListPushSubscriptions(ctx, userID)
	if err != nil {
		return 0, err
	}
	payload, err := json.Marshal(msg)
	if err != nil {
		return 0, err
	}

	sent := 0
	var errs []error
	for _, sub := range subs {
		ok, err := n.sendOne(ctx, payload, sub)
		if err != nil {
			errs = append(errs, err)
		}
		if ok {
			sent++
		}
	}
	return sent, errors.Join(errs...)
}

func (n *Notifier) sendOne(ctx context.Context, payload []byte, sub model.PushSubscription) (bool, error) {
	resp, err := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: sub.Endpoint,
		Keys:     webpush.Keys{P256dh: sub.Keys.P256dh, Auth: sub.Keys.Auth},
	}, &webpush.Options{
		Subscriber:      n.subject,
		VAPIDPublicKey:  n.publicKey,
		VAPIDPrivateKey: n.privateKey,
		TTL:             ttlSeconds,
		Urgency:         webpush.UrgencyNormal,
	})
	if err != nil {
		return false, fmt.Errorf("push to %s: %w", host(sub.Endpoint), err)
	}
	defer resp.Body.Close()
	// Push services explain rejections in the body (e.g. Apple's
	// {"reason":"BadJwtToken"}); keep a little of it for the error.
	reason, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	switch {
	case resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusGone:
		// The browser unsubscribed or the app was removed from the Home Screen.
		if err := n.store.DeletePushSubscriptionByEndpoint(ctx, sub.Endpoint); err != nil {
			return false, err
		}
		slog.Info("pruned expired push subscription", "service", host(sub.Endpoint))
		return false, nil
	case resp.StatusCode >= 300:
		return false, fmt.Errorf("push to %s: status %d: %s",
			host(sub.Endpoint), resp.StatusCode, strings.TrimSpace(string(reason)))
	}
	return true, nil
}

// host trims an endpoint to its push-service host for logs; the full URL is a
// bearer capability for that browser.
func host(endpoint string) string {
	for i := len("https://"); i < len(endpoint); i++ {
		if endpoint[i] == '/' {
			return endpoint[:i]
		}
	}
	return endpoint
}
