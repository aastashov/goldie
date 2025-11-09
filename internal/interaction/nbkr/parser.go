package nbkr

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func ParseGoldPrice(html string) ([]GoldPrice, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("parse html: %w", err)
	}

	var prices []GoldPrice

	doc.Find("table tbody tr").Each(func(_ int, tr *goquery.Selection) {
		tds := tr.Find("td")
		if tds.Length() < 4 {
			return
		}

		date := strings.TrimSpace(tds.Eq(0).Text())
		weightStr := cleanNumber(tds.Eq(1).Text())
		buyStr := cleanNumber(tds.Eq(2).Text())
		sellStr := cleanNumber(tds.Eq(3).Text())

		weight, _ := strconv.ParseFloat(weightStr, 64)
		buy, _ := strconv.ParseFloat(buyStr, 64)
		sell, _ := strconv.ParseFloat(sellStr, 64)

		if weight == 0 || buy == 0 || sell == 0 {
			return
		}

		prices = append(prices, GoldPrice{Date: date, Weight: weight, PurchasePrice: buy, SellPrice: sell})
	})

	sort.SliceStable(prices, func(i, j int) bool {
		di := prices[i].GetDateTime()
		dj := prices[j].GetDateTime()
		if di.Equal(dj) {
			return prices[i].Weight < prices[j].Weight
		}
		return di.Before(dj)
	})

	return prices, nil
}

func cleanNumber(s string) string {
	s = strings.ReplaceAll(s, " ", "")
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, ",", ".")
	return s
}
