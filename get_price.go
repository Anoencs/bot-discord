package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// Add these constants for API keys
const (
	COINMARKETCAP_API_KEY = "22a271f5-b3ea-49af-8531-573675f15b23"
	COINAPI_API_KEY       = "your_coinapi_api_key"
)

// CoinMarketCap response structure
type CMCResponse struct {
	Data map[string]struct {
		Quote struct {
			USD struct {
				Price            float64 `json:"price"`
				PercentChange24H float64 `json:"percent_change_24h"`
				MarketCap        float64 `json:"market_cap"`
				Volume24H        float64 `json:"volume_24h"`
			} `json:"USD"`
		} `json:"quote"`
	} `json:"data"`
}
type BinanceTickerResponse struct {
	Symbol             string `json:"symbol"`
	Price              string `json:"lastPrice"`
	PriceChangePercent string `json:"priceChangePercent"`
	Volume             string `json:"volume"`
	QuoteVolume        string `json:"quoteVolume"` // 24h volume in USDT
}

func getPriceFromBinance(id string) (*CryptoPrice, error) {
	// Convert CoinGecko ID to Binance symbol
	binanceSymbol := convertToBinanceSymbol(id)
	if binanceSymbol == "" {
		return nil, fmt.Errorf("unsupported cryptocurrency for Binance: %s", id)
	}

	url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/24hr?symbol=%s", binanceSymbol)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("binance API error: status %d", resp.StatusCode)
	}

	var binanceResp BinanceTickerResponse
	if err := json.NewDecoder(resp.Body).Decode(&binanceResp); err != nil {
		return nil, err
	}

	// Convert string values to float64
	price, err := strconv.ParseFloat(binanceResp.Price, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing price: %v", err)
	}

	change, err := strconv.ParseFloat(binanceResp.PriceChangePercent, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing price change: %v", err)
	}

	volume, err := strconv.ParseFloat(binanceResp.QuoteVolume, 64)
	if err != nil {
		return nil, fmt.Errorf("error parsing volume: %v", err)
	}

	return &CryptoPrice{
		Price:     price,
		Change24h: change,
		Volume24h: volume,
		MarketCap: 0,
	}, nil
}

func getPriceFromCoinGecko(id string) (*CryptoPrice, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/simple/price?ids=%s&vs_currencies=usd&include_market_cap=true&include_24hr_vol=true&include_24hr_change=true", id)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	var data map[string]CryptoPrice
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	if price, ok := data[id]; ok {
		return &price, nil
	}

	return nil, fmt.Errorf("cryptocurrency '%s' not found", id)
}

func getPriceFromCoinMarketCap(id string) (*CryptoPrice, error) {
	cmcID := convertToCMCID(id)

	url := fmt.Sprintf("https://pro-api.coinmarketcap.com/v2/cryptocurrency/quotes/latest?slug=%s", cmcID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-CMC_PRO_API_KEY", COINMARKETCAP_API_KEY)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limit exceeded")
	}

	var cmcResp CMCResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmcResp); err != nil {
		return nil, err
	}

	// Convert CMC response to CryptoPrice
	for _, data := range cmcResp.Data {
		return &CryptoPrice{
			Price:     data.Quote.USD.Price,
			Change24h: data.Quote.USD.PercentChange24H,
			MarketCap: data.Quote.USD.MarketCap,
			Volume24h: data.Quote.USD.Volume24H,
		}, nil
	}

	return nil, fmt.Errorf("cryptocurrency '%s' not found", id)
}

func getCryptoPrice(id string) (*CryptoPrice, error) {
	// Try CoinGecko first
	price, err := getPriceFromCoinGecko(id)
	if err == nil {
		return price, nil
	}

	// If CoinGecko fails with rate limit, try CoinMarketCap
	if strings.Contains(err.Error(), "rate limit exceeded") {
		price, err = getPriceFromCoinMarketCap(id)
		if err == nil {
			return price, nil
		}
	}

	// If CoinMarketCap fails, try Binance
	if strings.Contains(err.Error(), "rate limit exceeded") {
		price, err = getPriceFromBinance(id)
		if err == nil {
			return price, nil
		}
	}

	// If all APIs fail, return the original error
	return nil, err
}
func formatNickname(symbol string, price float64) string {
	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		return fmt.Sprintf("%s $%.2f", strings.ToUpper(symbol), price)
	}

	if cryptoInfo.Symbol == "BTC" {
		return fmt.Sprintf("BTC $%.0f", price)
	}
	return fmt.Sprintf("%s $%.2f", cryptoInfo.Symbol, price)
}

func formatPrice(symbol string, price float64) string {
	cryptoInfo, exists := commonCryptos[symbol]
	if !exists {
		return fmt.Sprintf("$%.2f", price)
	}

	if cryptoInfo.Symbol == "BTC" {
		return fmt.Sprintf("$%.0f", price)
	}
	return fmt.Sprintf("$%.2f", price)
}
func getColorForChange(change float64) int {
	if change > 0 {
		return 0x00ff00 // Green
	}
	return 0xff0000 // Red
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
