/* =====================================================
   app.js — Efficient Frontier Portfolio Optimizer
===================================================== */

'use strict';

// ---- State ----
let tickers = [];
let selectedPeriod = '2y';
let frontierChart = null;
let lastData = null;
let currentRiskFreeRate = 0.045;

// ---- Color palettes ----
const WEIGHT_COLORS = [
    '#10d9a0', '#38bdf8', '#f59e0b', '#a78bfa', '#f472b6',
    '#34d399', '#60a5fa', '#fbbf24', '#c084fc', '#fb7185',
    '#6ee7b7', '#93c5fd', '#fcd34d', '#d8b4fe', '#fda4af',
    '#5eead4', '#818cf8', '#fb923c', '#e879f9', '#4ade80',
];

// ---- Ticker management ----
function addTicker() {
    const input = document.getElementById('tickerInput');
    const val = input.value.trim().toUpperCase();
    if (!val) return;
    if (tickers.includes(val)) { input.value = ''; return; }
    if (tickers.length >= 20) { showError('Maximum 20 tickers allowed.'); return; }
    if (!/^[A-Z0-9.\-^]{1,10}$/.test(val)) { showError('Invalid ticker: ' + val); return; }

    tickers.push(val);
    input.value = '';
    renderChips();
    updateCurrentWeightsInputs();
    clearAlerts();
}

document.getElementById('tickerInput').addEventListener('keydown', (e) => {
    if (e.key === 'Enter' || e.key === ',') { e.preventDefault(); addTicker(); }
});

function removeTicker(t) {
    tickers = tickers.filter(x => x !== t);
    renderChips();
    updateCurrentWeightsInputs();
}

function renderChips() {
    const container = document.getElementById('tickerChips');
    const count = document.getElementById('tickerCount');
    container.innerHTML = tickers.map((t, i) =>
        `<div class="chip" style="border-color:${WEIGHT_COLORS[i % WEIGHT_COLORS.length]}33; color:${WEIGHT_COLORS[i % WEIGHT_COLORS.length]}">
      ${t}
      <button class="chip-remove" onclick="removeTicker('${t}')" title="Remove">✕</button>
    </div>`
    ).join('');
    count.textContent = `${tickers.length} / 20 assets`;
}

// ---- Period selection ----
function selectPeriod(btn) {
    document.querySelectorAll('.period-btn').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    selectedPeriod = btn.dataset.period;
}

// ---- Toggle compare section ----
function toggleCompare() {
    const on = document.getElementById('compareToggle').checked;
    document.getElementById('currentPortfolioSection').classList.toggle('hidden', !on);
}

function updateCurrentWeightsInputs() {
    const grid = document.getElementById('currentWeightsInputs');
    if (!document.getElementById('compareToggle').checked) return;
    grid.innerHTML = tickers.map(t =>
        `<div class="weight-field">
      <label>${t}</label>
      <input type="number" id="cw_${t}" min="0" max="100" step="0.1" placeholder="%" />
    </div>`
    ).join('');
}

document.getElementById('compareToggle').addEventListener('change', updateCurrentWeightsInputs);

// ---- Alerts ----
function showError(msg) {
    const el = document.getElementById('errorAlert');
    el.textContent = '⚠ ' + msg;
    el.classList.remove('hidden');
}
function showWarn(msg) {
    const el = document.getElementById('warnAlert');
    el.textContent = '⚡ ' + msg;
    el.classList.remove('hidden');
}
function clearAlerts() {
    document.getElementById('errorAlert').classList.add('hidden');
    document.getElementById('warnAlert').classList.add('hidden');
}

