package portfolio

import (
	"math"

	"frontier/internal/data"
)

// AssetStats holds annualized stats for one asset
type AssetStats struct {
	Ticker          string  `json:"ticker"`
	AnnualReturn    float64 `json:"annual_return"`
	AnnualVolatility float64 `json:"annual_volatility"`
}

// Returns computes log-returns from a price series
func Returns(prices []float64) []float64 {
	ret := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		ret[i-1] = math.Log(prices[i] / prices[i-1])
	}
	return ret
}

// mean computes the arithmetic mean of a slice
func mean(xs []float64) float64 {
	sum := 0.0
	for _, x := range xs {
		sum += x
	}
	return sum / float64(len(xs))
}

// PrepareAssets aligns return series across all assets using common length (shortest)
// and computes the annualized mean return vector and covariance matrix.
func PrepareAssets(priceData []*data.PriceData) (
	tickers []string,
	meanReturns []float64,
	covMatrix [][]float64,
	returnMatrix [][]float64,
	stats []AssetStats,
) {
	const tradingDays = 252.0

	// Build return series for each asset
	allReturns := make([][]float64, len(priceData))
	minLen := math.MaxInt32
	for i, pd := range priceData {
		r := Returns(pd.Closes)
		allReturns[i] = r
		if len(r) < minLen {
			minLen = len(r)
		}
	}

	// Trim all series to same length (use tail — most recent data)
	returnMatrix = make([][]float64, len(priceData))
	for i, r := range allReturns {
		returnMatrix[i] = r[len(r)-minLen:]
	}

	n := len(priceData)
	tickers = make([]string, n)
	meanReturns = make([]float64, n)
	stats = make([]AssetStats, n)

	for i, pd := range priceData {
		tickers[i] = pd.Ticker
		m := mean(returnMatrix[i])
		meanReturns[i] = m * tradingDays

		// Daily variance
		variance := 0.0
		for _, r := range returnMatrix[i] {
			d := r - m
			variance += d * d
		}
		variance /= float64(minLen - 1)

		stats[i] = AssetStats{
			Ticker:           pd.Ticker,
			AnnualReturn:     m * tradingDays,
			AnnualVolatility: math.Sqrt(variance * tradingDays),
		}
	}

	// Build covariance matrix (annualized)
	covMatrix = make([][]float64, n)
	for i := range covMatrix {
		covMatrix[i] = make([]float64, n)
	}
	for i := 0; i < n; i++ {
		mi := mean(returnMatrix[i])
		for j := i; j < n; j++ {
			mj := mean(returnMatrix[j])
			cov := 0.0
			for k := 0; k < minLen; k++ {
				cov += (returnMatrix[i][k] - mi) * (returnMatrix[j][k] - mj)
			}
			cov = cov / float64(minLen-1) * tradingDays
			covMatrix[i][j] = cov
			covMatrix[j][i] = cov
		}
	}

	return
}

// PortfolioStats computes annual return, volatility and Sharpe ratio for given weights
func PortfolioStats(weights, meanReturns []float64, covMatrix [][]float64, riskFreeRate float64) (ret, vol, sharpe float64) {
	n := len(weights)

	// Portfolio return
	for i := 0; i < n; i++ {
		ret += weights[i] * meanReturns[i]
	}

	// Portfolio variance = w^T * Cov * w
	variance := 0.0
	for i := 0; i < n; i++ {
		for j := 0; j < n; j++ {
			variance += weights[i] * weights[j] * covMatrix[i][j]
		}
	}
	vol = math.Sqrt(variance)
	if vol > 0 {
		sharpe = (ret - riskFreeRate) / vol
	}
	return
}
