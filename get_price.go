package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
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
func getSantimentData(symbol string) (string, error) {
	apiURL := "https://api.santiment.net/graphql"

	// GraphQL query to fetch sentiment data
	query := fmt.Sprintf(`{
		"query": "query { 
			getMetric(metric: \"sentiment_positive\") { 
				timeseriesData(
					slug: \"%s\", 
					from: \"%s\", 
					to: \"%s\", 
					interval: \"1d\"
				) { 
					datetime 
					value 
				} 
			} 
		}"
	}`, symbol, time.Now().Add(-24*time.Hour).Format("2006-01-02T15:04:05Z"), time.Now().Format("2006-01-02T15:04:05Z"))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(query))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add necessary headers
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", SANTIMENT_API_KEY))
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response from API: %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Define a structure to parse response JSON
	type TimeseriesData struct {
		Datetime string  `json:"datetime"`
		Value    float64 `json:"value"`
	}
	type MetricData struct {
		TimeseriesData []TimeseriesData `json:"timeseriesData"`
	}
	type ResponseData struct {
		GetMetric MetricData `json:"getMetric"`
	}
	type APIResponse struct {
		Data ResponseData `json:"data"`
	}

	var apiResponse APIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Analyze sentiment data
	if len(apiResponse.Data.GetMetric.TimeseriesData) == 0 {
		return "No sentiment data available", nil
	}

	// Aggregate sentiment values
	var totalSentiment float64
	for _, data := range apiResponse.Data.GetMetric.TimeseriesData {
		totalSentiment += data.Value
	}
	averageSentiment := totalSentiment / float64(len(apiResponse.Data.GetMetric.TimeseriesData))

	// Classify sentiment
	switch {
	case averageSentiment > 0.75:
		return "Very Positive 🟢", nil
	case averageSentiment > 0.5:
		return "Positive ✅", nil
	case averageSentiment > 0.25:
		return "Neutral 🟡", nil
	case averageSentiment > 0:
		return "Negative 🔴", nil
	default:
		return "Very Negative ⚠️", nil
	}
}
func getSantimentDataForGreedFear(symbol string) (string, error) {
	apiURL := "https://api.santiment.net/graphql"

	// GraphQL query for sentiment and activity data
	query := fmt.Sprintf(`{
		"query": "query { 
			getMetric(metric: \"sentiment_positive\") { 
				timeseriesData(
					slug: \"%s\", 
					from: \"%s\", 
					to: \"%s\", 
					interval: \"1d\"
				) { 
					datetime 
					value 
				} 
			}
			getMetric(metric: \"sentiment_negative\") { 
				timeseriesData(
					slug: \"%s\", 
					from: \"%s\", 
					to: \"%s\", 
					interval: \"1d\"
				) { 
					datetime 
					value 
				} 
			}
			getMetric(metric: \"social_volume_total\") { 
				timeseriesData(
					slug: \"%s\", 
					from: \"%s\", 
					to: \"%s\", 
					interval: \"1d\"
				) { 
					datetime 
					value 
				} 
			}
		}"
	}`, symbol, time.Now().Add(-24*time.Hour).Format("2006-01-02T15:04:05Z"), time.Now().Format("2006-01-02T15:04:05Z"), symbol, time.Now().Add(-24*time.Hour).Format("2006-01-02T15:04:05Z"), time.Now().Format("2006-01-02T15:04:05Z"), symbol, time.Now().Add(-24*time.Hour).Format("2006-01-02T15:04:05Z"), time.Now().Format("2006-01-02T15:04:05Z"))

	req, err := http.NewRequest("POST", apiURL, strings.NewReader(query))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Add necessary headers
	req.Header.Add("Authorization", "Bearer YOUR_API_KEY")
	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("non-200 response from API: %d", resp.StatusCode)
	}

	// Read and parse the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	var apiResponse struct {
		Data struct {
			GetMetricPositive struct {
				TimeseriesData []struct {
					Datetime string  `json:"datetime"`
					Value    float64 `json:"value"`
				} `json:"timeseriesData"`
			} `json:"getMetric"`
			GetMetricNegative struct {
				TimeseriesData []struct {
					Datetime string  `json:"datetime"`
					Value    float64 `json:"value"`
				} `json:"timeseriesData"`
			} `json:"getMetric"`
			GetMetricVolume struct {
				TimeseriesData []struct {
					Datetime string  `json:"datetime"`
					Value    float64 `json:"value"`
				} `json:"timeseriesData"`
			} `json:"getMetric"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	// Aggregating data
	var totalPositive, totalNegative, totalVolume float64
	var countPositive, countNegative, countVolume int

	for _, data := range apiResponse.Data.GetMetricPositive.TimeseriesData {
		totalPositive += data.Value
		countPositive++
	}
	for _, data := range apiResponse.Data.GetMetricNegative.TimeseriesData {
		totalNegative += data.Value
		countNegative++
	}
	for _, data := range apiResponse.Data.GetMetricVolume.TimeseriesData {
		totalVolume += data.Value
		countVolume++
	}

	if countPositive == 0 || countNegative == 0 || countVolume == 0 {
		return "Insufficient data for Greed and Fear Index", nil
	}

	// Calculate average values
	avgPositive := totalPositive / float64(countPositive)
	avgNegative := totalNegative / float64(countNegative)
	avgVolume := totalVolume / float64(countVolume)

	// Compute Greed-Fear Index (simplified)
	greedScore := avgPositive + (avgVolume / 10)
	fearScore := avgNegative - (avgVolume / 10)

	// Classify index
	switch {
	case greedScore > fearScore*1.5:
		return "Market sentiment: Extreme Greed üü¢", nil
	case greedScore > fearScore:
		return "Market sentiment: Greed ‚úÖ", nil
	case fearScore > greedScore*1.5:
		return "Market sentiment: Extreme Fear üî¥", nil
	case fearScore > greedScore:
		return "Market sentiment: Fear ‚ö†Ô∏è", nil
	default:
		return "Market sentiment: Neutral üü°", nil
	}
}