// ---- Main analyze function ----
async function analyze() {
    clearAlerts();

    if (tickers.length < 2) {
        showError('Please add at least 2 tickers to compute an efficient frontier.');
        return;
    }

    // Build current portfolio if enabled
    let currentPortfolio = null;
    if (document.getElementById('compareToggle').checked) {
        currentPortfolio = {};
        let sum = 0;
        for (const t of tickers) {
            const val = parseFloat(document.getElementById('cw_' + t)?.value) || 0;
            currentPortfolio[t] = val;
            sum += val;
        }
        if (sum <= 0) {
            showError('Please enter current portfolio weights (in %).');
            return;
        }
    }

    // Loading state
    setLoading(true);
    document.getElementById('results').classList.add('hidden');

    // Read risk-free rate
    const rfrInput = parseFloat(document.getElementById('riskFreeRateInput').value);
    const riskFreeRate = (!isNaN(rfrInput) && rfrInput >= 0 && rfrInput <= 100) ? rfrInput / 100 : 0.045;
    currentRiskFreeRate = riskFreeRate;

    const body = {
        tickers,
        period: selectedPeriod,
        risk_free_rate: riskFreeRate,
        current_portfolio: currentPortfolio || {},
    };

    try {
        const resp = await fetch('/api/analyze', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(body),
        });
        const data = await resp.json();

        if (data.error && (!data.frontier_points || data.frontier_points.length === 0)) {
            showError(data.error);
            setLoading(false);
            return;
        }
        if (data.error) {
            showWarn(data.error);
        }

        lastData = data;
        renderResults(data);
    } catch (err) {
        showError('Network error: ' + err.message + '. Is the server running?');
    } finally {
        setLoading(false);
    }
}

function setLoading(on) {
    const btn = document.getElementById('analyzeBtn');
    const txt = document.getElementById('analyzeBtnText');
    const sp = document.getElementById('spinner');
    btn.disabled = on;
    txt.textContent = on ? 'Calculating…' : '⚡ Calculate Efficient Frontier';
    sp.classList.toggle('hidden', !on);
}

// ---- Render all results ----
function renderResults(data) {
    renderStatCards(data);
    renderChart(data);
    renderWeights('maxSharpeWeights', data.tickers, data.max_sharpe.weights);
    renderWeights('minVarWeights', data.tickers, data.min_variance.weights);
    renderAssetTable(data);

    if (data.current_portfolio_stats) {
        renderComparison(data);
        document.getElementById('comparisonCard').classList.remove('hidden');
    } else {
        document.getElementById('comparisonCard').classList.add('hidden');
    }

    document.getElementById('results').classList.remove('hidden');
    document.getElementById('results').scrollIntoView({ behavior: 'smooth', block: 'start' });
}

// ---- Stats Cards ----
function renderStatCards(data) {
    const ms = data.max_sharpe;
    const mv = data.min_variance;
    const cards = [
        {
            label: 'Max Sharpe Return',
            value: pct(ms.return),
            sub: `Risk: ${pct(ms.risk)}`,
            cls: 'stat-green',
        },
        {
            label: 'Max Sharpe Ratio',
            value: ms.sharpe.toFixed(2),
            sub: 'Risk-adjusted performance',
            cls: 'stat-green',
        },
        {
            label: 'Min Variance Risk',
            value: pct(mv.risk),
            sub: `Return: ${pct(mv.return)}`,
            cls: 'stat-blue',
        },
        {
            label: 'Assets Analyzed',
            value: data.tickers.length,
            sub: `Period: ${selectedPeriod}`,
            cls: 'stat-amber',
        },
    ];

    document.getElementById('statsCards').innerHTML = cards.map(c =>
        `<div class="stat-card">
      <div class="stat-label">${c.label}</div>
      <div class="stat-value ${c.cls}">${c.value}</div>
      <div class="stat-sub">${c.sub}</div>
    </div>`
    ).join('');
}

