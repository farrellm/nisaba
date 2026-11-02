package store

import (
	"context"

	"github.com/farrellm/nisaba/internal/model"
)

// UpsertPushSubscription stores a browser's push endpoint for userID. An
// endpoint already on file is re-keyed and reassigned to userID, since a
// browser that logs in as someone else should notify the new user.
func (s *Store) UpsertPushSubscription(ctx context.Context, userID int64, sub model.PushSubscription) (err error) {
	defer wrap(&err, "upsert push subscription")
	_, err = s.pool.Exec(ctx,
		`INSERT INTO push_subscriptions (user_id, endpoint, p256dh, auth)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (endpoint) DO UPDATE
		 SET user_id = EXCLUDED.user_id, p256dh = EXCLUDED.p256dh, auth = EXCLUDED.auth`,
		userID, sub.Endpoint, sub.Keys.P256dh, sub.Keys.Auth)
	return err
}

// DeletePushSubscription removes userID's subscription for endpoint. Deleting
// one that doesn't exist (or belongs to someone else) is a no-op.
func (s *Store) DeletePushSubscription(ctx context.Context, userID int64, endpoint string) (err error) {
	defer wrap(&err, "delete push subscription")
	_, err = s.pool.Exec(ctx,
		`DELETE FROM push_subscriptions WHERE user_id = $1 AND endpoint = $2`, userID, endpoint)
	return err
}

// DeletePushSubscriptionByEndpoint removes a subscription the push service
// reported as gone (404/410), whoever owns it.
func (s *Store) DeletePushSubscriptionByEndpoint(ctx context.Context, endpoint string) (err error) {
	defer wrap(&err, "delete push subscription by endpoint")
	_, err = s.pool.Exec(ctx, `DELETE FROM push_subscriptions WHERE endpoint = $1`, endpoint)
	return err
}

// ListPushSubscriptions returns every push endpoint registered for userID.
func (s *Store) ListPushSubscriptions(ctx context.Context, userID int64) (subs []model.PushSubscription, err error) {
	defer wrap(&err, "list push subscriptions")
	rows, err := s.pool.Query(ctx,
		`SELECT endpoint, p256dh, auth FROM push_subscriptions WHERE user_id = $1 ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	subs = []model.PushSubscription{}
	for rows.Next() {
		var sub model.PushSubscription
		if err := rows.Scan(&sub.Endpoint, &sub.Keys.P256dh, &sub.Keys.Auth); err != nil {
			return nil, err
		}
		subs = append(subs, sub)
	}
	return subs, rows.Err()
}
