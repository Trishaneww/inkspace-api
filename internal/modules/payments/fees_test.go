package payments

import "testing"

func TestComputeFeeRateByTier(t *testing.T) {
	const amount = 10000 // $100.00

	standard := computeFee(amount, "artist", false)
	if standard.PlatformFeeCents != 600 {
		t.Errorf("standard fee = %d cents, want 600 (6%%)", standard.PlatformFeeCents)
	}

	premium := computeFee(amount, "artist", true)
	if premium.PlatformFeeCents != 500 {
		t.Errorf("premium fee = %d cents, want 500 (5%%)", premium.PlatformFeeCents)
	}
}

func TestComputeFeeMinimumFloor(t *testing.T) {
	// On a tiny charge, both tiers floor at PlatformFeeMinCents.
	fee := computeFee(1000, "artist", true)
	if fee.PlatformFeeCents != PlatformFeeMinCents {
		t.Errorf("premium fee on $10 = %d cents, want floor %d", fee.PlatformFeeCents, PlatformFeeMinCents)
	}
}

func TestComputeFeeClientChargeByPayer(t *testing.T) {
	const amount = 10000

	// Client pays the full premium fee on top of the base.
	client := computeFee(amount, "client", true)
	if client.ClientChargeCents != amount+500 {
		t.Errorf("client charge = %d, want %d", client.ClientChargeCents, amount+500)
	}

	// Artist absorbs the fee — client pays only the base.
	artist := computeFee(amount, "artist", true)
	if artist.ClientChargeCents != amount {
		t.Errorf("artist-payer client charge = %d, want %d", artist.ClientChargeCents, amount)
	}
}
