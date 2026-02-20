package data

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"time"
)

// PriceData holds the close price series for a ticker
type PriceData struct {
	Ticker    string
	Closes    []float64
	Dates     []time.Time
}

type yahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Symbol string `json:"symbol"`
			} `json:"meta"`
			Timestamp []int64 `json:"timestamp"`
			Indicators struct {
				Quote []struct {
					Close []interface{} `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
		Error *struct {
			Code        string `json:"code"`
			Description string `json:"description"`
		} `json:"error"`
	} `json:"chart"`
}

// FetchPrices downloads historical daily close prices from Yahoo Finance
// period: "1y", "2y", "5y"
func FetchPrices(ticker, period string) (*PriceData, error) {
	url := fmt.Sprintf(
		"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=%s",
		ticker, period,
	)

	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	// Yahoo requires a user-agent header
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; FrontierApp/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("network error fetching %s: %w", ticker, err)
	}
	defer resp.Body.Close()

	var yr yahooResponse
	if err := json.NewDecoder(resp.Body).Decode(&yr); err != nil {
		return nil, fmt.Errorf("failed to decode response for %s: %w", ticker, err)
	}

	if yr.Chart.Error != nil {
		return nil, fmt.Errorf("Yahoo Finance error for %s: %s", ticker, yr.Chart.Error.Description)
	}
	if len(yr.Chart.Result) == 0 {
		return nil, fmt.Errorf("no data returned for ticker %s — please check if it is valid", ticker)
	}

	result := yr.Chart.Result[0]
	quotes := result.Indicators.Quote
	if len(quotes) == 0 {
		return nil, fmt.Errorf("no quote data for %s", ticker)
	}

	closes := quotes[0].Close
	timestamps := result.Timestamp

	var prices []float64
	var dates []time.Time

	for i, c := range closes {
		if c == nil {
			continue
		}
		val, ok := c.(float64)
		if !ok || math.IsNaN(val) || val <= 0 {
			continue
		}
		prices = append(prices, val)
		if i < len(timestamps) {
			dates = append(dates, time.Unix(timestamps[i], 0))
		}
	}

	if len(prices) < 30 {
		return nil, fmt.Errorf("not enough price data for %s (got %d points, need at least 30)", ticker, len(prices))
	}

	return &PriceData{
		Ticker: result.Meta.Symbol,
		Closes: prices,
		Dates:  dates,
	}, nil
}
