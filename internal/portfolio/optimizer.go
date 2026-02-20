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

	lr := 0.01
	const iters = 5000

	for iter := 0; iter < iters; iter++ {
		// 1. Gradient of portfolio variance: 2 * Cov * w
		grad := make([]float64, n)
		for i := 0; i < n; i++ {
			for j := 0; j < n; j++ {
				grad[i] += 2 * covMatrix[i][j] * w[j]
			}
		}

		// 2. Gradient step for variance
		for i := range w {
			w[i] -= lr * grad[i]
		}

		// 3. Project onto return constraint and sum(w)=1
		// This is a simplified projection. We iteratively adjust for sum and return.
		for p := 0; p < 10; p++ {
			// Enforce sum(w) = 1
			sumW := 0.0
			for _, wi := range w {
				sumW += wi
			}
			for i := range w {
				w[i] += (1.0 - sumW) / float64(n)
			}

			// Enforce return = targetRet
			currRet := 0.0
			for i, wi := range w {
				currRet += wi * meanReturns[i]
			}

			retDiff := targetRet - currRet
			if math.Abs(retDiff) < 1e-10 {
				break
			}

			// Adjust weights along the direction of meanReturns to satisfy return constraint
			// while trying to minimize impact on sum(w)=1
			meanMean := mean(meanReturns)
			sqDiffSum := 0.0
			for _, r := range meanReturns {
				d := r - meanMean
				sqDiffSum += d * d
			}

			if sqDiffSum > 1e-12 {
				for i := range w {
					w[i] += retDiff * (meanReturns[i] - meanMean) / sqDiffSum
				}
			}
		}

		// 4. Enforce non-negativity (w >= 0)
		for i := range w {
			if w[i] < 0 {
				w[i] = 0
			}
		}

		// Decay learning rate
		if iter%1000 == 999 {
			lr *= 0.5
		}
	}

	return w
}
