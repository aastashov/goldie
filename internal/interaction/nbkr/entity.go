package nbkr

import (
	"time"
)

// DateLayout is a date layout used in the NBKR.
const DateLayout = "02.01.2006"

// GoldPrice describes a gold price.
type GoldPrice struct {
	Date          time.Time // ex: 2025-11-07
	Weight        float64   // ex: 1.00
	PurchasePrice float64   // ex: 12526.00
	SellPrice     float64   // ex: 12588.50
}
