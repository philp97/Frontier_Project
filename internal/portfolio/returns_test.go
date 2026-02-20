package portfolio

import (
	"math"
	"testing"

	"frontier/internal/data"
)

func TestReturns(t *testing.T) {
	prices := []float64{100, 110, 121}
	expected := []float64{math.Log(1.1), math.Log(1.1)}

	got := Returns(prices)
	if len(got) != len(expected) {
		t.Fatalf("expected length %d, got %d", len(expected), len(got))
	}

	for i := range got {
		if math.Abs(got[i]-expected[i]) > 1e-9 {
			t.Errorf("at index %d: expected %f, got %f", i, expected[i], got[i])
		}
	}
}

func TestPrepareAssets(t *testing.T) {
	// 3 days of prices for 2 assets
	pd1 := &data.PriceData{
		Ticker: "A",
		Closes: []float64{100, 105, 110}, // ~5% then ~4.7%
	}
	pd2 := &data.PriceData{
		Ticker: "B",
		Closes: []float64{100, 110, 121}, // 10% then 10%
	}

	tickers, meanReturns, covMatrix, _, stats := PrepareAssets([]*data.PriceData{pd1, pd2})

	if len(tickers) != 2 {
		t.Errorf("expected 2 tickers, got %d", len(tickers))
	}

	// Check cov matrix symmetry
	if math.Abs(covMatrix[0][1]-covMatrix[1][0]) > 1e-12 {
		t.Errorf("cov matrix not symmetric: %f != %f", covMatrix[0][1], covMatrix[1][0])
	}

	// Check annualization
	// log(110/100) = 0.09531. Mean daily = 0.09531. Annual = 0.09531 * 252 = 24.
	expectedB := math.Log(1.21) / 2 * 252
	if math.Abs(meanReturns[1]-expectedB) > 1e-6 {
		t.Errorf("expected B return %f, got %f", expectedB, meanReturns[1])
	}

	if stats[1].AnnualReturn != meanReturns[1] {
		t.Errorf("stats vs meanReturns mismatch for B")
	}
}
