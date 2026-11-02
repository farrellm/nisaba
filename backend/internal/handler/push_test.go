package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/farrellm/nisaba/internal/model"
	"github.com/farrellm/nisaba/internal/push"
)

type fakePushStore struct {
	upserted []model.PushSubscription
	userIDs  []int64
	deleted  []string
}

func (f *fakePushStore) UpsertPushSubscription(_ context.Context, userID int64, sub model.PushSubscription) error {
	f.userIDs = append(f.userIDs, userID)
	f.upserted = append(f.upserted, sub)
	return nil
}

func (f *fakePushStore) DeletePushSubscription(_ context.Context, userID int64, endpoint string) error {
	f.userIDs = append(f.userIDs, userID)
	f.deleted = append(f.deleted, endpoint)
	return nil
}

func testNotifier(st push.Store) *push.Notifier {
	return push.NewNotifier(st, "pub", "priv", "mailto:test@example.com")
}

func TestPushPublicKey(t *testing.T) {
	rec := httptest.NewRecorder()
	PushPublicKey(testNotifier(nil)).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "{\"publicKey\":\"pub\"}\n" {
		t.Errorf("got %d %q", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	PushPublicKey(nil).ServeHTTP(rec, httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("disabled status = %d, want 503", rec.Code)
	}
}

func TestSubscribePush(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"valid", `{"endpoint":"https://web.push.apple.com/abc","keys":{"p256dh":"k","auth":"a"}}`, http.StatusNoContent},
		{"http endpoint", `{"endpoint":"http://169.254.169.254/","keys":{"p256dh":"k","auth":"a"}}`, http.StatusBadRequest},
		{"missing keys", `{"endpoint":"https://web.push.apple.com/abc"}`, http.StatusBadRequest},
		{"bad json", `{`, http.StatusBadRequest},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			st := &fakePushStore{}
			rec := httptest.NewRecorder()
			req := asUser(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(tc.body)), 7)
			SubscribePush(st, testNotifier(nil)).ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tc.want, rec.Body.String())
			}
			if tc.want == http.StatusNoContent {
				if len(st.upserted) != 1 || st.userIDs[0] != 7 || st.upserted[0].Keys.Auth != "a" {
					t.Errorf("upserted = %+v for %v", st.upserted, st.userIDs)
				}
			} else if len(st.upserted) != 0 {
				t.Errorf("invalid subscription was saved: %+v", st.upserted)
			}
		})
	}
}

func TestSubscribePushDisabled(t *testing.T) {
	rec := httptest.NewRecorder()
	body := `{"endpoint":"https://web.push.apple.com/abc","keys":{"p256dh":"k","auth":"a"}}`
	req := asUser(httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/", strings.NewReader(body)), 7)
	SubscribePush(&fakePushStore{}, nil).ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("status = %d, want 503", rec.Code)
	}
}

func TestUnsubscribePush(t *testing.T) {
	st := &fakePushStore{}
	rec := httptest.NewRecorder()
	req := asUser(httptest.NewRequestWithContext(t.Context(), http.MethodDelete, "/", strings.NewReader(`{"endpoint":"https://x/1"}`)), 7)
	UnsubscribePush(st).ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent || len(st.deleted) != 1 || st.deleted[0] != "https://x/1" || st.userIDs[0] != 7 {
		t.Errorf("status %d, deleted %v for %v", rec.Code, st.deleted, st.userIDs)
	}
}
