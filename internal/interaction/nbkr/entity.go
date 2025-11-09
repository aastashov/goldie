package nbkr

import (
	"time"
)

// DateLayout is a date layout used in the NBKR.
const DateLayout = "02.01.2006"

// GoldPrice describes a gold price.
type GoldPrice struct {
	Date          string  // ex: 07.11.2025
	Weight        float64 // ex: 1.00
	PurchasePrice float64 // ex: 12526.00
	SellPrice     float64 // ex: 12588.50
}

// GetDateTime returns a time.Time object from the Date field.
func (p *GoldPrice) GetDateTime() time.Time {
	date, _ := time.Parse(DateLayout, p.Date)
	return date
}
