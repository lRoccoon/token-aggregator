package main

import "net/http"

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write([]byte(dashboardHTML))
}

const dashboardHTML = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Token Ledger — Usage Dashboard</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Instrument+Serif:ital@0;1&family=Instrument+Sans:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;600;700&family=Noto+Sans+SC:wght@400;500;700&display=swap" rel="stylesheet">
<style>
:root{
  --bg:#0b0c0d;
  --bg-2:#111315;
  --panel:#0e1012;
  --line:#1f2225;
  --line-2:#3a3e44;
  --paper:#f4f1ea;
  --paper-2:#cfcabf;
  --muted:#7a7e84;
  --signal:#a7ff7a;
  --signal-2:#7be251;
  --warm:#ff8b3d;
  --danger:#ff6b6b;
  --heat-0:#15181b;
  --heat-1:#293f2a;
  --heat-2:#4a7544;
  --heat-3:#7fbf5d;
  --heat-4:#a7ff7a;
}
*{box-sizing:border-box}
html,body{background:var(--bg);color:var(--paper)}
body{
  margin:0;
  font-family:'Instrument Sans','Noto Sans SC','Helvetica Neue',system-ui,"PingFang SC","Hiragino Sans GB","Microsoft YaHei","Noto Sans CJK SC",sans-serif;
  font-feature-settings:"ss01","cv11";
  font-size:15px;
  line-height:1.55;
  -webkit-font-smoothing:antialiased;
  position:relative;
  min-height:100vh;
}
body::before{
  content:'';
  position:fixed;inset:0;
  background-image:url("data:image/svg+xml;utf8,<svg xmlns='http://www.w3.org/2000/svg' width='180' height='180'><filter id='n'><feTurbulence type='fractalNoise' baseFrequency='0.9' numOctaves='2' stitchTiles='stitch'/><feColorMatrix values='0 0 0 0 1 0 0 0 0 1 0 0 0 0 1 0 0 0 0.06 0'/></filter><rect width='100%25' height='100%25' filter='url(%23n)'/></svg>");
  opacity:.45;
  mix-blend-mode:overlay;
  pointer-events:none;
  z-index:1;
}
body::after{
  content:'';
  position:fixed;
  inset:auto -20% -40% -20%;
  height:60vh;
  background:radial-gradient(ellipse at 50% 100%, rgba(167,255,122,.08), transparent 60%);
  pointer-events:none;
  z-index:0;
}
.shell{max-width:1280px;margin:0 auto;padding:24px 36px 96px;position:relative;z-index:2}

