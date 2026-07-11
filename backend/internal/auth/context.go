package auth

import "context"

// ctxKey is the private key type for values this package stores in a context,
// so no other package can collide with (or forge) them.
type ctxKey struct{}

// WithUserID returns a copy of ctx carrying the logged-in user's id.
func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, ctxKey{}, id)
}

// UserIDFrom returns the logged-in user id stored by WithUserID, or false when
// the context carries none (i.e. the request did not pass the auth middleware).
func UserIDFrom(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(ctxKey{}).(int64)
	return id, ok
}
