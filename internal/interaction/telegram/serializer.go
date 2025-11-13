package telegram

import (
	"fmt"
	"sort"
	"strings"

	"goldie/internal/model"
)

// PricesToString returns a string representation of the prices to send to the user.
func (that *Interaction) PricesToString(languageCode string, prices []*model.GoldPrice) string {
	sort.SliceStable(prices, func(i, j int) bool {
		if prices[i].Date.Equal(prices[j].Date) {
			return prices[i].Weight < prices[j].Weight
		}
		return prices[i].Date.After(prices[j].Date)
	})

	currentDate := prices[0].Date
	title, _ := that.renderLocaledMessage(languageCode, "goldPricesTitle", "Date", currentDate.Format("2006-01-02"))
	headerWeight, _ := that.renderLocaledMessage(languageCode, "columnWeight")
	headerBuy, _ := that.renderLocaledMessage(languageCode, "columnBuy")
	headerSell, _ := that.renderLocaledMessage(languageCode, "columnSell")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n<pre>\n", title))
	sb.WriteString(fmt.Sprintf("%-8s %-12s %-12s\n", headerWeight, headerBuy, headerSell))

	for _, p := range prices {
		if !p.Date.Equal(currentDate) {
			break
		}
		sb.WriteString(fmt.Sprintf("%-8.4g %-12.2f %-12.2f\n", p.Weight, p.PurchasePrice, p.SellPrice))
	}

	sb.WriteString("</pre>")
	return sb.String()
}

// PricesWithGainToString returns a string representation of the prices to send to the user.
func (that *Interaction) PricesWithGainToString(languageCode string, prices []*model.GoldPrice, buyingPrices []*model.GoldPrice) string {
	sort.SliceStable(prices, func(i, j int) bool {
		if prices[i].Date.Equal(prices[j].Date) {
			return prices[i].Weight < prices[j].Weight
		}
		return prices[i].Date.After(prices[j].Date)
	})

	currentDate := prices[0].Date
	title, _ := that.renderLocaledMessage(languageCode, "goldPricesTitle", "Date", currentDate.Format("2006-01-02"))
	headerWeight, _ := that.renderLocaledMessage(languageCode, "columnWeight")
	headerBuy, _ := that.renderLocaledMessage(languageCode, "columnBuy")
	headerSell, _ := that.renderLocaledMessage(languageCode, "columnSell")
	headerGain, _ := that.renderLocaledMessage(languageCode, "columnGain")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<b>%s</b>\n<pre>\n", title))
	sb.WriteString(fmt.Sprintf("%-8s %-12s %-12s %-12s\n", headerWeight, headerBuy, headerSell, headerGain))

	weightLookup := make(map[float64]*model.GoldPrice, len(prices))
	for _, bp := range buyingPrices {
		weightLookup[bp.Weight] = bp
	}

	for _, p := range prices {
		if !p.Date.Equal(currentDate) {
			break
		}

		bp := weightLookup[p.Weight]
		gain := p.SellPrice * 100 / bp.SellPrice
		sb.WriteString(fmt.Sprintf("%-8.4g %-12.2f %-12.2f %-12.2f\n", p.Weight, p.PurchasePrice, bp.SellPrice, gain))
	}

	sb.WriteString("</pre>")
	return sb.String()
}
