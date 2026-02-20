package portfolio

import (
	"math"
	"math/rand"
	"sort"
)

// SimulatedPortfolio represents one random portfolio
type SimulatedPortfolio struct {
	Weights []float64 `json:"weights"`
	Return  float64   `json:"return"`
	Risk    float64   `json:"risk"`
	Sharpe  float64   `json:"sharpe"`
}

// FrontierPoint is a point on the efficient frontier line
type FrontierPoint struct {
	Return  float64   `json:"return"`
	Risk    float64   `json:"risk"`
	Weights []float64 `json:"weights"`
}

// OptimizationResult holds everything returned to the client
type OptimizationResult struct {
	MonteCarloPoints []SimulatedPortfolio `json:"monte_carlo_points"`
	FrontierPoints   []FrontierPoint      `json:"frontier_points"`
	MaxSharpe        SimulatedPortfolio   `json:"max_sharpe"`
	MinVariance      SimulatedPortfolio   `json:"min_variance"`
}

// randomWeights generates n random weights that sum to 1
func randomWeights(n int, rng *rand.Rand) []float64 {
	w := make([]float64, n)
	sum := 0.0
	for i := range w {
		w[i] = rng.ExpFloat64()
		sum += w[i]
	}
	for i := range w {
		w[i] /= sum
	}
	return w
}

// RunMonteCarlo simulates numSims random portfolios and finds max Sharpe and min variance
func RunMonteCarlo(meanReturns []float64, covMatrix [][]float64, numSims int, riskFreeRate float64) OptimizationResult {
	rng := rand.New(rand.NewSource(42))
	n := len(meanReturns)

	sims := make([]SimulatedPortfolio, 0, numSims)
	maxSharpe := SimulatedPortfolio{Sharpe: math.Inf(-1)}
	minVar := SimulatedPortfolio{Risk: math.Inf(1)}

	for s := 0; s < numSims; s++ {
		w := randomWeights(n, rng)
		ret, vol, sharpe := PortfolioStats(w, meanReturns, covMatrix, riskFreeRate)

		sp := SimulatedPortfolio{
			Weights: w,
			Return:  ret,
			Risk:    vol,
			Sharpe:  sharpe,
		}
		sims = append(sims, sp)

		if sharpe > maxSharpe.Sharpe {
			maxSharpe = sp
		}
		if vol < minVar.Risk {
			minVar = sp
		}
	}

	frontier := computeFrontierLineFromSimulations(sims, 60)

	return OptimizationResult{
		MonteCarloPoints: sims,
		FrontierPoints:   frontier,
		MaxSharpe:        maxSharpe,
		MinVariance:      minVar,
	}
}

// computeFrontierLineFromSimulations computes an efficient frontier approximation
// from long-only Monte Carlo portfolios by keeping only non-dominated points
// (highest return seen so far when scanning from low to high risk).
func computeFrontierLineFromSimulations(sims []SimulatedPortfolio, maxPoints int) []FrontierPoint {
	if len(sims) == 0 {
		return nil
	}

	sorted := append([]SimulatedPortfolio(nil), sims...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Risk == sorted[j].Risk {
			return sorted[i].Return < sorted[j].Return
		}
		return sorted[i].Risk < sorted[j].Risk
	})

	efficient := make([]FrontierPoint, 0, len(sorted))
	bestReturn := math.Inf(-1)
	for _, p := range sorted {
		if p.Return > bestReturn {
			efficient = append(efficient, FrontierPoint{
				Return:  p.Return,
				Risk:    p.Risk,
				Weights: p.Weights,
			})
			bestReturn = p.Return
		}
	}

	if len(efficient) <= maxPoints {
		return efficient
	}

	step := float64(len(efficient)-1) / float64(maxPoints-1)
	resampled := make([]FrontierPoint, 0, maxPoints)
	for i := 0; i < maxPoints; i++ {
		idx := int(math.Round(float64(i) * step))
		if idx >= len(efficient) {
			idx = len(efficient) - 1
		}
		resampled = append(resampled, efficient[idx])
	}

	return resampled
}
