package subscriptions

import "time"

type CheckoutResponse struct {
	URL string `json:"url"`
}

type PortalResponse struct {
	URL string `json:"url"`
}

type SubscriptionStatus struct {
	Status            string     `json:"status"`
	IsPremium         bool       `json:"isPremium"`
	CancelAtPeriodEnd bool       `json:"cancelAtPeriodEnd"`
	CurrentPeriodEnd  *time.Time `json:"currentPeriodEnd"`
}
