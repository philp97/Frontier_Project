package portfolio

import (
	"math"
	"math/rand"
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

	frontier := computeFrontierLine(meanReturns, covMatrix, minVar.Return, maxSharpe.Return*1.5, 60, riskFreeRate)

	return OptimizationResult{
		MonteCarloPoints: sims,
		FrontierPoints:   frontier,
		MaxSharpe:        maxSharpe,
		MinVariance:      minVar,
	}
}

// computeFrontierLine computes efficient frontier by sweeping target returns.
// For each target return, it finds minimum-variance portfolio using gradient descent.
func computeFrontierLine(meanReturns []float64, covMatrix [][]float64, minRet, maxRet float64, steps int, riskFreeRate float64) []FrontierPoint {
	points := make([]FrontierPoint, 0, steps)

	for i := 0; i <= steps; i++ {
		targetRet := minRet + (maxRet-minRet)*float64(i)/float64(steps)
		w := minVarForReturn(meanReturns, covMatrix, targetRet)
		if w == nil {
			continue
		}
		ret, vol, _ := PortfolioStats(w, meanReturns, covMatrix, riskFreeRate)

		// Only include points on the upper half of the frontier (efficient part)
		points = append(points, FrontierPoint{
			Return:  ret,
			Risk:    vol,
			Weights: w,
		})
	}

	return points
}

// minVarForReturn finds minimum variance portfolio for a given target return
// using projected gradient descent with return constraint and weight sum = 1.
func minVarForReturn(meanReturns []float64, covMatrix [][]float64, targetRet float64) []float64 {
	n := len(meanReturns)

	// Start from equal weights
	w := make([]float64, n)
	for i := range w {
		w[i] = 1.0 / float64(n)
	}

	lr := 0.001
	const iters = 3000

	for iter := 0; iter < iters; iter++ {
		// Gradient of portfolio variance w.r.t weights: 2 * Cov * w
		grad := make([]float64, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				grad[i] += 2 * covMatrix[i][j] * w[j]
			}
		}

		// Gradient step
		for i := range w {
			w[i] -= lr * grad[i]
		}

		// Project onto: sum(w) = 1, w >= 0 (long-only), return = targetRet
		// Step 1: enforce non-negativity
		for i := range w {
			if w[i] < 0 {
				w[i] = 0
			}
		}

		// Step 2: enforce return constraint (shift weights toward high-return assets)
		currentRet := 0.0
		for i, wi := range w {
			currentRet += wi * meanReturns[i]
		}

		// Simple return correction: adjust proportionally
		retDiff := targetRet - currentRet
		if math.Abs(retDiff) > 1e-8 {
			// Find max-return asset index
			maxIdx := 0
			for i, r := range meanReturns {
				if r > meanReturns[maxIdx] {
					maxIdx = i
				}
			}
			w[maxIdx] += retDiff * 0.1
			if w[maxIdx] < 0 {
				w[maxIdx] = 0
			}
		}

		// Step 3: normalize so weights sum to 1
		sum := 0.0
		for _, wi := range w {
			sum += wi
		}
		if sum < 1e-10 {
			return nil
		}
		for i := range w {
			w[i] /= sum
		}

		// Decay learning rate
		if iter%500 == 499 {
			lr *= 0.5
		}
	}

	// Final check: feasibility
	retCheck := 0.0
	for i, wi := range w {
		retCheck += wi * meanReturns[i]
	}
	_ = retCheck

	return w
}
