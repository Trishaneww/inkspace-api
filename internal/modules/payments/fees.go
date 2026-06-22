package payments

import (
	"math"
	"time"
)

const (
	StandardPlatformFeePercent       = 0.06
	PremiumPlatformFeePercent        = 0.05
	PlatformFeeMinCents        int64 = 200
	MinChargeCents             int64 = 1000
	PayLinkTTL                       = 7 * 24 * time.Hour
	ResendCooldown                   = 15 * time.Minute
)

type FeeBreakdown struct {
	AmountCents       int64
	PlatformFeeCents  int64
	ClientChargeCents int64
}

func computeFee(amountCents int64, feePayer string, premium bool) FeeBreakdown {
	rate := StandardPlatformFeePercent
	if premium {
		rate = PremiumPlatformFeePercent
	}
	fee := max(int64(math.Round(float64(amountCents)*rate)), PlatformFeeMinCents)

	var clientCharge int64
	switch feePayer {
	case feePayerClient:
		clientCharge = amountCents + fee
	case feePayerSplit:
		clientCharge = amountCents + fee/2
	default: // feePayerArtist — client pays only the base; fee comes out of payout.
		clientCharge = amountCents
	}

	return FeeBreakdown{
		AmountCents:       amountCents,
		PlatformFeeCents:  fee,
		ClientChargeCents: clientCharge,
	}
}
