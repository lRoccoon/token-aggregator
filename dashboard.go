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
<title>Token Usage Dashboard</title>
<style>
:root{color-scheme:dark;--bg:#070b19;--panel:rgba(255,255,255,.08);--panel2:rgba(255,255,255,.12);--line:rgba(255,255,255,.14);--text:#eff6ff;--muted:#9fb2d8;--brand:#7c3aed;--brand2:#06b6d4;--good:#22c55e;--warn:#f59e0b;--danger:#fb7185;--shadow:0 24px 80px rgba(0,0,0,.38)}
*{box-sizing:border-box}body{margin:0;min-height:100vh;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI","PingFang SC","Microsoft YaHei",sans-serif;background:radial-gradient(circle at 15% 5%,rgba(124,58,237,.45),transparent 32rem),radial-gradient(circle at 88% 8%,rgba(6,182,212,.30),transparent 28rem),linear-gradient(145deg,#050816,#0f172a 58%,#111827);color:var(--text)}
.shell{width:min(1180px,calc(100vw - 36px));margin:0 auto;padding:34px 0 54px}.hero{display:flex;justify-content:space-between;gap:24px;align-items:flex-start;margin-bottom:26px}.eyebrow{display:inline-flex;gap:8px;align-items:center;padding:7px 12px;border:1px solid var(--line);border-radius:999px;background:rgba(255,255,255,.06);color:#c4b5fd;font-weight:700;font-size:13px}.hero h1{font-size:clamp(34px,5vw,62px);line-height:1;margin:16px 0 12px;letter-spacing:-.05em}.hero p{max-width:680px;margin:0;color:var(--muted);font-size:17px;line-height:1.7}.toolbar{display:flex;flex-wrap:wrap;gap:10px;justify-content:flex-end;align-items:center;margin-top:10px}.pill,.input{border:1px solid var(--line);background:rgba(255,255,255,.08);color:var(--text);border-radius:14px;padding:11px 14px;font-weight:750;box-shadow:inset 0 1px 0 rgba(255,255,255,.08)}button.pill{cursor:pointer;transition:.18s ease}button.pill:hover,button.pill.active{transform:translateY(-1px);background:linear-gradient(135deg,rgba(124,58,237,.75),rgba(6,182,212,.55));border-color:rgba(255,255,255,.32)}.input{width:104px;outline:none}.grid{display:grid;grid-template-columns:repeat(4,1fr);gap:16px;margin:24px 0}.card,.panel{border:1px solid var(--line);background:linear-gradient(180deg,rgba(255,255,255,.11),rgba(255,255,255,.06));backdrop-filter:blur(18px);border-radius:24px;box-shadow:var(--shadow)}.card{padding:20px}.label{color:var(--muted);font-size:13px;font-weight:750;text-transform:uppercase;letter-spacing:.08em}.value{font-size:30px;font-weight:900;margin-top:10px;letter-spacing:-.03em}.sub{color:var(--muted);font-size:13px;margin-top:8px}.panels{display:grid;grid-template-columns:2fr 1fr;gap:16px}.panel{padding:20px;min-width:0}.panel h2{margin:0 0 14px;font-size:18px}.chart{width:100%;height:330px;display:block;overflow:visible}.axis{stroke:rgba(255,255,255,.12);stroke-width:1}.bar{rx:7;transition:.18s}.bar:hover{filter:brightness(1.25)}.line{fill:none;stroke:var(--brand2);stroke-width:3.5;stroke-linecap:round;stroke-linejoin:round}.dot{fill:#67e8f9;stroke:#082f49;stroke-width:2}.legend{display:flex;gap:14px;flex-wrap:wrap;color:var(--muted);font-size:13px;margin-top:8px}.sw{width:10px;height:10px;border-radius:999px;display:inline-block;margin-right:6px}.table-wrap{margin-top:16px;overflow:hidden;border-radius:20px;border:1px solid var(--line)}table{width:100%;border-collapse:collapse;background:rgba(2,6,23,.35)}th,td{padding:13px 15px;border-bottom:1px solid rgba(255,255,255,.09);text-align:right}th:first-child,td:first-child{text-align:left}th{font-size:12px;text-transform:uppercase;letter-spacing:.08em;color:#bfdbfe;background:rgba(255,255,255,.07)}td{font-variant-numeric:tabular-nums;color:#dbeafe}tr:hover td{background:rgba(255,255,255,.045)}.sources{display:grid;gap:10px}.source{padding:13px;border:1px solid var(--line);border-radius:16px;background:rgba(255,255,255,.055)}.source-top{display:flex;justify-content:space-between;gap:8px;font-weight:800}.meter{height:8px;border-radius:999px;background:rgba(255,255,255,.09);overflow:hidden;margin-top:10px}.meter span{display:block;height:100%;border-radius:999px;background:linear-gradient(90deg,var(--brand),var(--brand2))}.status{margin-top:14px;color:var(--muted);font-size:13px}.error{color:#fecdd3}.empty{height:330px;display:grid;place-items:center;color:var(--muted);border:1px dashed var(--line);border-radius:20px}@media(max-width:920px){.hero{display:block}.toolbar{justify-content:flex-start}.grid{grid-template-columns:repeat(2,1fr)}.panels{grid-template-columns:1fr}}@media(max-width:560px){.grid{grid-template-columns:1fr}.shell{width:min(100vw - 22px,1180px)}th,td{padding:10px 8px;font-size:13px}}
</style>
</head>
<body>
<main class="shell">
  <section class="hero">
    <div>
      <span class="eyebrow">✦ Token Aggregator · Dashboard</span>
      <h1>Token 用量看板</h1>
      <p>查看过去 N 天的 token 消耗、cost 趋势和来源分布。支持最近一周、一月、半年、一年快速筛选。</p>
    </div>
    <div class="toolbar" aria-label="time range filters">
      <button class="pill" data-days="7">最近一周</button>
      <button class="pill active" data-days="30">最近一月</button>
      <button class="pill" data-days="180">最近半年</button>
      <button class="pill" data-days="365">最近一年</button>
      <input class="input" id="authToken" type="password" placeholder="Bearer token" title="Bearer token" style="width:150px">
      <input class="input" id="customDays" type="number" min="1" max="365" value="30" title="自定义天数">
      <button class="pill" id="applyDays">应用</button>
    </div>
  </section>

  <section class="grid">
    <article class="card"><div class="label">总 Token</div><div class="value" id="totalTokens">--</div><div class="sub" id="rangeLabel">--</div></article>
    <article class="card"><div class="label">总 Cost</div><div class="value" id="totalCost">--</div><div class="sub">按已入库 cost 汇总</div></article>
    <article class="card"><div class="label">日均 Token</div><div class="value" id="avgTokens">--</div><div class="sub">窗口内平均值</div></article>
    <article class="card"><div class="label">日均 Cost</div><div class="value" id="avgCost">--</div><div class="sub">窗口内平均值</div></article>
  </section>

  <section class="panels">
    <article class="panel">
      <h2>Tokens & Cost Chart</h2>
      <div id="chartMount"></div>
      <div class="legend"><span><i class="sw" style="background:linear-gradient(90deg,var(--brand),var(--brand2))"></i>Token bars</span><span><i class="sw" style="background:var(--brand2)"></i>Cost line</span></div>
      <div class="status" id="status">正在加载数据…</div>
    </article>
    <aside class="panel">
      <h2>来源分布</h2>
      <div class="sources" id="sources"></div>
    </aside>
  </section>

  <section class="panel" style="margin-top:16px">
    <h2>每日明细</h2>
    <div class="table-wrap"><table><thead><tr><th>日期</th><th>Tokens</th><th>Cost</th><th>Sources</th></tr></thead><tbody id="dailyRows"></tbody></table></div>
  </section>
</main>
<script>
const fmtTokens = n => Intl.NumberFormat('en', {notation:n>=10000?'compact':'standard', maximumFractionDigits:1}).format(n || 0)
const fmtMoney = n => '$' + Number(n || 0).toFixed(4).replace(/0+$/,'').replace(/\.$/,'.00')
const esc = v => String(v ?? '').replace(/[&<>'"]/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;',"'":'&#39;','"':'&quot;'}[c]))
const qs = new URLSearchParams(location.search)
let currentDays = Math.min(365, Math.max(1, Number(qs.get('days')) || 30))
let token = qs.get('token') || sessionStorage.getItem('tokenAggregatorToken') || ''
const authInput = document.getElementById('authToken')
authInput.value = token
if (qs.get('token')) {
  sessionStorage.setItem('tokenAggregatorToken', qs.get('token'))
  qs.delete('token')
  history.replaceState(null,'','?'+qs.toString())
}

function setActive(days){document.querySelectorAll('[data-days]').forEach(b=>b.classList.toggle('active', Number(b.dataset.days)===days));document.getElementById('customDays').value=days}
async function load(days){
  currentDays = days; setActive(days)
  const status = document.getElementById('status')
  status.textContent = '正在加载数据…'
  status.className = 'status'
  token = authInput.value.trim()
  if (token) sessionStorage.setItem('tokenAggregatorToken', token); else sessionStorage.removeItem('tokenAggregatorToken')
  const headers = token ? {Authorization:'Bearer '+token} : {}
  const params = new URLSearchParams({days: String(days)})
  if (qs.get('today')) params.set('today', qs.get('today'))
  const res = await fetch('/history?'+params.toString(), {headers})
  if(!res.ok){status.textContent='加载失败：'+res.status+' '+await res.text();status.className='status error';return}
  const data = await res.json(); render(data)
  const next = new URLSearchParams({days: String(days)})
  if (qs.get('today')) next.set('today', qs.get('today'))
  history.replaceState(null,'','?'+next.toString())
}
function render(data){
  document.getElementById('totalTokens').textContent = fmtTokens(data.summary.total_tokens)
  document.getElementById('totalCost').textContent = fmtMoney(data.summary.total_cost_usd)
  document.getElementById('avgTokens').textContent = fmtTokens(data.summary.avg_tokens_per_day)
  document.getElementById('avgCost').textContent = fmtMoney(data.summary.avg_cost_usd_per_day)
  document.getElementById('rangeLabel').textContent = data.from + ' → ' + data.to + ' · ' + data.days + ' 天'
  document.getElementById('status').textContent = '已更新：' + new Date().toLocaleString()
  drawChart(data.daily || [])
  drawSources(data.daily || [])
  drawRows(data.daily || [])
}
function drawChart(rows){
  const mount = document.getElementById('chartMount')
  if(!rows.length){mount.innerHTML='<div class="empty">暂无数据</div>';return}
  const W=920,H=330,p={l:42,r:28,t:18,b:42},cw=W-p.l-p.r,ch=H-p.t-p.b
  const maxTokens=Math.max(1,...rows.map(d=>d.total_tokens||0)), maxCost=Math.max(.000001,...rows.map(d=>d.cost_usd||0))
  const step=cw/rows.length, barW=Math.max(2,Math.min(22,step*.62))
  const points=[]
  const bars=rows.map((d,i)=>{const x=p.l+i*step+(step-barW)/2;const h=(d.total_tokens||0)/maxTokens*ch;const y=p.t+ch-h;const cy=p.t+ch-((d.cost_usd||0)/maxCost*ch);points.push([p.l+i*step+step/2,cy]);return '<rect class="bar" x="'+x.toFixed(1)+'" y="'+y.toFixed(1)+'" width="'+barW.toFixed(1)+'" height="'+Math.max(1,h).toFixed(1)+'" fill="url(#barGrad)"><title>'+esc(d.date)+'\n'+fmtTokens(d.total_tokens)+' tokens\n'+fmtMoney(d.cost_usd)+'</title></rect>'}).join('')
  const line=points.map((pt,i)=>(i?'L':'M')+pt[0].toFixed(1)+' '+pt[1].toFixed(1)).join(' ')
  const ticks=[0,.25,.5,.75,1].map(t=>'<line class="axis" x1="'+p.l+'" x2="'+(W-p.r)+'" y1="'+(p.t+ch-t*ch)+'" y2="'+(p.t+ch-t*ch)+'"/>').join('')
  const labels=rows.filter((_,i)=>i===0||i===rows.length-1||i===Math.floor(rows.length/2)).map((d,i)=>'<text x="'+(i===0?p.l:i===1?W/2:W-p.r)+'" y="'+(H-12)+'" text-anchor="'+(i===0?'start':i===1?'middle':'end')+'" fill="#9fb2d8" font-size="12">'+esc(String(d.date).slice(5))+'</text>').join('')
  const dots=points.map(pt=>'<circle class="dot" cx="'+pt[0].toFixed(1)+'" cy="'+pt[1].toFixed(1)+'" r="3"/>').join('')
  mount.innerHTML='<svg class="chart" viewBox="0 0 '+W+' '+H+'" role="img" aria-label="Token and cost history chart"><defs><linearGradient id="barGrad" x1="0" x2="0" y1="0" y2="1"><stop offset="0" stop-color="#22d3ee"/><stop offset="1" stop-color="#7c3aed"/></linearGradient></defs>'+ticks+bars+'<path class="line" d="'+line+'"/>'+dots+labels+'</svg>'
}
function drawSources(rows){
  const totals={}; rows.forEach(d=>Object.entries(d.sources||{}).forEach(([name,v])=>{totals[name]=(totals[name]||0)+(v.total_tokens||0)}))
  const max=Math.max(1,...Object.values(totals)); const list=Object.entries(totals).sort((a,b)=>b[1]-a[1])
  document.getElementById('sources').innerHTML = list.length ? list.map(([name,val])=>'<div class="source"><div class="source-top"><span>'+esc(name)+'</span><span>'+fmtTokens(val)+'</span></div><div class="meter"><span style="width:'+(val/max*100).toFixed(1)+'%"></span></div></div>').join('') : '<div class="empty" style="height:180px">暂无来源数据</div>'
}
function drawRows(rows){
  document.getElementById('dailyRows').innerHTML = rows.slice().reverse().map(d=>'<tr><td>'+esc(d.date)+'</td><td>'+fmtTokens(d.total_tokens)+'</td><td>'+fmtMoney(d.cost_usd)+'</td><td>'+esc(Object.keys(d.sources||{}).join(', ') || '—')+'</td></tr>').join('')
}
document.querySelectorAll('[data-days]').forEach(btn=>btn.addEventListener('click',()=>load(Number(btn.dataset.days))))
document.getElementById('applyDays').addEventListener('click',()=>load(Math.min(365,Math.max(1,Number(document.getElementById('customDays').value)||30))))
document.getElementById('customDays').addEventListener('keydown',e=>{if(e.key==='Enter')document.getElementById('applyDays').click()})
load(currentDays).catch(err=>{const s=document.getElementById('status');s.textContent='加载失败：'+err.message;s.className='status error'})
</script>
</body>
</html>`