/* ─────── Masthead ─────── */
.masthead{
  display:grid;
  grid-template-columns:auto 1fr auto;
  gap:24px;align-items:center;
  padding:14px 0 18px;
  border-bottom:1px solid var(--line);
  font-family:'JetBrains Mono',ui-monospace,monospace;
  font-size:11px;letter-spacing:.16em;text-transform:uppercase;
}
.masthead .iss{color:var(--muted)}
.masthead .iss b{color:var(--signal);font-weight:600;letter-spacing:.18em}
.masthead .iss .live{display:inline-block;width:6px;height:6px;border-radius:50%;background:var(--signal);margin-right:8px;box-shadow:0 0 6px var(--signal);animation:blink 1.6s infinite}
.masthead .center{display:flex;justify-content:center;gap:2px}
.ranges{display:flex;gap:0}
.ranges button{
  background:transparent;color:var(--muted);
  border:1px solid var(--line-2);border-right:none;
  padding:9px 14px;cursor:pointer;
  font-family:inherit;font-size:11px;letter-spacing:.14em;text-transform:uppercase;
  transition:.14s ease;
}
.ranges button:last-child{border-right:1px solid var(--line-2)}
.ranges button:hover{color:var(--paper);border-color:var(--paper)}
.ranges button.active{background:var(--signal);color:#0a0c0d;border-color:var(--signal);font-weight:600}
.masthead .auth{display:flex;gap:6px;align-items:center}
.masthead .auth .input{
  background:transparent;color:var(--paper);
  border:1px solid var(--line-2);
  padding:9px 12px;
  font-family:inherit;font-size:11px;letter-spacing:.1em;
  width:140px;outline:none;text-transform:uppercase;
}
.masthead .auth .input:focus{border-color:var(--signal)}
.masthead .auth .input.num{width:64px;text-align:center}
.masthead .auth .apply{
  background:transparent;color:var(--paper);
  border:1px solid var(--paper);
  padding:9px 14px;cursor:pointer;
  font-family:inherit;font-size:11px;letter-spacing:.14em;text-transform:uppercase;
  transition:.14s;
}
.masthead .auth .apply:hover{background:var(--paper);color:#0a0c0d}

/* ─────── Hero ─────── */
.hero{
  padding:72px 0 36px;
  display:grid;
  grid-template-columns:1.4fr 1fr;
  gap:48px;align-items:end;
}
.hero h1{
  font-family:'Instrument Serif','Times New Roman',serif;
  font-weight:400;
  font-size:clamp(56px,9.6vw,140px);
  line-height:.88;
  letter-spacing:-.045em;
  margin:0;
}
.hero h1 em{font-style:italic;color:var(--signal);position:relative}
.hero h1 em::after{
  content:'';position:absolute;left:0;right:0;bottom:-.12em;
  height:2px;background:linear-gradient(90deg,var(--signal),transparent);
  transform-origin:left;animation:underline 1.2s .35s ease both;
}
.hero .kicker{
  font-family:'Instrument Serif',serif;
  font-style:italic;font-size:18px;line-height:1.55;
  color:var(--paper-2);
  max-width:420px;margin:0;
}
.hero .kicker .stamp{
  display:block;margin-top:14px;
  font-family:'JetBrains Mono',monospace;font-style:normal;
  font-size:10px;letter-spacing:.2em;text-transform:uppercase;color:var(--muted);
}

/* ─────── Filters ─────── */
.filters{
  margin-top:48px;
  border-top:1px solid var(--line);
  border-bottom:1px solid var(--line);
  padding:18px 0;
}
.filter-row{
  display:grid;
  grid-template-columns:160px 1fr;
  gap:24px;
  padding:10px 0;
  align-items:start;
}
.filter-row + .filter-row{border-top:1px dashed var(--line)}
.filter-row .label{
  font-family:'JetBrains Mono',monospace;
  font-size:11px;letter-spacing:.18em;text-transform:uppercase;
  color:var(--muted);
  padding-top:6px;
}
.filter-row .label::after{content:'  ────';color:var(--line-2)}
.chips{display:flex;gap:6px;flex-wrap:wrap}
.chip{
  font-family:'JetBrains Mono','Noto Sans SC',ui-monospace,monospace;
  font-size:12px;letter-spacing:.04em;
  background:transparent;color:var(--paper);
  border:1px solid var(--line-2);
  padding:7px 12px;cursor:pointer;
  transition:.14s ease;
  border-radius:0;
}
.chip:hover{border-color:var(--paper)}
.chip.active{background:var(--paper);color:#0a0c0d;border-color:var(--paper)}
.chip.all{font-weight:600}
.chip.all.active{background:var(--signal);color:#0a0c0d;border-color:var(--signal)}

/* ─────── Chapter heading ─────── */
.chapter{
  display:grid;
  grid-template-columns:auto 1fr auto;
  gap:18px;align-items:baseline;
  padding:64px 0 22px;
}
.chap-no{
  font-family:'JetBrains Mono',monospace;
  font-size:12px;letter-spacing:.18em;color:var(--signal);
}
.chap-title{
  font-family:'Instrument Serif',serif;
  font-style:italic;font-weight:400;
  font-size:clamp(28px,3.4vw,42px);
  letter-spacing:-.02em;margin:0;
}
.chap-title b{font-style:normal;color:var(--paper-2);font-weight:400;font-family:'Noto Sans SC','Instrument Sans',system-ui,"PingFang SC","Microsoft YaHei",sans-serif;font-size:.5em;letter-spacing:.06em;margin-left:14px;font-weight:500}
.chap-meta{
  font-family:'JetBrains Mono',monospace;
  font-size:11px;letter-spacing:.16em;text-transform:uppercase;color:var(--muted);
}

/* ─────── KPIs ─────── */
.kpis{
  display:grid;
  grid-template-columns:repeat(4,1fr);
  border:1px solid var(--line);
}
.kpi{
  padding:24px 26px 28px;
  border-right:1px solid var(--line);
  position:relative;
  background:linear-gradient(180deg,rgba(255,255,255,.012),transparent);
  animation:slide-in .6s ease both;
}
.kpi:last-child{border-right:none}
.kpi:nth-child(1){animation-delay:.04s}.kpi:nth-child(2){animation-delay:.08s}.kpi:nth-child(3){animation-delay:.12s}.kpi:nth-child(4){animation-delay:.16s}
.kpi-key{
  font-family:'JetBrains Mono',monospace;
  font-size:10px;letter-spacing:.18em;text-transform:uppercase;color:var(--muted);
  display:flex;justify-content:space-between;align-items:center;
}
.kpi-key .marker{width:6px;height:6px;background:var(--signal);border-radius:0}
.kpi:nth-child(2) .kpi-key .marker,.kpi:nth-child(4) .kpi-key .marker{background:var(--warm)}
.kpi-num{
  font-family:'JetBrains Mono',monospace;
  font-size:clamp(34px,4.8vw,60px);
  font-weight:600;letter-spacing:-.025em;line-height:1;
  margin:18px 0 14px;
  font-variant-numeric:tabular-nums;
  color:var(--paper);
}
.kpi-foot{
  font-family:'Instrument Serif',serif;
  font-style:italic;font-size:13px;color:var(--muted);
}

/* ─────── Distribution grid ─────── */
.distribution{
  display:grid;
  grid-template-columns:1.85fr 1fr;
  border:1px solid var(--line);
  border-top:none;
}
.block{padding:28px;border-right:1px solid var(--line);min-width:0}
.block-head{
  display:flex;justify-content:space-between;align-items:baseline;
  margin-bottom:22px;
  padding-bottom:14px;border-bottom:1px dashed var(--line);
}
.block-head h3{
  font-family:'Instrument Serif',serif;
  font-style:italic;font-weight:400;font-size:22px;margin:0;
  letter-spacing:-.01em;
}
.block-head .badge{
  font-family:'JetBrains Mono',monospace;
  font-size:10px;letter-spacing:.16em;text-transform:uppercase;color:var(--muted);
  border:1px solid var(--line-2);padding:4px 8px;
}
.chart{display:block;width:100%;height:340px;overflow:visible}
.axis{stroke:var(--line);stroke-width:1}
.bar{fill:var(--signal);transition:.15s}
.bar:hover{fill:var(--paper)}
.line{fill:none;stroke:var(--warm);stroke-width:2;stroke-linecap:round;stroke-linejoin:round}
.dot{fill:var(--warm);stroke:var(--bg);stroke-width:2}
.legend{
  display:flex;gap:18px;flex-wrap:wrap;
  margin-top:16px;
  font-family:'JetBrains Mono',monospace;
  font-size:11px;letter-spacing:.12em;text-transform:uppercase;color:var(--muted);
  align-items:center;
}
.legend .sw{display:inline-block;width:10px;height:10px;margin-right:6px;vertical-align:middle}
.legend .sw.sig{background:var(--signal)}
.legend .sw.warm{background:var(--warm)}
.heatmap{overflow-x:auto;padding:6px 2px 14px}
.heat-cell{stroke:var(--bg);stroke-width:1;cursor:pointer;transition:.15s}
.heat-cell:hover{stroke:var(--paper);stroke-width:1.5}
.heat-legend{display:inline-flex;align-items:center;gap:6px;margin-left:auto}
.heat-legend i{display:inline-block;width:11px;height:11px}
.empty{
  height:330px;display:grid;place-items:center;
  border:1px dashed var(--line);
  font-family:'Instrument Serif',serif;font-style:italic;color:var(--muted);
}
.status{
  margin-top:14px;color:var(--muted);font-size:12px;
  font-family:'JetBrains Mono',monospace;letter-spacing:.08em;
}
.status.error{color:var(--danger)}

/* Rank lists */
.ranks{padding:28px}
.rank-block + .rank-block{margin-top:30px;padding-top:22px;border-top:1px solid var(--line)}
.rank-head{
  display:flex;justify-content:space-between;align-items:baseline;
  margin-bottom:14px;
}
.rank-head h4{
  font-family:'Instrument Serif',serif;font-style:italic;font-weight:400;
  font-size:18px;margin:0;letter-spacing:-.01em;
}
.rank-head .meta{
  font-family:'JetBrains Mono',monospace;
  font-size:10px;letter-spacing:.14em;text-transform:uppercase;color:var(--muted);
}
.rank-list{list-style:none;padding:0;margin:0;counter-reset:rank}
.rank-list li{
  position:relative;padding:14px 0;border-bottom:1px solid var(--line);
  display:grid;grid-template-columns:28px 1fr;gap:14px;
  counter-increment:rank;
}
.rank-list li:last-child{border-bottom:none}
.rank-list li::before{
  content:counter(rank,decimal-leading-zero);
  font-family:'JetBrains Mono',monospace;font-size:10px;color:var(--muted);
  align-self:center;
}
.rank-row{display:flex;justify-content:space-between;gap:8px;align-items:baseline}
.rank-row .name{font-family:'Instrument Sans',sans-serif;font-size:14px;color:var(--paper);font-weight:500}
.rank-row .num{font-family:'JetBrains Mono',monospace;font-size:13px;color:var(--paper-2);font-variant-numeric:tabular-nums}
.rank-bar{grid-column:2;height:2px;background:var(--line);position:relative;margin-top:8px}
.rank-bar span{position:absolute;left:0;top:0;bottom:0;background:var(--signal);transition:width .5s ease}
.rank-block.devices .rank-bar span{background:var(--warm)}
.rank-list li.dim{opacity:.32}

/* ─────── Daily ledger ─────── */
.ledger-wrap{
  border:1px solid var(--line);border-top:none;
}
.ledger{
  width:100%;border-collapse:collapse;
  font-family:'JetBrains Mono',monospace;
  font-size:13px;font-variant-numeric:tabular-nums;
}
.ledger thead th{
  text-align:left;
  font-family:'Instrument Sans',sans-serif;font-weight:600;
  font-size:11px;letter-spacing:.18em;text-transform:uppercase;color:var(--muted);
  padding:14px 22px;
  border-bottom:1px solid var(--line-2);
  background:linear-gradient(180deg,rgba(255,255,255,.02),transparent);
}
.ledger thead th:not(:first-child){text-align:right}
.ledger tbody td{
  padding:14px 22px;border-bottom:1px solid var(--line);
  color:var(--paper);
}
.ledger tbody td:not(:first-child){text-align:right}
.ledger tr.daily-row{cursor:pointer;transition:.14s}
.ledger tr.daily-row:hover td{background:rgba(167,255,122,.04);color:var(--signal)}
.ledger tr.daily-row.open td{background:rgba(167,255,122,.07);color:var(--signal)}
.ledger tr.daily-row td:first-child{position:relative;padding-left:46px}
.ledger tr.daily-row td:first-child::before{
  content:'';position:absolute;left:22px;top:50%;
  width:8px;height:8px;background:var(--line-2);
  transform:translateY(-50%) rotate(45deg);
  transition:.14s;
}
.ledger tr.daily-row:hover td:first-child::before{background:var(--paper)}
.ledger tr.daily-row.open td:first-child::before{background:var(--signal);transform:translateY(-50%) rotate(0)}
.toggle{display:inline-block;color:var(--muted);margin-right:10px;transition:transform .15s}
.daily-row.open .toggle{transform:rotate(90deg);color:var(--signal)}
.ledger tr.detail-row td{background:var(--bg-2);padding:0}
.detail-wrap{padding:22px 30px 26px}
.detail-wrap h4{
  font-family:'Instrument Serif',serif;font-style:italic;font-weight:400;
  font-size:16px;color:var(--paper-2);
  margin:0 0 16px;letter-spacing:-.005em;
}
.detail-wrap h4 .stamp{
  font-family:'JetBrains Mono',monospace;font-style:normal;
  font-size:10px;letter-spacing:.16em;text-transform:uppercase;
  color:var(--muted);margin-left:14px;
}
table.detail{
  width:auto;min-width:65%;border-collapse:collapse;
  font-family:'JetBrains Mono',monospace;font-size:12px;
}
table.detail th,table.detail td{
  padding:9px 16px;border-bottom:1px solid var(--line);
  text-align:right;
}
table.detail thead th{
  font-family:'Instrument Sans',sans-serif;font-weight:600;
  font-size:10px;letter-spacing:.16em;text-transform:uppercase;
  color:var(--muted);
  border-bottom:1px solid var(--line-2);
}
table.detail th:first-child,table.detail td:first-child{
  text-align:left;color:var(--signal);font-weight:600;
}
table.detail td .sub{
  display:block;font-family:'Instrument Serif',serif;font-style:italic;
  font-size:11px;color:var(--muted);margin-top:2px;
}
table.detail tr.totals td{border-top:1px solid var(--line-2);border-bottom:none;color:var(--paper)}

/* ─────── Animations ─────── */
@keyframes slide-in{from{opacity:0;transform:translateY(10px)}to{opacity:1;transform:translateY(0)}}
@keyframes underline{from{transform:scaleX(0)}to{transform:scaleX(1)}}
@keyframes blink{0%,55%{opacity:1}56%,100%{opacity:.18}}

/* ─────── Footer ─────── */
.colophon{
  margin-top:60px;padding-top:24px;
  border-top:1px solid var(--line);
  display:flex;justify-content:space-between;
  font-family:'JetBrains Mono',monospace;
  font-size:11px;letter-spacing:.16em;text-transform:uppercase;color:var(--muted);
}
.colophon em{color:var(--paper-2);font-style:italic;font-family:'Instrument Serif',serif;text-transform:none;letter-spacing:0;font-size:13px}

/* ─────── Responsive ─────── */
@media (max-width:980px){
  .shell{padding:20px 22px 60px}
  .masthead{grid-template-columns:1fr;gap:14px}
  .masthead .center,.masthead .auth{justify-content:flex-start}
  .hero{grid-template-columns:1fr;gap:24px;padding:48px 0 24px}
  .hero h1{font-size:clamp(48px,12vw,96px)}
  .kpis{grid-template-columns:repeat(2,1fr)}
  .kpi{border-right:1px solid var(--line);border-bottom:1px solid var(--line)}
  .kpi:nth-child(2){border-right:none}
  .kpi:nth-child(3),.kpi:nth-child(4){border-bottom:none}
  .distribution{grid-template-columns:1fr}
  .block{border-right:none;border-bottom:1px solid var(--line)}
  .filter-row{grid-template-columns:1fr;gap:8px}
  .chapter{grid-template-columns:auto 1fr;gap:12px}
  .chap-meta{display:none}
}
@media (max-width:560px){
  .shell{padding:16px 16px 40px}
  .kpis{grid-template-columns:1fr}
  .kpi{border-right:none}
  .ledger thead th,.ledger tbody td{padding:12px 14px;font-size:12px}
  .ledger tr.daily-row td:first-child{padding-left:36px}
  .ledger tr.daily-row td:first-child::before{left:14px}
}
</style>
</head>
<body>
<main class="shell">

  <header class="masthead">
    <div class="iss"><span class="live"></span><b>token.ledger</b> · № 001 · personal-automation</div>
    <div class="center">
      <div class="ranges" aria-label="time range filters">
        <button data-days="7">7D</button>
        <button data-days="30" class="active">30D</button>
        <button data-days="180">180D</button>
        <button data-days="365">365D</button>
      </div>
    </div>
    <div class="auth">
      <input class="input" id="authToken" type="password" placeholder="bearer" title="Bearer token">
      <input class="input num" id="customDays" type="number" min="1" max="365" value="30" title="自定义天数">
      <button class="apply" id="applyDays">apply</button>
    </div>
  </header>

  <section class="hero">
    <h1>Token usage,<br><em>over time.</em></h1>
    <p class="kicker">A quiet ledger of every conversation, every model, every device — kept in plain sight.<span class="stamp">∎ updated <span id="updatedAt">just now</span></span></p>
  </section>

  <section class="filters" aria-label="filters">
    <div class="filter-row">
      <span class="label">Devices</span>
      <div class="chips" id="deviceChips"></div>
    </div>
    <div class="filter-row">
      <span class="label">Sources</span>
      <div class="chips" id="sourceChips"></div>
    </div>
  </section>

  <section class="chapter">
    <span class="chap-no">[ 01 ]</span>
    <h2 class="chap-title">Overview<b>概览</b></h2>
    <span class="chap-meta" id="rangeLabel">—</span>
  </section>

  <section class="kpis">
    <article class="kpi">
      <div class="kpi-key"><span>Total Tokens</span><span class="marker"></span></div>
      <div class="kpi-num" id="totalTokens">—</div>
      <div class="kpi-foot">over the selected window</div>
    </article>
    <article class="kpi">
      <div class="kpi-key"><span>Total Cost</span><span class="marker"></span></div>
      <div class="kpi-num" id="totalCost">—</div>
      <div class="kpi-foot">aggregated from price book</div>
    </article>
    <article class="kpi">
      <div class="kpi-key"><span>Avg / Day · Tokens</span><span class="marker"></span></div>
      <div class="kpi-num" id="avgTokens">—</div>
      <div class="kpi-foot">mean across active days</div>
    </article>
    <article class="kpi">
      <div class="kpi-key"><span>Avg / Day · Cost</span><span class="marker"></span></div>
      <div class="kpi-num" id="avgCost">—</div>
      <div class="kpi-foot">mean across active days</div>
    </article>
  </section>

  <section class="chapter">
    <span class="chap-no">[ 02 ]</span>
    <h2 class="chap-title">Distribution<b>分布</b></h2>
    <span class="chap-meta">device · source</span>
  </section>

  <section class="distribution">
    <article class="block">
      <header class="block-head">
        <h3 id="chartTitle">Tokens &amp; Cost Chart</h3>
        <span class="badge" id="viewBadge">bars + line</span>
      </header>
      <div id="chartMount"></div>
      <div class="legend" id="legend"></div>
      <div class="status" id="status">loading…</div>
    </article>
    <aside class="ranks">
      <div class="rank-block devices">
        <div class="rank-head"><h4>Devices</h4><span class="meta">by tokens</span></div>
        <ol class="rank-list" id="devices"></ol>
      </div>
      <div class="rank-block sources">
        <div class="rank-head"><h4>Sources</h4><span class="meta">by tokens</span></div>
        <ol class="rank-list" id="sources"></ol>
      </div>
    </aside>
  </section>

  <section class="chapter">
    <span class="chap-no">[ 03 ]</span>
    <h2 class="chap-title">Daily Ledger<b>每日明细</b></h2>
    <span class="chap-meta">click row to expand · device × source</span>
  </section>

  <section class="ledger-wrap">
    <table class="ledger">
      <thead><tr><th>Date</th><th>Tokens</th><th>Cost (USD)</th><th>Composition</th></tr></thead>
      <tbody id="dailyRows"></tbody>
    </table>
  </section>

  <footer class="colophon">
    <span>token.ledger / dashboard</span>
    <em>kept honest, one row at a time.</em>
    <span>v.dashboard.history</span>
  </footer>

</main>
<script>
const fmtTokens = n => Intl.NumberFormat('en', {notation:n>=10000?'compact':'standard', maximumFractionDigits:1}).format(n || 0)
const fmtMoney = n => '$' + Number(n || 0).toFixed(4).replace(/0+$/,'').replace(/\.$/,'.00')
const esc = v => String(v ?? '').replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]))
const qs = new URLSearchParams(location.search)
const state = {
  days: Math.min(365, Math.max(1, Number(qs.get('days')) || 30)),
  devices: new Set((qs.get('devices')||'').split(',').filter(Boolean)),
  sources: new Set((qs.get('sources')||'').split(',').filter(Boolean)),
  candidates: {devices: [], sources: []},
  expanded: new Set(),
}
let token = qs.get('token') || sessionStorage.getItem('tokenAggregatorToken') || ''
const authInput = document.getElementById('authToken')
authInput.value = token
if (qs.get('token')) {
  sessionStorage.setItem('tokenAggregatorToken', qs.get('token'))
  qs.delete('token')
  syncURL()
}

function syncURL(){
  const params = new URLSearchParams()
  params.set('days', String(state.days))
  if (state.devices.size) params.set('devices', [...state.devices].join(','))
  if (state.sources.size) params.set('sources', [...state.sources].join(','))
  if (qs.get('today')) params.set('today', qs.get('today'))
  history.replaceState(null,'','?'+params.toString())
}

function setActive(days){
  document.querySelectorAll('[data-days]').forEach(b=>b.classList.toggle('active', Number(b.dataset.days)===days))
  document.getElementById('customDays').value = days
}

async function load(days){
  state.days = days
  setActive(days)
  const status = document.getElementById('status')
  status.textContent = 'loading…'
  status.className = 'status'
  token = authInput.value.trim()
  if (token) sessionStorage.setItem('tokenAggregatorToken', token); else sessionStorage.removeItem('tokenAggregatorToken')
  const headers = token ? {Authorization:'Bearer '+token} : {}
  const params = new URLSearchParams({days: String(days)})
  if (state.devices.size) params.set('devices', [...state.devices].join(','))
  if (state.sources.size) params.set('sources', [...state.sources].join(','))
  if (qs.get('today')) params.set('today', qs.get('today'))
  const res = await fetch('/history?'+params.toString(), {headers})
  if(!res.ok){status.textContent='load failed: '+res.status+' '+await res.text();status.className='status error';return}
  const data = await res.json()
  render(data)
  syncURL()
}

function render(data){
  state.candidates.devices = data.devices || []
  state.candidates.sources = data.sources || []
  for (const v of [...state.devices]) if (!state.candidates.devices.includes(v)) state.devices.delete(v)
  for (const v of [...state.sources]) if (!state.candidates.sources.includes(v)) state.sources.delete(v)

  document.getElementById('totalTokens').textContent = fmtTokens(data.summary.total_tokens)
  document.getElementById('totalCost').textContent = fmtMoney(data.summary.total_cost_usd)
  document.getElementById('avgTokens').textContent = fmtTokens(data.summary.avg_tokens_per_day)
  document.getElementById('avgCost').textContent = fmtMoney(data.summary.avg_cost_usd_per_day)
  document.getElementById('rangeLabel').textContent = data.from + ' → ' + data.to + ' · ' + data.days + ' days'
  const ts = new Date()
  document.getElementById('updatedAt').textContent = ts.toLocaleString('en-GB',{hour12:false})
  document.getElementById('status').textContent = 'updated ' + ts.toLocaleTimeString('en-GB',{hour12:false})

  renderChips()
  if (state.days === 365) {
    document.getElementById('chartTitle').textContent = 'Yearly Heatmap'
    document.getElementById('viewBadge').textContent = 'heatmap · 53w × 7d'
    drawHeatmap(data.daily || [])
  } else {
    document.getElementById('chartTitle').textContent = 'Tokens & Cost Chart'
    document.getElementById('viewBadge').textContent = 'bars + line'
    drawChart(data.daily || [])
  }
  drawDistribution('sources', data.daily || [])
  drawDistribution('devices', data.daily || [])
  drawRows(data.daily || [])
}

function renderChips(){
  const groups = [['deviceChips','devices','全部设备'], ['sourceChips','sources','全部来源']]
  for (const [mountId, dim, allLabel] of groups){
    const mount = document.getElementById(mountId)
    const cands = state.candidates[dim]
    const sel = state[dim]
    const allActive = sel.size === 0
    const html = ['<button class="chip all'+(allActive?' active':'')+'" data-dim="'+dim+'" data-all="1">'+esc(allLabel)+'</button>']
    for (const v of cands){
      const active = sel.has(v) || allActive
      html.push('<button class="chip'+(active?' active':'')+'" data-dim="'+dim+'" data-value="'+esc(v)+'">'+esc(v)+'</button>')
    }
    mount.innerHTML = html.join('')
  }
}

function onChipClick(e){
  const btn = e.target.closest('.chip')
  if (!btn) return
  const dim = btn.dataset.dim
  const sel = state[dim]
  const cands = state.candidates[dim]
  if (btn.dataset.all === '1') {
    sel.clear()
  } else {
    const v = btn.dataset.value
    const allActive = sel.size === 0
    if (allActive) {
      cands.forEach(x => { if (x !== v) sel.add(x) })
    } else if (sel.has(v)) {
      sel.delete(v)
    } else {
      sel.add(v)
    }
    if (sel.size === cands.length) sel.clear()
  }
  load(state.days).catch(showError)
}

function showError(err){
  const s = document.getElementById('status')
  s.textContent = 'load failed: ' + err.message
  s.className = 'status error'
}

function drawChart(rows){
  const mount = document.getElementById('chartMount')
  document.getElementById('legend').innerHTML = '<span><i class="sw sig"></i>tokens</span><span><i class="sw warm"></i>cost (usd)</span>'
  if(!rows.length){mount.innerHTML='<div class="empty">no data in range</div>';return}
  const W=920,H=340,p={l:42,r:28,t:18,b:42},cw=W-p.l-p.r,ch=H-p.t-p.b
  const maxTokens=Math.max(1,...rows.map(d=>d.total_tokens||0))
  const maxCost=Math.max(.000001,...rows.map(d=>d.cost_usd||0))
  const step=cw/rows.length, barW=Math.max(2,Math.min(20,step*.6))
  const points=[]
  const bars=rows.map((d,i)=>{
    const x=p.l+i*step+(step-barW)/2
    const h=(d.total_tokens||0)/maxTokens*ch
    const y=p.t+ch-h
    const cy=p.t+ch-((d.cost_usd||0)/maxCost*ch)
    points.push([p.l+i*step+step/2,cy])
    return '<rect class="bar" x="'+x.toFixed(1)+'" y="'+y.toFixed(1)+'" width="'+barW.toFixed(1)+'" height="'+Math.max(1,h).toFixed(1)+'"><title>'+esc(d.date)+'\n'+fmtTokens(d.total_tokens)+' tokens\n'+fmtMoney(d.cost_usd)+'</title></rect>'
  }).join('')
  const line=points.map((pt,i)=>(i?'L':'M')+pt[0].toFixed(1)+' '+pt[1].toFixed(1)).join(' ')
  const ticks=[0,.25,.5,.75,1].map(t=>'<line class="axis" x1="'+p.l+'" x2="'+(W-p.r)+'" y1="'+(p.t+ch-t*ch)+'" y2="'+(p.t+ch-t*ch)+'"/>').join('')
  const labelIdx = rows.length<=3 ? [0,rows.length-1] : [0,Math.floor(rows.length/2),rows.length-1]
  const labels = labelIdx.map((i,k)=>{
    const d = rows[i]
    const anchor = k===0?'start':(k===labelIdx.length-1?'end':'middle')
    const xc = k===0?p.l:(k===labelIdx.length-1?(W-p.r):(W/2))
    return '<text x="'+xc+'" y="'+(H-12)+'" text-anchor="'+anchor+'" fill="#7a7e84" font-family="JetBrains Mono,monospace" font-size="10" letter-spacing="1">'+esc(String(d.date))+'</text>'
  }).join('')
  const dots=points.map(pt=>'<circle class="dot" cx="'+pt[0].toFixed(1)+'" cy="'+pt[1].toFixed(1)+'" r="2.5"/>').join('')
  mount.innerHTML='<svg class="chart" viewBox="0 0 '+W+' '+H+'" role="img" aria-label="Token and cost history chart">'+ticks+bars+'<path class="line" d="'+line+'"/>'+dots+labels+'</svg>'
}

function drawHeatmap(rows){
  const mount = document.getElementById('chartMount')
  document.getElementById('legend').innerHTML = '<span class="heat-legend">less <i style="background:var(--heat-0)"></i><i style="background:var(--heat-1)"></i><i style="background:var(--heat-2)"></i><i style="background:var(--heat-3)"></i><i style="background:var(--heat-4)"></i> more</span>'
  if(!rows.length){mount.innerHTML='<div class="empty">no data in range</div>';return}
  const cell = 14, gap = 3, padTop = 24, padLeft = 30
  const firstDate = new Date(rows[0].date+'T00:00:00')
  const firstWd = firstDate.getDay()
  const totalDays = rows.length
  const totalCells = firstWd + totalDays
  const cols = Math.ceil(totalCells / 7)
  const W = padLeft + cols*(cell+gap) + 12
  const H = padTop + 7*(cell+gap) + 18
  const nz = rows.map(r=>r.total_tokens||0).filter(v=>v>0).sort((a,b)=>a-b)
  const q = p => nz.length ? nz[Math.min(nz.length-1, Math.floor(p*nz.length))] : 0
  const thr = [q(.25), q(.5), q(.75), q(.95)]
  const bucket = v => v<=0 ? 0 : v<=thr[0] ? 1 : v<=thr[1] ? 2 : v<=thr[2] ? 3 : 4
  const fills = ['var(--heat-0)','var(--heat-1)','var(--heat-2)','var(--heat-3)','var(--heat-4)']
  const dayLabels = ['','Mon','','Wed','','Fri','']
  let svg = '<svg class="chart" viewBox="0 0 '+W+' '+H+'" width="'+W+'" height="'+H+'" role="img" aria-label="Yearly heatmap">'
  for (let r=0;r<7;r++){
    if (dayLabels[r]) svg += '<text x="0" y="'+(padTop + r*(cell+gap) + cell - 2)+'" fill="#7a7e84" font-family="JetBrains Mono,monospace" font-size="9" letter-spacing="1">'+dayLabels[r]+'</text>'
  }
  let monthLabels = ''
  let lastMonth = -1
  for (let i=0;i<totalDays;i++){
    const d = rows[i]
    const idx = firstWd + i
    const col = Math.floor(idx/7)
    const row = idx % 7
    const x = padLeft + col*(cell+gap)
    const y = padTop + row*(cell+gap)
    const b = bucket(d.total_tokens||0)
    const tip = d.date+' · '+fmtTokens(d.total_tokens)+' tokens · '+fmtMoney(d.cost_usd)
    svg += '<rect class="heat-cell" x="'+x+'" y="'+y+'" width="'+cell+'" height="'+cell+'" style="fill:'+fills[b]+'"><title>'+esc(tip)+'</title></rect>'
    const m = Number(d.date.slice(5,7))
    if (m !== lastMonth && row === 0){
      const mn = ['Jan','Feb','Mar','Apr','May','Jun','Jul','Aug','Sep','Oct','Nov','Dec'][m-1]
      monthLabels += '<text x="'+x+'" y="'+(padTop-8)+'" fill="#7a7e84" font-family="JetBrains Mono,monospace" font-size="9" letter-spacing="1.2">'+mn+'</text>'
      lastMonth = m
    }
  }
  svg += monthLabels + '</svg>'
  mount.innerHTML = '<div class="heatmap">'+svg+'</div>'
}

function drawDistribution(dim, rows){
  const mountId = dim
  const totals={}
  rows.forEach(d=>{
    const obj = dim === 'sources' ? (d.sources||{}) : (d.devices||{})
    Object.entries(obj).forEach(([name,v])=>{ totals[name]=(totals[name]||0)+(v.total_tokens||0) })
  })
  const max = Math.max(1, ...Object.values(totals))
  const list = Object.entries(totals).sort((a,b)=>b[1]-a[1])
  const mount = document.getElementById(mountId)
  if (!list.length){ mount.innerHTML='<li style="display:block;padding:14px 0;color:var(--muted);font-style:italic;border:none">— no data —</li>'; return }
  const sel = state[dim]
  mount.innerHTML = list.map(([name,val])=>{
    const active = sel.size === 0 || sel.has(name)
    return '<li'+(active?'':' class="dim"')+'><div class="rank-row"><span class="name">'+esc(name)+'</span><span class="num">'+fmtTokens(val)+'</span></div><div class="rank-bar"><span style="width:'+(val/max*100).toFixed(1)+'%"></span></div></li>'
  }).join('')
}

function drawRows(rows){
  const tbody = document.getElementById('dailyRows')
  const recent = rows.slice().reverse()
  const html = []
  recent.forEach(d => {
    const open = state.expanded.has(d.date)
    const compose = ((d.total_tokens||0)
      ? (Object.keys(d.devices||{}).length + ' dev · ' + Object.keys(d.sources||{}).length + ' src')
      : '—')
    html.push('<tr class="daily-row'+(open?' open':'')+'" data-date="'+esc(d.date)+'">'
      + '<td><span class="toggle">▶</span>'+esc(d.date)+'</td>'
      + '<td>'+fmtTokens(d.total_tokens)+'</td>'
      + '<td>'+fmtMoney(d.cost_usd)+'</td>'
      + '<td>'+esc(compose)+'</td>'
      + '</tr>')
    if (open){
      html.push('<tr class="detail-row" data-date="'+esc(d.date)+'"><td colspan="4">'+renderBreakdown(d)+'</td></tr>')
    }
  })
  tbody.innerHTML = html.join('')
}

function renderBreakdown(d){
  const breakdown = d.breakdown || {}
  const devices = Object.keys(breakdown).sort()
  if (!devices.length) return '<div class="detail-wrap" style="color:var(--muted);font-style:italic">no entries on this date</div>'
  const sourceSet = new Set()
  devices.forEach(dev => Object.keys(breakdown[dev]||{}).forEach(s => sourceSet.add(s)))
  const sources = [...sourceSet].sort()
  let html = '<div class="detail-wrap"><h4>Breakdown<span class="stamp">'+esc(d.date)+' · device × source</span></h4><table class="detail">'
  html += '<thead><tr><th>device / source</th>'
  for (const s of sources) html += '<th>'+esc(s)+'</th>'
  html += '<th>total</th></tr></thead><tbody>'
  for (const dev of devices){
    let rowTokens = 0, rowCost = 0
    let cells = ''
    for (const s of sources){
      const c = (breakdown[dev]||{})[s]
      if (c){ rowTokens += c.total_tokens||0; rowCost += c.cost_usd||0 }
      cells += '<td>'+ (c ? fmtTokens(c.total_tokens)+'<span class="sub">'+fmtMoney(c.cost_usd)+'</span>' : '<span style="color:var(--muted)">—</span>') +'</td>'
    }
    html += '<tr><td>'+esc(dev)+'</td>'+cells+'<td>'+fmtTokens(rowTokens)+'<span class="sub">'+fmtMoney(rowCost)+'</span></td></tr>'
  }
  let totRow = '<tr class="totals"><td>total</td>'
  let grandT = 0, grandC = 0
  for (const s of sources){
    let t = 0, c = 0
    for (const dev of devices){
      const cell = (breakdown[dev]||{})[s]
      if (cell){ t += cell.total_tokens||0; c += cell.cost_usd||0 }
    }
    grandT += t; grandC += c
    totRow += '<td>'+fmtTokens(t)+'<span class="sub">'+fmtMoney(c)+'</span></td>'
  }
  totRow += '<td>'+fmtTokens(grandT)+'<span class="sub">'+fmtMoney(grandC)+'</span></td></tr>'
  html += totRow + '</tbody></table></div>'
  return html
}

document.querySelectorAll('[data-days]').forEach(btn=>btn.addEventListener('click',()=>load(Number(btn.dataset.days)).catch(showError)))
document.getElementById('applyDays').addEventListener('click',()=>load(Math.min(365,Math.max(1,Number(document.getElementById('customDays').value)||30))).catch(showError))
document.getElementById('customDays').addEventListener('keydown',e=>{if(e.key==='Enter')document.getElementById('applyDays').click()})
document.getElementById('deviceChips').addEventListener('click', onChipClick)
document.getElementById('sourceChips').addEventListener('click', onChipClick)
document.getElementById('dailyRows').addEventListener('click', e => {
  const tr = e.target.closest('tr.daily-row')
  if (!tr) return
  const d = tr.dataset.date
  if (state.expanded.has(d)) state.expanded.delete(d); else state.expanded.add(d)
  load(state.days).catch(showError)
})

load(state.days).catch(showError)
</script>
</body>
</html>`