// ---- Chart ----
function renderChart(data) {
    if (frontierChart) { frontierChart.destroy(); frontierChart = null; }

    const mc = data.monte_carlo_points || [];
    const fp = data.frontier_points || [];
    const ms = data.max_sharpe;
    const mv = data.min_variance;
    const cp = data.current_portfolio_stats;

    // Downsample Monte Carlo points for performance
    const maxDots = 2500;
    const step = mc.length > maxDots ? Math.floor(mc.length / maxDots) : 1;
    const mcSampled = mc.filter((_, i) => i % step === 0);

    // Color MC points by Sharpe ratio
    const sharpes = mcSampled.map(p => p.sharpe);
    const minS = Math.min(...sharpes), maxS = Math.max(...sharpes);
    const mcColors = mcSampled.map(p => sharpeColor(p.sharpe, minS, maxS));

    const datasets = [
        {
            label: 'Simulated Portfolios',
            data: mcSampled.map(p => ({ x: p.risk * 100, y: p.return * 100 })),
            backgroundColor: mcColors,
            pointRadius: 2.5,
            pointHoverRadius: 4,
            type: 'scatter',
            order: 3,
        },
        {
            label: 'Efficient Frontier',
            data: fp.map(p => ({ x: p.risk * 100, y: p.return * 100 })),
            borderColor: '#10d9a0',
            backgroundColor: 'transparent',
            borderWidth: 2.5,
            pointRadius: 0,
            type: 'line',
            tension: 0.4,
            order: 2,
        },
        {
            label: 'Max Sharpe Ratio',
            data: [{ x: ms.risk * 100, y: ms.return * 100 }],
            backgroundColor: '#fbbf24',
            borderColor: '#000',
            borderWidth: 1.5,
            pointRadius: 10,
            pointStyle: 'star',
            type: 'scatter',
            order: 1,
        },
        {
            label: 'Min Variance',
            data: [{ x: mv.risk * 100, y: mv.return * 100 }],
            backgroundColor: '#38bdf8',
            borderColor: '#000',
            borderWidth: 1.5,
            pointRadius: 9,
            pointStyle: 'triangle',
            type: 'scatter',
            order: 1,
        },
    ];

    if (cp) {
        datasets.push({
            label: 'Your Portfolio',
            data: [{ x: cp.risk * 100, y: cp.return * 100 }],
            backgroundColor: '#f472b6',
            borderColor: '#fff',
            borderWidth: 2,
            pointRadius: 11,
            pointStyle: 'rectRot',
            type: 'scatter',
            order: 0,
        });
    }

    // Legend
    const legendItems = [
        { label: 'Simulated (colored by Sharpe)', color: '#8b949e', shape: 'circle' },
        { label: 'Efficient Frontier', color: '#10d9a0', shape: 'line' },
        { label: '★ Max Sharpe', color: '#fbbf24', shape: 'circle' },
        { label: '▲ Min Variance', color: '#38bdf8', shape: 'circle' },
    ];
    if (cp) legendItems.push({ label: '◆ Your Portfolio', color: '#f472b6', shape: 'circle' });

    document.getElementById('chartLegend').innerHTML = legendItems.map(l =>
        `<div class="legend-item">
      <div class="legend-dot" style="background:${l.color}"></div>
      <span>${l.label}</span>
    </div>`
    ).join('');

    const ctx = document.getElementById('frontierChart').getContext('2d');
    frontierChart = new Chart(ctx, {
        type: 'scatter',
        data: { datasets },
        options: {
            responsive: true,
            maintainAspectRatio: false,
            animation: { duration: 600 },
            plugins: {
                legend: { display: false },
                tooltip: {
                    callbacks: {
                        label: (ctx) => {
                            const { x, y } = ctx.raw;
                            const ds = ctx.dataset.label;
                            return ` ${ds}: Return ${y.toFixed(2)}%  |  Risk ${x.toFixed(2)}%`;
                        },
                    },
                    backgroundColor: 'rgba(22,27,34,0.95)',
                    borderColor: 'rgba(255,255,255,0.1)',
                    borderWidth: 1,
                    titleColor: '#e6edf3',
                    bodyColor: '#8b949e',
                    padding: 10,
                },
            },
            scales: {
                x: {
                    title: { display: true, text: 'Annual Risk (Volatility %)', color: '#8b949e', font: { size: 12 } },
                    grid: { color: 'rgba(255,255,255,0.04)' },
                    ticks: { color: '#8b949e', callback: v => v.toFixed(1) + '%' },
                },
                y: {
                    title: { display: true, text: 'Annual Return (%)', color: '#8b949e', font: { size: 12 } },
                    grid: { color: 'rgba(255,255,255,0.04)' },
                    ticks: { color: '#8b949e', callback: v => v.toFixed(1) + '%' },
                },
            },
        },
    });
}

// Map Sharpe to a green→yellow→red gradient
function sharpeColor(s, min, max) {
    const t = max === min ? 0.5 : (s - min) / (max - min);
    // low sharpe → muted blue, high → emerald
    const r = Math.round(lerp(99, 16, t));
    const g = Math.round(lerp(102, 217, t));
    const b = Math.round(lerp(220, 160, t));
    return `rgba(${r},${g},${b},0.55)`;
}
function lerp(a, b, t) { return a + (b - a) * t; }

