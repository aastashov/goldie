package suite

import (
	"testing"
	"time"
)

// GetDateTime returns a time.Time object from a string.
// Example: GetDateTime("2021-01-01 00:00:00")
func GetDateTime(t *testing.T, incomingDateTime string) time.Time {
	t.Helper()

	dateTime, err := time.Parse("2006-01-02", incomingDateTime)
	if err != nil {
		t.Fatalf("could not parse date time: %v", err)
	}
	return dateTime
}
