package model

// PushSubscription is one browser's Web Push endpoint for a user, in the shape
// of the browser's PushSubscription.toJSON().
type PushSubscription struct {
	Endpoint string `json:"endpoint"`
	Keys     struct {
		P256dh string `json:"p256dh"`
		Auth   string `json:"auth"`
	} `json:"keys"`
}
