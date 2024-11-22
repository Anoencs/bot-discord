package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// Add this new struct for price history
type PriceHistory struct {
	Prices [][2]float64 `json:"prices"` // [[timestamp, price], ...]
}

// Add this function to get price history
func getCryptoPriceHistory(id string, days string) (*PriceHistory, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/%s/market_chart?vs_currency=usd&days=%s", id, days)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	}

	var data PriceHistory
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &data, nil
}
