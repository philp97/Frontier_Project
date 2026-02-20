package api

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
	"sync"

	"frontier/internal/data"
	"frontier/internal/portfolio"
)

// HealthHandler returns a simple health check
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// AnalyzeRequest is the JSON body for the analyze endpoint
type AnalyzeRequest struct {
	Tickers          []string           `json:"tickers"`
	Years            int                `json:"years"`
	RiskFreeRate     *float64           `json:"risk_free_rate"`
	CurrentPortfolio map[string]float64 `json:"current_portfolio"`
}

// AnalyzeResponse is the full JSON response
type AnalyzeResponse struct {
	Tickers               []string                       `json:"tickers"`
	AssetStats            []portfolio.AssetStats         `json:"asset_stats"`
	MonteCarloPoints      []portfolio.SimulatedPortfolio `json:"monte_carlo_points"`
	FrontierPoints        []portfolio.FrontierPoint      `json:"frontier_points"`
	MaxSharpe             portfolio.SimulatedPortfolio   `json:"max_sharpe"`
	MinVariance           portfolio.SimulatedPortfolio   `json:"min_variance"`
	CurrentPortfolioStats *portfolio.SimulatedPortfolio  `json:"current_portfolio_stats,omitempty"`
	Warnings              []string                       `json:"warnings,omitempty"`
	Error                 string                         `json:"error,omitempty"`
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(AnalyzeResponse{Error: msg})
}

// AnalyzeHandler handles POST /api/analyze
func AnalyzeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "only POST is supported")
		return
	}

	var req AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}

	// Sanitize tickers
	if len(req.Tickers) < 2 {
		writeError(w, http.StatusBadRequest, "please provide at least 2 tickers to compute a frontier")
		return
	}
	if len(req.Tickers) > 20 {
		writeError(w, http.StatusBadRequest, "maximum 20 tickers allowed")
		return
	}

	// Validate years (default 2, must be integer >= 1)
	years := req.Years
	if years < 1 {
		years = 2
	}
	if years > 100 {
		writeError(w, http.StatusBadRequest, "maximum 100 years of historical data allowed")
		return
	}

	// Deduplicate and uppercase tickers
	seen := map[string]bool{}
	var tickers []string
	for _, t := range req.Tickers {
		up := strings.ToUpper(strings.TrimSpace(t))
		if up != "" && !seen[up] {
			seen[up] = true
			tickers = append(tickers, up)
		}
	}

	// Fetch data concurrently
	type fetchResult struct {
		pd  *data.PriceData
		err error
		idx int
	}

	results := make([]fetchResult, len(tickers))
	var wg sync.WaitGroup

	for i, ticker := range tickers {
		wg.Add(1)
		go func(i int, ticker string) {
			defer wg.Done()
			pd, err := data.FetchPrices(ticker, years)
			results[i] = fetchResult{pd: pd, err: err, idx: i}
		}(i, ticker)
	}
	wg.Wait()

	// Check for fetch errors and partial data
	var errMsgs []string
	var warnings []string
	var priceData []*data.PriceData
	var validTickers []string
	for _, r := range results {
		if r.err != nil {
			errMsgs = append(errMsgs, fmt.Sprintf("%s: %v", tickers[r.idx], r.err))
			log.Printf("fetch error: %v", r.err)
		} else {
			if r.pd.Partial {
				warnings = append(warnings, fmt.Sprintf(
					"%s: only ~%.1f years of data available (requested %d years) — using available data",
					r.pd.Ticker, r.pd.YearsAvail, r.pd.YearsRequested,
				))
				log.Printf("partial data: %s has %.1f years, requested %d", r.pd.Ticker, r.pd.YearsAvail, r.pd.YearsRequested)
			}
			priceData = append(priceData, r.pd)
			validTickers = append(validTickers, tickers[r.idx])
		}
	}

	if len(priceData) < 2 {
		msg := "could not fetch enough data to compute the frontier"
		if len(errMsgs) > 0 {
			msg += ": " + strings.Join(errMsgs, "; ")
		}
		writeError(w, http.StatusBadRequest, msg)
		return
	}

	// Compute portfolio math
	_, meanReturns, covMatrix, _, assetStats := portfolio.PrepareAssets(priceData)

	// Risk-free rate: default 4.5%
	riskFreeRate := 0.045
	if req.RiskFreeRate != nil && *req.RiskFreeRate >= 0 && *req.RiskFreeRate <= 1 {
		riskFreeRate = *req.RiskFreeRate
	}

	// Run Monte Carlo + frontier
	result := portfolio.RunMonteCarlo(meanReturns, covMatrix, 10000, riskFreeRate)

	resp := AnalyzeResponse{
		Tickers:          validTickers,
		AssetStats:       assetStats,
		MonteCarloPoints: result.MonteCarloPoints,
		FrontierPoints:   result.FrontierPoints,
		MaxSharpe:        result.MaxSharpe,
		MinVariance:      result.MinVariance,
		Warnings:         warnings,
	}

	// Warn about any failed tickers
	if len(errMsgs) > 0 {
		resp.Error = "some tickers failed: " + strings.Join(errMsgs, "; ")
	}

	// Optional: current portfolio comparison
	if len(req.CurrentPortfolio) > 0 {
		weights := make([]float64, len(validTickers))
		sum := 0.0
		for i, t := range validTickers {
			w := req.CurrentPortfolio[t]
			weights[i] = w
			sum += w
		}
		if sum > 0 {
			// Normalize
			for i := range weights {
				weights[i] /= sum
			}
			ret, vol, sharpe := portfolio.PortfolioStats(weights, meanReturns, covMatrix, riskFreeRate)
			if !math.IsNaN(ret) && !math.IsNaN(vol) {
				resp.CurrentPortfolioStats = &portfolio.SimulatedPortfolio{
					Weights: weights,
					Return:  ret,
					Risk:    vol,
					Sharpe:  sharpe,
				}
			}
		}
	}

	json.NewEncoder(w).Encode(resp)
}