// ---- Weight bars ----
function renderWeights(containerId, tickers, weights) {
    const container = document.getElementById(containerId);
    if (!weights || weights.length === 0) { container.innerHTML = '<p style="color:var(--text-muted);font-size:0.8rem">No data</p>'; return; }

    // Sort by descending weight
    const pairs = tickers.map((t, i) => ({ t, w: weights[i] || 0 }))
        .sort((a, b) => b.w - a.w);

    container.innerHTML = pairs.map(({ t, w }, i) => {
        const wpct = (w * 100).toFixed(1);
        const color = WEIGHT_COLORS[tickers.indexOf(t) % WEIGHT_COLORS.length];
        return `<div class="weight-row">
      <div class="weight-meta">
        <span class="weight-ticker">${t}</span>
        <span class="weight-pct" style="color:${color}">${wpct}%</span>
      </div>
      <div class="weight-bar-bg">
        <div class="weight-bar" style="width:${Math.min(w * 100, 100)}%; background:${color}"></div>
      </div>
    </div>`;
    }).join('');
}

// ---- Asset stats table ----
function renderAssetTable(data) {
    const tbody = document.getElementById('assetTableBody');
    tbody.innerHTML = (data.asset_stats || []).map((a, i) => {
        const sharpe = (a.annual_return - currentRiskFreeRate) / a.annual_volatility;
        const color = WEIGHT_COLORS[i % WEIGHT_COLORS.length];
        return `<tr>
      <td><strong style="color:${color}">${a.ticker}</strong></td>
      <td class="${a.annual_return >= 0 ? 'cmp-better' : 'cmp-worse'}">${pct(a.annual_return)}</td>
      <td>${pct(a.annual_volatility)}</td>
      <td class="${sharpe >= 1 ? 'cmp-better' : ''}">${sharpe.toFixed(2)}</td>
    </tr>`;
    }).join('');
}

// ---- Comparison panel ----
function renderComparison(data) {
    const cp = data.current_portfolio_stats;
    const ms = data.max_sharpe;
    const mv = data.min_variance;

    const rows = [
        { label: 'Annual Return', cur: cp.return, ms: ms.return, mv: mv.return, fmt: pct, higherBetter: true },
        { label: 'Annual Volatility', cur: cp.risk, ms: ms.risk, mv: mv.risk, fmt: pct, higherBetter: false },
        { label: 'Sharpe Ratio', cur: cp.sharpe, ms: ms.sharpe, mv: mv.sharpe, fmt: v => v.toFixed(2), higherBetter: true },
    ];

    document.getElementById('comparisonBody').innerHTML = rows.map(r => {
        const msCls = cmpClass(r.cur, r.ms, r.higherBetter);
        const mvCls = cmpClass(r.cur, r.mv, r.higherBetter);
        return `<tr>
      <td>${r.label}</td>
      <td>${r.fmt(r.cur)}</td>
      <td class="${msCls}">${r.fmt(r.ms)} ${cmpArrow(r.cur, r.ms, r.higherBetter)}</td>
      <td class="${mvCls}">${r.fmt(r.mv)} ${cmpArrow(r.cur, r.mv, r.higherBetter)}</td>
    </tr>`;
    }).join('');

    renderWeights('currentWeightsDisplay', data.tickers, cp.weights);
    renderWeights('rebalanceDisplay', data.tickers, ms.weights);
}

function cmpClass(cur, opt, higherBetter) {
    if (higherBetter) return opt > cur * 1.01 ? 'cmp-better' : opt < cur * 0.99 ? 'cmp-worse' : '';
    return opt < cur * 0.99 ? 'cmp-better' : opt > cur * 1.01 ? 'cmp-worse' : '';
}
function cmpArrow(cur, opt, higherBetter) {
    const better = higherBetter ? opt > cur * 1.01 : opt < cur * 0.99;
    const worse = higherBetter ? opt < cur * 0.99 : opt > cur * 1.01;
    if (better) return '↑';
    if (worse) return '↓';
    return '—';
}

// ---- Utils ----
function pct(v) {
    if (v === undefined || v === null || isNaN(v)) return '—';
    return (v * 100).toFixed(2) + '%';
}
