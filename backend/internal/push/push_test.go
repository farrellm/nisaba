package push

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/SherClockHolmes/webpush-go"

	"github.com/farrellm/nisaba/internal/model"
)

type fakeStore struct {
	subs    []model.PushSubscription
	deleted []string
}

func (f *fakeStore) ListPushSubscriptions(context.Context, int64) ([]model.PushSubscription, error) {
	return f.subs, nil
}

func (f *fakeStore) DeletePushSubscriptionByEndpoint(_ context.Context, endpoint string) error {
	f.deleted = append(f.deleted, endpoint)
	return nil
}

// browserSub builds a subscription with real client keys, as a browser would.
func browserSub(t *testing.T, endpoint string) model.PushSubscription {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	secret := make([]byte, 16)
	_, _ = rand.Read(secret)
	var sub model.PushSubscription
	sub.Endpoint = endpoint
	sub.Keys.P256dh = base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes())
	sub.Keys.Auth = base64.RawURLEncoding.EncodeToString(secret)
	return sub
}

func TestSendDeliversAndPrunesGone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			if r.Header.Get("Authorization") == "" || r.Header.Get("TTL") == "" {
				t.Errorf("missing VAPID/TTL headers: %v", r.Header)
			}
			w.WriteHeader(http.StatusCreated)
		case "/gone":
			w.WriteHeader(http.StatusGone)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		t.Fatal(err)
	}
	st := &fakeStore{subs: []model.PushSubscription{
		browserSub(t, srv.URL+"/ok"),
		browserSub(t, srv.URL+"/gone"),
		browserSub(t, srv.URL+"/broken"),
	}}
	n := NewNotifier(st, pub, priv, "mailto:test@example.com")

	sent, err := n.Send(t.Context(), 1, Message{Title: "t", Body: "b", URL: "/"})
	if sent != 1 {
		t.Errorf("sent = %d, want 1", sent)
	}
	if err == nil {
		t.Error("want an error for the 500 endpoint")
	}
	if len(st.deleted) != 1 || st.deleted[0] != srv.URL+"/gone" {
		t.Errorf("deleted = %v, want only the 410 endpoint", st.deleted)
	}
}

func TestSubjectNormalized(t *testing.T) {
	if got := NewNotifier(nil, "pub", "priv", "mailto:me@example.com").subject; got != "me@example.com" {
		t.Errorf("subject = %q, want the bare address (webpush-go adds mailto:)", got)
	}
	if got := NewNotifier(nil, "pub", "priv", "https://example.com").subject; got != "https://example.com" {
		t.Errorf("subject = %q", got)
	}
	if NewNotifier(nil, "pub", "priv", "").Enabled() {
		t.Error("notifier without a subject should be disabled")
	}
}

func TestDisabledNotifier(t *testing.T) {
	n := NewNotifier(&fakeStore{}, "", "", "")
	if n.Enabled() {
		t.Fatal("notifier without keys should be disabled")
	}
	n.Notify(t.Context(), 1, Message{}) // must not panic
	if _, err := n.Send(t.Context(), 1, Message{}); !errors.Is(err, ErrDisabled) {
		t.Errorf("err = %v, want ErrDisabled", err)
	}
}
