// Package subscription defines an artist's premium-tier entitlement, derived
package subscription

const (
	StatusNone              = "none"
	StatusIncomplete        = "incomplete"
	StatusIncompleteExpired = "incomplete_expired"
	StatusTrialing          = "trialing"
	StatusActive            = "active"
	StatusPastDue           = "past_due"
	StatusCanceled          = "canceled"
	StatusUnpaid            = "unpaid"
	StatusPaused            = "paused"
)

func IsPremium(subscriptionStatus string) bool {
	return subscriptionStatus == StatusActive
}
