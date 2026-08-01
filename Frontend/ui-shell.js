window.blsReady = (async () => {
  for (const url of ['https://esm.sh/@noble/curves@1.4.2/bls12-381','https://cdn.skypack.dev/@noble/curves@1.4.2/bls12-381']) {
    try { const m = await import(url); bls12_381 = m.bls12_381; break; } catch {}
  }
  if (!bls12_381) toast('BLS library failed to load — check internet', true);
  return bls12_381;
})();
window.currentHeight = 0;
window.currentNetworkID = 1;
window.currentChainID   = 266;
window.toast=function(msg,isErr=false){
  const el=document.getElementById('toast');
  el.textContent=msg;el.className=isErr?'err':'ok';el.style.display='block';
  clearTimeout(_tt);_tt=setTimeout(()=>el.style.display='none',5000);
};
window.showPage=function(id,btn,skipPush){
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  document.getElementById('page-'+id).classList.add('active');
  document.querySelectorAll('#deskNav .ni').forEach(b=>b.classList.remove('active'));
  const dm=document.querySelector(`#deskNav [data-p="${id}"]`);if(dm)dm.classList.add('active');
  document.querySelectorAll('#bnav .btab').forEach(b=>b.classList.remove('active'));
  const bm=document.querySelector(`#bnav [data-p="${id}"]`);if(bm)bm.classList.add('active');
  if(id==='markets')loadMarkets();
  if(id==='profile'){refreshBalance();loadMyPredictions();}
  if(id==='create'){updateCreateBreakdown();setTimeout(initExpiryDate,50);}
  if(id==='predict')updatePredictBreakdown();
  if(id==='resolvers')loadResolvers();
  if(id==='unstake-resolver')renderMyResolverStatus('unstake');
  if(id==='claim-unbonded')renderMyResolverStatus('claim-unbonded');
  closeNav();
  setTimeout(wireCopyBtns, 50);
  if(!skipPush){
    const _path = id === 'markets' ? '/' : '/' + id;
    if(location.pathname !== _path) history.pushState({page:id}, '', _path);
  }
};
window.openNav=function(){document.getElementById('deskNav').classList.add('open');document.getElementById('mobNav').classList.add('open');};window.closeNav=function(e){if(!e||e.target===document.getElementById('mobNav')||e.currentTarget===document.getElementById('mobNav')){document.getElementById('deskNav').classList.remove('open');document.getElementById('mobNav').classList.remove('open');}};
window.toggleTheme=function(){
  const html=document.documentElement;
  const d=html.getAttribute('data-theme')==='dark';
  html.setAttribute('data-theme',d?'light':'dark');
  localStorage.setItem('praxis_theme',d?'light':'dark');
  updateTL();
};
window.setOut=function(v){selectedOut=v;document.getElementById('btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('btn_no').className='obtn no'+(!v?' active':'');};
window.setPropOut=function(v){propOut=v;document.getElementById('pr_btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('pr_btn_no').className='obtn no'+(!v?' active':'');};
window.setRevOut=function(v){revOut=v;document.getElementById('rv_btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('rv_btn_no').className='obtn no'+(!v?' active':'');};
window.loadMyPredictions = async function () {
  const el = document.getElementById('myPredictions');
  if (!signerAddress) {
    el.innerHTML = '<div style="font-family:JetBrains Mono,monospace;font-size:10px;color:var(--text3)">Load wallet to see predictions</div>';
    return;
  }
  el.innerHTML = '<div style="padding:12px;color:var(--text3);font-family:JetBrains Mono,monospace;font-size:10px"><span class="blink">▪▪▪</span> loading predictions</div>';
  try {
    const data = await rpc('/v1/query/txs-by-sender', { address: signerAddress, perPage: 200 });
    const results = data.results || [];
    const seen = {};
    const predictions = [];

    for (const tx of results) {
      const t = tx.transaction || tx;
      const type = t.type || t.messageType || '';
      if (type !== 'submit_prediction') continue;
      const msg = t.msg || t;
      let marketId = '', outcome = false, shares = 0n, maxCost = 0n;
      if (t.msgBytes) {
        const bytes = h2b(t.msgBytes);
        let pos = 0;
        while (pos < bytes.length) {
          const { v: tagV, p: p1 } = decVarint(bytes, pos); pos = p1;
          const fn = Number(tagV >> 3n), wt = Number(tagV & 7n);
          if (fn === 3 && wt === 0) { const { v, p: p2 } = decVarint(bytes, pos); pos = p2; outcome = v === 1n; }
          else if (wt === 0) { const { v: _, p: p2 } = decVarint(bytes, pos); pos = p2; if (fn === 4) shares = _; if (fn === 5) maxCost = _; }
          else if (wt === 2) { const { v: lenV, p: p2 } = decVarint(bytes, pos); pos = p2 + Number(lenV); if (fn === 1) marketId = b2h(bytes.slice(p2 - Number(lenV), pos)); }
          else if (wt === 1) { pos += 8; } else if (wt === 5) { pos += 4; } else break;
        }
      } else {
        marketId = msg.marketId || '';
        outcome = msg.outcome === true || msg.outcome === 'true' || msg.outcome === 1;
        shares = BigInt(msg.shares || 0);
        maxCost = BigInt(msg.maxCost || msg.max_cost || 0);
      }
      const key = marketId || tx.txHash;
      if (!seen[key]) {
        seen[key] = true;
        predictions.push({ marketId: marketId || tx.txHash, outcome, shares, maxCost, height: tx.height || 0 });
      }
    }

    if (predictions.length === 0) {
      el.innerHTML = '<div style="padding:12px;color:var(--text3);font-family:JetBrains Mono,monospace;font-size:10px">No predictions yet</div>';
      return;
    }

    el.innerHTML = predictions.map(p => {
      const m = window._allMarkets.find(x => x.id === p.marketId);
      let payoutHtml = '';
      if (m && m.status === 6) {
        // finalized — compute expected payout
        const totalPool = m.qYes + m.qNo;
        const winPool   = p.outcome ? m.qYes : m.qNo;
        const won       = m.proposedOutcome === p.outcome;
        if (won && winPool > 0n) {
          const payout = totalPool * p.shares / winPool;
          payoutHtml = '<div style="margin-top:6px;font-family:JetBrains Mono,monospace;font-size:10px;color:var(--green)">✓ Est. payout: ' + fmtPRX(payout) + ' PRX</div>';
        } else if (!won) {
          payoutHtml = '<div style="margin-top:6px;font-family:JetBrains Mono,monospace;font-size:10px;color:var(--red)">✗ Lost</div>';
        }
      } else if (m && m.status === 4) {
        payoutHtml = '<div style="margin-top:6px;font-family:JetBrains Mono,monospace;font-size:10px;color:var(--text3)">⏳ Awaiting finalization</div>';
      }
      return '<div style="background:var(--bg);border:1px solid var(--border);padding:12px;margin-bottom:8px">' +
        '<div style="display:flex;justify-content:space-between;align-items:center">' +
          '<div>' +
            '<div style="font-family:JetBrains Mono,monospace;font-size:10px;color:var(--text3);margin-bottom:4px">MKT ' + p.marketId.slice(0,12) + '…</div>' +
            '<div style="display:flex;gap:12px">' +
              '<span style="font-family:JetBrains Mono,monospace;font-size:11px;color:' + (p.outcome ? 'var(--green)' : 'var(--red)') + '">' + (p.outcome ? 'YES' : 'NO') + '</span>' +
              '<span style="font-family:JetBrains Mono,monospace;font-size:11px;color:var(--text2)">Shares: ' + fmtPRX(p.shares) + '</span>' +
              '<span style="font-family:JetBrains Mono,monospace;font-size:11px;color:var(--text2)">Max: ' + fmtPRX(p.maxCost) + ' PRX</span>' +
            '</div>' +
          '</div>' +
          '<span style="font-family:JetBrains Mono,monospace;font-size:9px;color:var(--text3)">#' + p.height + '</span>' +
        '</div>' +
        payoutHtml +
      '</div>';
    }).join('');
  } catch (e) {
    el.innerHTML = '<div style="padding:12px;color:var(--red);font-family:JetBrains Mono,monospace;font-size:10px">Error: ' + esc(e.message) + '</div>';
  }
};
window._allMarkets = [];
window.openDetail = window.showDetail;
window._activeTab = 'live';
window.switchTab = function(tab) {
  window._activeTab = tab;
  window._activeTab = tab;
  document.querySelectorAll('.mtab').forEach(b => b.classList.remove('active'));
  const btn = document.getElementById('tab-' + tab);
  if (btn) btn.classList.add('active');
  renderCurrentTab();
};
window.updateCreateBreakdown=function(){
  const b0=parseInt(document.getElementById('c_b0')?.value||0);
  const fee=parseInt(document.getElementById('c_fee')?.value||10000);
  const bond=5000;
  const total=b0+bond+(fee/1000000);
  const el=document.getElementById('create_breakdown');
  if(!el)return;
  el.innerHTML=
    '<div class="cm-row"><span class="cm-l">B0 liquidity seed</span><span class="cm-v g">'+b0.toLocaleString()+' PRX</span></div>'+
    '<div class="cm-row"><span class="cm-l">Creator bond (locked)</span><span class="cm-v">5,000 PRX</span></div>'+
    '<div class="cm-row"><span class="cm-l">TX fee</span><span class="cm-v">'+fee.toLocaleString()+' uPRX</span></div>'+
    '<div class="cm-row" style="border-top:1px solid var(--border2);margin-top:4px"><span class="cm-l" style="color:var(--text)">Total deducted</span><span class="cm-v g" style="font-size:13px">'+(b0+bond).toLocaleString()+' PRX</span></div>';
};
window.updatePredictBreakdown=function(){
  const shares=parseInt(document.getElementById('p_shares')?.value||0);
  const fee=parseInt(document.getElementById('p_fee')?.value||10000);
  const slipPct=parseFloat(document.getElementById('p_slippage')?.value||5);
  const el=document.getElementById('predict_breakdown');
  const slipLbl=document.getElementById('p_slip_lbl');
  if(slipLbl)slipLbl.textContent=slipPct.toFixed(1)+'%';
  if(!el)return;
  const tradeCost=shares;
  const creatorFee=Math.ceil(shares*0.01);
  const resolverFee=Math.ceil(shares*0.01);
  const total=tradeCost+creatorFee+resolverFee;
  const maxCost=Math.ceil(total*(1+slipPct/100));
  const mcEl=document.getElementById('p_maxcost');
  if(mcEl)mcEl.value=maxCost;
  el.innerHTML=
    '<div class="cm-row"><span class="cm-l">Trade cost</span><span class="cm-v g">'+tradeCost.toLocaleString()+' PRX</span></div>'+
    '<div class="cm-row"><span class="cm-l">Market fee (2%)</span><span class="cm-v" title="Creator fee 1% + Resolver fee 1%">'+(creatorFee+resolverFee).toLocaleString()+' PRX</span></div>'+
    '<div class="cm-row"><span class="cm-l">TX fee</span><span class="cm-v">'+fee.toLocaleString()+' uPRX</span></div>'+
    '<div class="cm-row" style="border-top:1px solid var(--border2);margin-top:4px"><span class="cm-l" style="color:var(--text)">Max cost ('+slipPct.toFixed(1)+'% slippage)</span><span class="cm-v g" style="font-size:13px">'+maxCost.toLocaleString()+' PRX</span></div>';
};
window.copyText = async function(text, btn) {
  try {
    await navigator.clipboard.writeText(text);
    if (btn) { btn.textContent = '✓'; btn.classList.add('ok'); setTimeout(() => { btn.textContent = '⎘'; btn.classList.remove('ok'); }, 1800); }
    toast('Copied');
  } catch { toast('Copy failed', true); }
};
window.closeConfirm = function() {
  document.getElementById('confOverlay').classList.remove('open');
  if (_confirmResolve) { _confirmResolve(false); _confirmResolve = null; }
};
window.clearScanCache = function() {
  localStorage.removeItem('praxis_tx_cache');
  localStorage.removeItem('praxis_scan_height');
  toast('Cache cleared — full rescan on next refresh');
  loadMarkets();
};
window.checkPositionCap = async function() {
  const mid    = document.getElementById('p_mid').value.trim().toLowerCase();
  const bettor = document.getElementById('p_bettor').value.trim().toLowerCase();
  const mc     = (parseInt(document.getElementById('p_maxcost').value) || 0)*1000000;
  const capEl  = document.getElementById('cap_indicator');
  const btn    = document.getElementById('btn_predict');

  if (!capEl) return;
  if (!mid || mid.length !== 40 || !bettor || bettor.length !== 40) {
    capEl.style.display = 'none';
    return;
  }

  // find market in window._allMarkets
  const m = window._allMarkets.find(x => x.marketId === mid || x.txHash === mid);
  if (!m) { capEl.style.display = 'none'; return; }

  const pool = Number(m.qYes + m.qNo);
  const cap  = Math.floor(pool * 2000 / 10000); // 20%

  // try to get user's current cost paid from chain
  let costPaid = 0;
  try {
    const d = await rpc('/v1/query/account', { address: bettor });
    // costPaid not available without plugin query — use 0 for now
    costPaid = 0;
  } catch {}

  const newTotal = costPaid + mc;
  const remaining = cap - costPaid;
  const pct = pool > 0 ? Math.round((newTotal / pool) * 100) : 0;
  const over = newTotal > cap;

  capEl.style.display = '';
  if (over) {
    capEl.style.background = 'rgba(255,61,90,.08)';
    capEl.style.border = '1px solid rgba(255,61,90,.3)';
    capEl.style.color = 'var(--red)';
    capEl.textContent = '⚠ Exceeds 20% position cap — max ' + fmtPRX(remaining) + ' PRX remaining';
    if (btn) btn.setAttribute('disabled', '');
  } else {
    capEl.style.background = 'rgba(0,232,122,.05)';
    capEl.style.border = '1px solid rgba(0,232,122,.15)';
    capEl.style.color = 'var(--text2)';
    capEl.textContent = 'Position: ' + fmtPRX(newTotal) + ' PRX / Cap: ' + fmtPRX(cap) + ' PRX (' + pct + '% of pool)';
    if (btn) btn.removeAttribute('disabled');
  }
};

// ═══════════════════════════════════════════
// FORFEIT POSITION
// ═══════════════════════════════════════════



window.updateMinBondHint = function() {
  const mid = document.getElementById('pr_mid').value.trim().toLowerCase();
  const hint = document.getElementById('prop_bond_hint');
  const bondEl = document.getElementById('pr_bond');
  if (!hint) return;
  if (!mid || mid.length !== 40) {
    hint.textContent = 'Enter Market ID to compute min bond';
    hint.style.color = '';
    return;
  }
  const m = window._allMarkets.find(x => x.marketId === mid || x.txHash === mid);
  if (!m) {
    hint.textContent = 'Market not found in cache — browse Markets first';
    hint.style.color = 'var(--red)';
    return;
  }
  // BEff = current pool size (qYes + qNo)
  const beff = Number(m.qYes + m.qNo) / 1_000_000;
  const onePct = beff * 0.01;
  const minBond = Math.max(onePct, 60);
  hint.textContent = 'Min bond: ' + minBond.toFixed(2) + ' PRX  (max(1% of pool, 60 PRX) — deducted from resolver stake)';
  hint.style.color = 'var(--amber)';
  const bondEl2 = document.getElementById('pr_bond'); if (bondEl2 && parseFloat(bondEl2.value) < minBond) bondEl2.value = Math.ceil(minBond);
};
checkSavedKeystore();


// ═══════════════════════════════════════════
// ELEVATED RISK / PANEL SIZE INDICATOR
// ═══════════════════════════════════════════
const ELEVATED_RISK_THRESHOLD = 25_000_000_000n; // 25,000 PRX in uPRX
const STANDARD_PANEL_SIZE = 5;
const ELEVATED_PANEL_SIZE = 7;

function getRiskInfo(mid) {
  if (!mid || mid.length !== 40) return null;
  const m = (window._allMarkets || []).find(x => x.marketId === mid || x.txHash === mid);
  if (!m) return null;
  const pool = m.qYes + m.qNo;
  const elevated = pool >= ELEVATED_RISK_THRESHOLD;
  const poolPRX = (Number(pool) / 1_000_000).toFixed(2);
  return { elevated, pool, poolPRX, panelSize: elevated ? ELEVATED_PANEL_SIZE : STANDARD_PANEL_SIZE };
}

function renderRiskBox(boxEl, info) {
  if (!info) {
    boxEl.style.display = 'none';
    return;
  }
  const { elevated, poolPRX, panelSize } = info;
  boxEl.style.display = '';
  if (elevated) {
    boxEl.style.background = 'rgba(255,64,96,.08)';
    boxEl.style.border = '1px solid rgba(255,64,96,.25)';
    boxEl.style.color = 'var(--red)';
    boxEl.innerHTML = '⚠ ELEVATED RISK MARKET<br>Pool: ' + poolPRX + ' PRX (&gt;= 25,000 PRX threshold)<br>Panel size: <b>' + panelSize + ' resolvers</b> (extended panel)';
  } else {
    boxEl.style.background = 'var(--gdim)';
    boxEl.style.border = '1px solid rgba(0,232,122,.15)';
    boxEl.style.color = 'var(--text2)';
    boxEl.innerHTML = '✓ Standard market<br>Pool: ' + poolPRX + ' PRX<br>Panel size: <b>' + panelSize + ' resolvers</b> (standard panel)';
  }
}

window.updateProposeRisk = function() {
  const mid = (document.getElementById('pr_mid')?.value || '').trim().toLowerCase();
  const box = document.getElementById('pr_risk_box');
  if (!box) return;
  renderRiskBox(box, getRiskInfo(mid));
  // also update bond hint
  if (typeof updateMinBondHint === 'function') updateMinBondHint();
};

window.updateDisputeRisk = function() {
  const mid = (document.getElementById('di_mid')?.value || '').trim().toLowerCase();
  const box = document.getElementById('di_risk_box');
  if (!box) return;
  renderRiskBox(box, getRiskInfo(mid));
};




// ═══════════════════════════════════════════
// MARKET BANNER IMAGE SYSTEM
// ═══════════════════════════════════════════

function mkCardIcon(rules) {
  const u = extractImg(rules||'');
  if (u) {
    return '<div class="mcard-icon-wrap"><img class="mcard-icon" src="' + u + '" alt="" onerror="this.parentElement.innerHTML=\'\u25c8\';this.parentElement.classList.add(\'mcard-icon-empty\')"></div>';
  }
  return '<div class="mcard-icon-wrap mcard-icon-empty">\u25c8</div>';
}

function extractImg(rules) {
  if (!rules) return '';
  const m = rules.match(/\[IMG:([^\]]+)\]/);
  return m ? m[1].trim() : '';
}

function stripImgTag(rules) {
  if (!rules) return '';
  return rules.replace(/\[IMG:[^\]]+\]\s*/g, '').trim();
}

function buildRulesWithImg(rules, imgUrl) {
  const stripped = stripImgTag(rules);
  if (!imgUrl) return stripped;
  return stripped + (stripped ? ' ' : '') + '[IMG:' + imgUrl.trim() + ']';
}

window.handleImageUpload = async function(input) {
  const file = input.files && input.files[0];
  const hint = document.getElementById('c_img_hint');
  const preview = document.getElementById('c_img_preview');
  const img = document.getElementById('c_img_preview_img');
  const hidden = document.getElementById('c_img');
  if (!file) return;

  const MAX_SIZE = 5 * 1024 * 1024;
  if (file.size > MAX_SIZE) {
    if (hint) { hint.textContent = '✗ Image exceeds 5MB limit'; hint.style.color = 'var(--red)'; }
    input.value = '';
    return;
  }
  const allowed = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'];
  if (!allowed.includes(file.type)) {
    if (hint) { hint.textContent = '✗ Unsupported file type — use PNG, JPEG, WEBP, or GIF'; hint.style.color = 'var(--red)'; }
    input.value = '';
    return;
  }

  // instant local preview while upload is in flight
  const localUrl = URL.createObjectURL(file);
  if (preview && img) {
    img.src = localUrl;
    preview.style.display = '';
  }
  if (hint) { hint.textContent = 'Uploading…'; hint.style.color = ''; }

  try {
    const res = await fetch('/api/upload-image', {
      method: 'POST',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Upload failed');

    if (hidden) hidden.value = data.url;
    if (hint) { hint.textContent = '✓ Image uploaded'; hint.style.color = 'var(--green)'; }
  } catch (e) {
    if (hint) { hint.textContent = '✗ ' + e.message; hint.style.color = 'var(--red)'; }
    if (preview) preview.style.display = 'none';
    if (hidden) hidden.value = '';
  } finally {
    URL.revokeObjectURL(localUrl);
  }
};

window.previewBanner = function() {
  const url = (document.getElementById('c_img')?.value || '').trim();
  const preview = document.getElementById('c_img_preview');
  const img = document.getElementById('c_img_preview_img');
  const hint = document.getElementById('c_img_hint');
  if (!preview || !img) return;
  if (url && (url.startsWith('http') || url.startsWith('ipfs'))) {
    img.src = url;
    preview.style.display = '';
    img.onload = () => { if (hint) { hint.textContent = '✓ Image loaded'; hint.style.color = 'var(--green)'; } };
    img.onerror = () => {
      preview.style.display = 'none';
      if (hint) { hint.textContent = '✗ Could not load image — check URL or CORS policy'; hint.style.color = 'var(--red)'; }
    };
  } else {
    preview.style.display = 'none';
    if (hint) { hint.textContent = 'Image will be stored on-chain via IPFS or direct URL. Recommended: 16:9, min 800x450px.'; hint.style.color = ''; }
  }
};

// ═══════════════════════════════════════════
// EXPIRY DATE → BLOCK HEIGHT CONVERTER
// ═══════════════════════════════════════════
const BLOCK_TIME_MS = 5000; // 5s per block

function blocksFromNow(ms) {
  return Math.ceil(ms / BLOCK_TIME_MS);
}

function fmtDuration(ms) {
  const s = Math.floor(ms / 1000);
  const d = Math.floor(s / 86400);
  const h = Math.floor((s % 86400) / 3600);
  const m = Math.floor((s % 3600) / 60);
  if (d > 0) return d + 'd ' + h + 'h';
  if (h > 0) return h + 'h ' + m + 'm';
  return m + 'm';
}

window.updateExpiryFromDate = function() {
  const dtEl   = document.getElementById('c_expiry_dt');
  const hidden = document.getElementById('c_expiry');
  const hint   = document.getElementById('c_expiry_hint');
  if (!dtEl || !hidden || !hint) return;

  const val = dtEl.value;
  if (!val) {
    hint.textContent = 'Select a date to compute block height';
    hint.style.color = '';
    hidden.value = '';
    return;
  }

  const targetMs = new Date(val).getTime();
  const nowMs    = Date.now();
  const diffMs   = targetMs - nowMs;

  if (diffMs <= 0) {
    hint.textContent = 'Date must be in the future';
    hint.style.color = 'var(--red)';
    hidden.value = '';
    return;
  }

  const blocksNeeded = blocksFromNow(diffMs);
  const blockHeight  = window.currentHeight + blocksNeeded;
  hidden.value = blockHeight;

  const dur = fmtDuration(diffMs);
  hint.textContent = 'Block #' + blockHeight + '  (~' + dur + ' from now, ' + blocksNeeded + ' blocks)';
  hint.style.color = blocksNeeded < 100 ? 'var(--red)' : 'var(--amber)';
};

// Set default expiry to 7 days from now when page loads
window.initExpiryDate = function() {
  const dtEl = document.getElementById('c_expiry_dt');
  if (!dtEl || dtEl.value) return;
  const d = new Date(Date.now() + 7 * 24 * 3600 * 1000);
  // Format: YYYY-MM-DDTHH:MM
  const pad = n => String(n).padStart(2, '0');
  dtEl.value = d.getFullYear() + '-' + pad(d.getMonth()+1) + '-' + pad(d.getDate()) + 'T' + pad(d.getHours()) + ':' + pad(d.getMinutes());
  updateExpiryFromDate();
};

// ═══════════════════════════════════════════
// CATEGORY SYSTEM
// ═══════════════════════════════════════════
const CAT_LABELS = {
  crypto: '🪙 Crypto', sports: '⚽ Sports', politics: '🗳 Politics',
  finance: '📈 Finance', other: '◈ Other'
};

function extractOutcomes(rules) {
  if (!rules) return { yes: 'YES', no: 'NO' };
  const m = rules.match(/\[OUT:([^\|\]]+)\|([^\]]+)\]/);
  if (!m) return { yes: 'YES', no: 'NO' };
  return { yes: m[1].trim(), no: m[2].trim() };
}

function stripOutcomesTag(rules) {
  if (!rules) return '';
  return rules.replace(/\[OUT:[^\]]+\]\s*/g, '').trim();
}

function buildRulesWithOutcomes(rules, yesLabel, noLabel) {
  const stripped = stripOutcomesTag(rules);
  const yl = (yesLabel || '').trim();
  const nl = (noLabel || '').trim();
  if (!yl || !nl || (yl.toUpperCase() === 'YES' && nl.toUpperCase() === 'NO')) {
    return stripped;
  }
  return stripped + (stripped ? ' ' : '') + '[OUT:' + yl + '|' + nl + ']';
}

function extractCat(rules) {
  if (!rules) return 'other';
  const m = rules.match(/^\[CAT:(\w+)\]/);
  return m ? m[1] : 'other';
}

function stripCatPrefix(rules) {
  if (!rules) return '';
  return rules.replace(/^\[CAT:\w+\]\s*/, '');
}

function buildRulesWithCat(cat, rules) {
  const stripped = stripCatPrefix(rules);
  return '[CAT:' + cat + '] ' + stripped;
}

window.pickCat = function(el) {
  document.querySelectorAll('#c_cat_pick .cpick').forEach(e => e.classList.remove('active'));
  el.classList.add('active');
};



// ═══════════════════════════════════════════
// CLAIM CREATOR FEE
// ═══════════════════════════════════════════


// ═══════════════════════════════════════════
// CANCEL MARKET
// ═══════════════════════════════════════════


// ═══════════════════════════════════════════
// UNSTAKE RESOLVER
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// CLAIM UNBONDED STAKE
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// RESOLVER RECORD STATE QUERY
// prefix 0x16 + len + addr bytes
// ═══════════════════════════════════════════



// ═══════════════════════════════════════════
// BROWSE RESOLVERS
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// RESOLVER SELF-STATUS (stake / unbonding / claimable)
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// MARKET DETAIL — ACTIVITY FEED + TOP HOLDERS
// ═══════════════════════════════════════════


// ═══════════════════════════════════════════
// PRIS REWARD PAGES
// ═══════════════════════════════════════════

const EPOCH_BLOCKS = 1000;
const AUTHORIZED_BUILDER   = '954378ba109c5ca45b23bfa284f3ac70e2671b87';
const AUTHORIZED_COMMUNITY = '15e658698d2510799339273f6fccb0484c4f4b6f';
const AUTHORIZED_INVESTOR  = '125c1bb803a2dd9194dca40d77445cf75647cb12';
const AUTHORIZED_PROTOCOL  = 'c1764f10ad672558afe1a3b666185fd141ae1ea8';

// Encoding

// Auth guard helper

// Auto-fill address fields when reward page opens

// Generic pool stat loader (reads from chain via admin RPC)

// Resolver reward data

// Builder reward data

// ── Submit handlers ──











// ═══════════════════════════════════════════
// SEARCH PAGE
// ═══════════════════════════════════════════



// ═══════════════════════════════════════════
// NEW DETAIL PAGE FUNCTIONS
// ═══════════════════════════════════════════






// Live ticker
window.updateTicker = function() {
  const track = document.getElementById('tickerTrack');
  if (!track) return;
  try {
    const txs = JSON.parse(localStorage.getItem('praxis_tx_cache') || '[]');
    if (!txs.length) return;
    const recent = txs.slice(-20).reverse();
    const items = recent.map(tx => {
      const type = tx.messageType || '';
      const sender = (tx.sender||'?').slice(0,8)+'…';
      const typeMap = {submit_prediction:'predicted',create_market:'created',claim_winnings:'claimed',propose_outcome:'proposed'};
      const action = typeMap[type] || type;
      const msg = (tx.transaction&&tx.transaction.msg)||{};
      const outcome = msg.outcome===true||msg.outcome==='true'||msg.outcome===1;
      const detail = type==='submit_prediction' ? (outcome?'<span class="t-yes">YES</span>':'<span class="t-no">NO</span>') : '';
      return `<div class="ticker-item"><span class="ticker-dot"></span><span class="t-user">${sender}</span><span class="t-action">${action}</span>${detail}</div>`;
    }).join('');
    track.innerHTML = items + items; // duplicate for seamless loop
  } catch(e) {}
};
setTimeout(updateTicker, 2000);
setInterval(updateTicker, 30000);
window.updateMinBondHint = function() {
  const mid = document.getElementById('pr_mid').value.trim().toLowerCase();
  const hint = document.getElementById('prop_bond_hint');
  const bondEl = document.getElementById('pr_bond');
  if (!hint) return;
  if (!mid || mid.length !== 40) {
    hint.textContent = 'Enter Market ID to compute min bond';
    hint.style.color = '';
    return;
  }
  const m = window._allMarkets.find(x => x.marketId === mid || x.txHash === mid);
  if (!m) {
    hint.textContent = 'Market not found in cache — browse Markets first';
    hint.style.color = 'var(--red)';
    return;
  }
  // BEff = current pool size (qYes + qNo)
  const beff = Number(m.qYes + m.qNo) / 1_000_000;
  const onePct = beff * 0.01;
  const minBond = Math.max(onePct, 60);
  hint.textContent = 'Min bond: ' + minBond.toFixed(2) + ' PRX  (max(1% of pool, 60 PRX) — deducted from resolver stake)';
  hint.style.color = 'var(--amber)';
  const bondEl2 = document.getElementById('pr_bond'); if (bondEl2 && parseFloat(bondEl2.value) < minBond) bondEl2.value = Math.ceil(minBond);
};
window.updateProposeRisk = function() {
  const mid = (document.getElementById('pr_mid')?.value || '').trim().toLowerCase();
  const box = document.getElementById('pr_risk_box');
  if (!box) return;
  renderRiskBox(box, getRiskInfo(mid));
  // also update bond hint
  if (typeof updateMinBondHint === 'function') updateMinBondHint();
};
window.updateDisputeRisk = function() {
  const mid = (document.getElementById('di_mid')?.value || '').trim().toLowerCase();
  const box = document.getElementById('di_risk_box');
  if (!box) return;
  renderRiskBox(box, getRiskInfo(mid));
};
window.handleImageUpload = async function(input) {
  const file = input.files && input.files[0];
  const hint = document.getElementById('c_img_hint');
  const preview = document.getElementById('c_img_preview');
  const img = document.getElementById('c_img_preview_img');
  const hidden = document.getElementById('c_img');
  if (!file) return;

  const MAX_SIZE = 5 * 1024 * 1024;
  if (file.size > MAX_SIZE) {
    if (hint) { hint.textContent = '✗ Image exceeds 5MB limit'; hint.style.color = 'var(--red)'; }
    input.value = '';
    return;
  }
  const allowed = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'];
  if (!allowed.includes(file.type)) {
    if (hint) { hint.textContent = '✗ Unsupported file type — use PNG, JPEG, WEBP, or GIF'; hint.style.color = 'var(--red)'; }
    input.value = '';
    return;
  }

  // instant local preview while upload is in flight
  const localUrl = URL.createObjectURL(file);
  if (preview && img) {
    img.src = localUrl;
    preview.style.display = '';
  }
  if (hint) { hint.textContent = 'Uploading…'; hint.style.color = ''; }

  try {
    const res = await fetch('/api/upload-image', {
      method: 'POST',
      headers: { 'Content-Type': file.type },
      body: file,
    });
    const data = await res.json();
    if (!res.ok) throw new Error(data.error || 'Upload failed');

    if (hidden) hidden.value = data.url;
    if (hint) { hint.textContent = '✓ Image uploaded'; hint.style.color = 'var(--green)'; }
  } catch (e) {
    if (hint) { hint.textContent = '✗ ' + e.message; hint.style.color = 'var(--red)'; }
    if (preview) preview.style.display = 'none';
    if (hidden) hidden.value = '';
  } finally {
    URL.revokeObjectURL(localUrl);
  }
};
window.previewBanner = function() {
  const url = (document.getElementById('c_img')?.value || '').trim();
  const preview = document.getElementById('c_img_preview');
  const img = document.getElementById('c_img_preview_img');
  const hint = document.getElementById('c_img_hint');
  if (!preview || !img) return;
  if (url && (url.startsWith('http') || url.startsWith('ipfs'))) {
    img.src = url;
    preview.style.display = '';
    img.onload = () => { if (hint) { hint.textContent = '✓ Image loaded'; hint.style.color = 'var(--green)'; } };
    img.onerror = () => {
      preview.style.display = 'none';
      if (hint) { hint.textContent = '✗ Could not load image — check URL or CORS policy'; hint.style.color = 'var(--red)'; }
    };
  } else {
    preview.style.display = 'none';
    if (hint) { hint.textContent = 'Image will be stored on-chain via IPFS or direct URL. Recommended: 16:9, min 800x450px.'; hint.style.color = ''; }
  }
};
window.updateExpiryFromDate = function() {
  const dtEl   = document.getElementById('c_expiry_dt');
  const hidden = document.getElementById('c_expiry');
  const hint   = document.getElementById('c_expiry_hint');
  if (!dtEl || !hidden || !hint) return;

  const val = dtEl.value;
  if (!val) {
    hint.textContent = 'Select a date to compute block height';
    hint.style.color = '';
    hidden.value = '';
    return;
  }

  const targetMs = new Date(val).getTime();
  const nowMs    = Date.now();
  const diffMs   = targetMs - nowMs;

  if (diffMs <= 0) {
    hint.textContent = 'Date must be in the future';
    hint.style.color = 'var(--red)';
    hidden.value = '';
    return;
  }

  const blocksNeeded = blocksFromNow(diffMs);
  const blockHeight  = window.currentHeight + blocksNeeded;
  hidden.value = blockHeight;

  const dur = fmtDuration(diffMs);
  hint.textContent = 'Block #' + blockHeight + '  (~' + dur + ' from now, ' + blocksNeeded + ' blocks)';
  hint.style.color = blocksNeeded < 100 ? 'var(--red)' : 'var(--amber)';
};
window.initExpiryDate = function() {
  const dtEl = document.getElementById('c_expiry_dt');
  if (!dtEl || dtEl.value) return;
  const d = new Date(Date.now() + 7 * 24 * 3600 * 1000);
  // Format: YYYY-MM-DDTHH:MM
  const pad = n => String(n).padStart(2, '0');
  dtEl.value = d.getFullYear() + '-' + pad(d.getMonth()+1) + '-' + pad(d.getDate()) + 'T' + pad(d.getHours()) + ':' + pad(d.getMinutes());
  updateExpiryFromDate();
};
window.pickCat = function(el) {
  document.querySelectorAll('#c_cat_pick .cpick').forEach(e => e.classList.remove('active'));
  el.classList.add('active');
};
window.updateTicker = function() {
  const track = document.getElementById('tickerTrack');
  if (!track) return;
  try {
    const txs = JSON.parse(localStorage.getItem('praxis_tx_cache') || '[]');
    if (!txs.length) return;
    const recent = txs.slice(-20).reverse();
    const items = recent.map(tx => {
      const type = tx.messageType || '';
      const sender = (tx.sender||'?').slice(0,8)+'…';
      const typeMap = {submit_prediction:'predicted',create_market:'created',claim_winnings:'claimed',propose_outcome:'proposed'};
      const action = typeMap[type] || type;
      const msg = (tx.transaction&&tx.transaction.msg)||{};
      const outcome = msg.outcome===true||msg.outcome==='true'||msg.outcome===1;
      const detail = type==='submit_prediction' ? (outcome?'<span class="t-yes">YES</span>':'<span class="t-no">NO</span>') : '';
      return `<div class="ticker-item"><span class="ticker-dot"></span><span class="t-user">${sender}</span><span class="t-action">${action}</span>${detail}</div>`;
    }).join('');
    track.innerHTML = items + items; // duplicate for seamless loop
  } catch(e) {}
};
