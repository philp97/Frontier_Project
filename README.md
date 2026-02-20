# ◈ Frontier — Portfolio Optimizer

A web application that computes the **Markowitz Efficient Frontier** for any combination of stocks and ETFs. Built with **Go** (backend) and vanilla **HTML/CSS/JavaScript** (frontend).

Enter your tickers, and the app fetches real market data from Yahoo Finance, runs a Monte Carlo simulation with 10,000 portfolios, and shows you the optimal asset allocation — all in your browser.

---

## ✨ Features

- **Portfolio Visualization** — Interactive scatter plot showing simulated portfolios colored by Sharpe ratio
- **Optimal Portfolios** — Identifies the **Maximum Sharpe Ratio** and **Minimum Variance** portfolios with exact weight breakdowns
- **Portfolio Comparison** — Optionally enter your current allocation to see how it stacks up against the optimized portfolios
- **Configurable Parameters** — Choose any number of historical years and set your own risk-free rate
- **Partial Data Handling** — When a ticker doesn't have enough history, the app warns you and uses whatever data is available
- **Real Market Data** — Fetches daily close prices from Yahoo Finance (no API key required)
- **Supports up to 20 assets** — Stocks, ETFs, or any Yahoo Finance ticker

---

## 📐 How It Works

The app implements **Modern Portfolio Theory (MPT)** by Harry Markowitz:

1. **Data Fetching** — Downloads historical daily close prices for each ticker from Yahoo Finance
2. **Log Returns** — Computes daily log-returns from the price series
3. **Covariance Matrix** — Builds an annualized covariance matrix across all assets
4. **Monte Carlo Simulation** — Generates 10,000 random portfolio weight combinations and calculates each portfolio's return, risk (volatility), and Sharpe ratio
5. **Efficient Frontier** — Uses projected gradient descent to trace the minimum-variance boundary curve
6. **Optimization** — Identifies the portfolio with the highest Sharpe ratio (best risk-adjusted return) and the one with the lowest variance (safest)

---

## 📁 Project Structure

```
Frontier_Project/
├── main.go                          # Entry point — HTTP server on :8080
├── go.mod                           # Go module definition
├── internal/
│   ├── api/
│   │   └── handlers.go              # REST API handlers (/api/health, /api/analyze)
│   ├── data/
│   │   └── fetcher.go               # Yahoo Finance price data fetcher
│   └── portfolio/
│       ├── optimizer.go             # Monte Carlo simulation & frontier computation
│       └── returns.go               # Return calculations, covariance matrix, portfolio stats
└── static/
    ├── index.html                   # Main UI page
    ├── css/
    │   └── style.css                # Dark theme dashboard styling
    └── js/
        └── app.js                   # Frontend logic & Chart.js visualization
```

---

## 🚀 Setup & Run

### Prerequisites

- **Go 1.21+** — [Install Go](https://go.dev/doc/install)

### Clone and Run

```bash
# Clone the repository
git clone https://github.com/philp97/Frontier_Project.git
cd Frontier_Project

# Build and run
go run .
```

The server starts at **http://localhost:8080** — open it in your browser.

Alternatively, build a binary:

```bash
go build -o frontier .
./frontier
```

> **Note:** No external Go dependencies are used — only the standard library. No `go mod tidy` or dependency installation needed.

---

## 📖 How to Use

### 1. Add Tickers

Type a stock or ETF symbol (e.g., `AAPL`, `SPY`, `MSFT`, `QQQ`) into the input field and press **Enter** or click **Add**. You need at least 2 tickers and can add up to 20.

### 2. Set Historical Period

Enter the number of years of historical data to use (integer, default **2**). You can enter any value from 1 to 100.

- If a ticker doesn't have enough data for the requested period (e.g., it IPO'd 3 years ago but you asked for 10 years), the app will **warn you** and use whatever data is available
- If a ticker has no data at all, it will be excluded with an error message
- The analysis proceeds as long as at least 2 tickers have valid data

### 3. Set Risk-Free Rate (Optional)

The default is **4.5%** (approximate US Treasury rate). Adjust this to match your benchmark or local risk-free rate. The Sharpe ratio calculations use this value.

### 4. Compare Your Portfolio (Optional)

Toggle **"Compare my current portfolio"** and enter your current allocation percentages for each ticker. The app will show how your portfolio compares to the optimized ones.

### 5. Calculate

Click **⚡ Calculate Efficient Frontier**. The app will:

- Fetch price data from Yahoo Finance
- Run the Monte Carlo simulation
- Display results including:
  - **Stats cards** — Key metrics at a glance
  - **Efficient Frontier chart** — Interactive scatter plot with all simulated portfolios
  - **Optimal weights** — Exact percentage allocation for Max Sharpe and Min Variance portfolios
  - **Comparison table** — Side-by-side metrics (if portfolio comparison is enabled)
  - **Asset statistics** — Individual return, volatility, and Sharpe ratio for each ticker

---

## 🔌 API Reference

### `GET /api/health`

Health check endpoint.

**Response:**
```json
{"status": "ok"}
```

### `POST /api/analyze`

Run the portfolio optimization.

**Request body:**
```json
{
  "tickers": ["AAPL", "MSFT", "SPY", "BND"],
  "years": 2,
  "risk_free_rate": 0.045,
  "current_portfolio": {
    "AAPL": 40,
    "MSFT": 30,
    "SPY": 20,
    "BND": 10
  }
}
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `tickers` | string[] | Yes | 2–20 stock/ETF symbols |
| `years` | int | No | Number of years of history, 1–100 (default: `2`) |
| `risk_free_rate` | float | No | Annual rate as decimal (default: `0.045`) |
| `current_portfolio` | object | No | Current weights by ticker (in %) |

**Response:** JSON containing `asset_stats`, `monte_carlo_points`, `frontier_points`, `max_sharpe`, `min_variance`, optionally `current_portfolio_stats`, and `warnings` (array of partial-data messages).

---

## ⚠️ Disclaimer

This tool is for **educational purposes only** and does not constitute financial advice. Past performance does not guarantee future results. Always consult a qualified financial advisor before making investment decisions.

---

## 📄 License

MIT
