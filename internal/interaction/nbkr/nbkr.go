package nbkr

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Interaction struct {
	logger *slog.Logger
	client *http.Client
}

// NewInteraction creates a new instance of Interaction with NBKR.
func NewInteraction(logger *slog.Logger, client *http.Client) *Interaction {
	return &Interaction{
		logger: logger.With("component", "nbkr"),
		client: client,
	}
}

// GetGoldPrices returns a list of gold prices for the given period.
func (that *Interaction) GetGoldPrices(ctx context.Context, beginDate time.Time, endDate time.Time) ([]GoldPrice, error) {
	target := "https://www.nbkr.kg/index1.jsp?item=2747&lang=RUS"

	if !beginDate.IsZero() && !endDate.IsZero() {
		target += fmt.Sprintf("&begin_day=%02d&begin_month=%02d&begin_year=%d&end_day=%02d&end_month=%02d&end_year=%d", beginDate.Day(), beginDate.Month(), beginDate.Year(), endDate.Day(), endDate.Month(), endDate.Year())
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := that.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bad status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	return ParseGoldPrice(string(body))
}
