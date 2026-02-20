package portfolio

import (
	"math"
	"testing"
)

func TestPortfolioStats(t *testing.T) {
	meanReturns := []float64{0.1, 0.2}
	covMatrix := [][]float64{
		{0.04, 0.01},
		{0.01, 0.09},
	}
	weights := []float64{0.5, 0.5}
	rf := 0.05

	ret, vol, sharpe := PortfolioStats(weights, meanReturns, covMatrix, rf)

	expectedRet := 0.15
	if math.Abs(ret-expectedRet) > 1e-10 {
		t.Errorf("expected return %f, got %f", expectedRet, ret)
	}

	// var = 0.5^2 * 0.04 + 0.5^2 * 0.09 + 2 * 0.5 * 0.5 * 0.01
	// var = 0.25 * 0.04 + 0.25 * 0.09 + 0.5 * 0.01
	// var = 0.01 + 0.0225 + 0.005 = 0.0375
	expectedVol := math.Sqrt(0.0375)
	if math.Abs(vol-expectedVol) > 1e-10 {
		t.Errorf("expected vol %f, got %f", expectedVol, vol)
	}

	expectedSharpe := (0.15 - 0.05) / expectedVol
	if math.Abs(sharpe-expectedSharpe) > 1e-10 {
		t.Errorf("expected sharpe %f, got %f", expectedSharpe, sharpe)
	}
}

func TestMinVarForReturn(t *testing.T) {
	// Simple two asset case
	// A: ret 10%, vol 20%
	// B: ret 20%, vol 30%
	meanReturns := []float64{0.1, 0.2}
	covMatrix := [][]float64{
		{0.04, 0},
		{0, 0.09},
	}

	// Test target return exactly in middle (15%)
	w := minVarForReturn(meanReturns, covMatrix, 0.15)
	if w == nil {
		t.Fatal("got nil weights")
	}

	sum := 0.0
	ret := 0.0
	for i := range w {
		sum += w[i]
		ret += w[i] * meanReturns[i]
	}

	if math.Abs(sum-1.0) > 1e-3 {
		t.Errorf("weights do not sum to 1: %f", sum)
	}

	if math.Abs(ret-0.15) > 1e-3 {
		t.Errorf("return does not match target: got %f, want 0.15", ret)
	}
}

func TestRunMonteCarlo(t *testing.T) {
	meanReturns := []float64{0.1, 0.2}
	covMatrix := [][]float64{
		{0.04, 0},
		{0, 0.09},
	}

	res := RunMonteCarlo(meanReturns, covMatrix, 1000, 0.05)

	if len(res.MonteCarloPoints) != 1000 {
		t.Errorf("expected 1000 MC points, got %d", len(res.MonteCarloPoints))
	}

	// Max Sharpe should have higher Sharpe than any individual point (roughly)
	// and Min Var should have lower Risk than any point with significantly different return
	if res.MaxSharpe.Sharpe < 0 {
		t.Errorf("MaxSharpe ratio seems invalid: %f", res.MaxSharpe.Sharpe)
	}
}
