
// ═══════════════════════════════════════════
// BLS
// ═══════════════════════════════════════════
let bls12_381 = null;
(async () => {
  for (const url of ['https://esm.sh/@noble/curves@1.4.2/bls12-381','https://cdn.skypack.dev/@noble/curves@1.4.2/bls12-381']) {
    try { const m = await import(url); bls12_381 = m.bls12_381; break; } catch {}
  }
  if (!bls12_381) toast('BLS library failed to load — check internet', true);
})();

// ═══════════════════════════════════════════
// CONFIG & STATE
// ═══════════════════════════════════════════
const getRPCHost = () => localStorage.getItem('praxis_rpc_host') || 'localhost';
const getRPC     = () => `http://${getRPCHost()}:50002`;

let currentHeight = 0;
let currentNetworkID = 1;
let currentChainID   = 1;
let selectedOut   = true;
let resolveOut    = true;
let propOut       = true;
let revOut        = true;
let signerPrivKey = null, signerPubKey = null, signerAddress = null;

// ═══════════════════════════════════════════
// PROTO ENCODER
// ═══════════════════════════════════════════
function encV(value) {
  const out = []; let v = typeof value==='bigint'?value:BigInt(value);
  while(v>127n){out.push(Number((v&0x7fn)|0x80n));v>>=7n;}out.push(Number(v));return new Uint8Array(out);
}
function cat(...a){const t=a.reduce((s,x)=>s+x.length,0);const o=new Uint8Array(t);let off=0;for(const x of a){o.set(x,off);off+=x.length;}return o;}
function tag(f,w){return encV((BigInt(f)<<3n)|BigInt(w));}
function vf(f,v){const x=typeof v==='bigint'?v:BigInt(v);if(x===0n)return new Uint8Array(0);return cat(tag(f,0),encV(x));}
function bf(f,b){if(!b||!b.length)return new Uint8Array(0);return cat(tag(f,2),encV(b.length),b);}
function sf(f,s){if(!s||!s.length)return new Uint8Array(0);const e=new TextEncoder().encode(s);return cat(tag(f,2),encV(e.length),e);}
function ef(f,m){if(!m||!m.length)return new Uint8Array(0);return cat(tag(f,2),encV(m.length),m);}
function boolF(f,v){return cat(tag(f,0),new Uint8Array([v?1:0]));}

// ═══════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════
function h2b(hex){hex=hex.trim().toLowerCase();if(hex.length%2)throw new Error('Odd hex');const o=new Uint8Array(hex.length/2);for(let i=0;i<o.length;i++)o[i]=parseInt(hex.slice(i*2,i*2+2),16);return o;}
function b2h(b){return Array.from(b).map(x=>x.toString(16).padStart(2,'0')).join('');}
function fmtA(n){if(!n&&n!==0)return'—';const x=Number(n);if(x>=1e9)return(x/1e9).toFixed(2)+'B';if(x>=1e6)return(x/1e6).toFixed(2)+'M';if(x>=1e3)return(x/1e3).toFixed(1)+'k';return String(x);}
function esc(s){return String(s).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');}
function addr40(s,label){if(!s||s.length!==40)throw new Error(`${label||'Address'} must be 40 hex chars`);}
function mid40(s){addr40(s,'Market ID');}

// ═══════════════════════════════════════════
// google.protobuf.Any
// ═══════════════════════════════════════════
function encAny(typeUrl,inner){return cat(sf(1,typeUrl),bf(2,inner));}

// ═══════════════════════════════════════════
// INNER MESSAGE ENCODERS — field numbers match tx.proto
// ═══════════════════════════════════════════
function encSend(from,to,amt){return cat(bf(1,h2b(from)),bf(2,h2b(to)),vf(3,amt));}
function encCreate(creator,b0,expiry,nonce,question){return cat(bf(1,h2b(creator)),vf(2,b0),vf(3,expiry),vf(4,nonce),sf(5,question));}
function encPredict(mid,bettor,outcome,shares,maxcost){return cat(bf(1,h2b(mid)),bf(2,h2b(bettor)),boolF(3,outcome),vf(4,shares),vf(5,maxcost));}
function encResolve(mid,resolver,outcome){return cat(bf(1,h2b(mid)),bf(2,h2b(resolver)),boolF(3,outcome));}
function encClaim(mid,claimant){return cat(bf(1,h2b(mid)),bf(2,h2b(claimant)));}
function encRegister(addr,stake){return cat(bf(1,h2b(addr)),vf(2,stake));}
function encPropose(mid,resolver,outcome,bond){return cat(bf(1,h2b(mid)),bf(2,h2b(resolver)),boolF(3,outcome),vf(4,bond));}
function encDispute(mid,addr,bond){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)),vf(3,bond));}
function encCommit(mid,voter,hash){return cat(bf(1,h2b(mid)),bf(2,h2b(voter)),bf(3,h2b(hash)));}
function encReveal(mid,voter,vote,nonce){return cat(bf(1,h2b(mid)),bf(2,h2b(voter)),boolF(3,vote),bf(4,h2b(nonce)));}
function encTally(mid,addr){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)));}
function encFinalize(mid,addr){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)));}
function encSlash(mid,addr){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)));}

// ═══════════════════════════════════════════
// TX SIGN BYTES ENCODER
// ═══════════════════════════════════════════
function encSignBytes(msgType,typeUrl,inner,{txTime,fee,height,memo,netId,chainId}){
  const any=encAny(typeUrl,inner);
  return cat(
    sf(1,msgType),ef(2,any),
    vf(4,height||currentHeight),vf(5,txTime),vf(6,fee||10000),
    memo?sf(7,memo):new Uint8Array(0),
    vf(8,netId||1),vf(9,chainId||1),
  );
}

// ═══════════════════════════════════════════
// BLS SIGN
// ═══════════════════════════════════════════
async function blsSign(msg){
  if(!signerPrivKey)throw new Error('No key loaded — go to Signer');
  if(!bls12_381)throw new Error('BLS library not loaded');
  return await bls12_381.sign(msg,signerPrivKey);
}

// ═══════════════════════════════════════════
// BUILD SIGNED TX
// ═══════════════════════════════════════════
async function buildSigned(msgType,typeUrl,inner,meta){
  const txTime=BigInt(Date.now())*1000n;
  const p={txTime,fee:meta.fee||10000,height:meta.height||currentHeight,memo:'',netId:currentNetworkID,chainId:currentChainID};
  const sb=encSignBytes(msgType,typeUrl,inner,p);
  const sig=await blsSign(sb);
  const sigObj={publicKey:b2h(signerPubKey),signature:b2h(sig)};
  const base={signature:sigObj,createdHeight:p.height,time:Number(txTime),fee:p.fee,memo:'',networkID:currentNetworkID,chainID:currentChainID};
  return{type:msgType,...base,msgTypeUrl:typeUrl,msgBytes:b2h(inner)};
}

function buildUnsigned(msgType,typeUrl,inner,meta){
  const txTime=BigInt(Date.now())*1000n;
  const base={signature:null,createdHeight:meta.height||currentHeight,time:Number(txTime),fee:meta.fee||10000,memo:'',networkID:1,chainID:1};
  return{type:msgType,...base,msgTypeUrl:typeUrl,msgBytes:b2h(inner)};
}

// ═══════════════════════════════════════════
// RPC
// ═══════════════════════════════════════════
async function rpc(path,body={}){
  const r=await fetch(getRPC()+path,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
  const t=await r.text();if(!r.ok)throw new Error(`HTTP ${r.status}: ${t}`);
  try{return JSON.parse(t);}catch{return t;}
}
async function submitTxRPC(obj){const d=await rpc('/v1/tx',obj);return typeof d==='string'?d.replace(/^"|"$/g,''):JSON.stringify(d);}

// ═══════════════════════════════════════════
// TOAST
// ═══════════════════════════════════════════
let _tt;
window.toast=function(msg,isErr=false){
  const el=document.getElementById('toast');
  el.textContent=msg;el.className=isErr?'err':'ok';el.style.display='block';
  clearTimeout(_tt);_tt=setTimeout(()=>el.style.display='none',5000);
};

// ═══════════════════════════════════════════
// NAVIGATION
// ═══════════════════════════════════════════
window.showPage=function(id,btn){
  document.querySelectorAll('.page').forEach(p=>p.classList.remove('active'));
  document.getElementById('page-'+id).classList.add('active');
  document.querySelectorAll('#deskNav .ni').forEach(b=>b.classList.remove('active'));
  const dm=document.querySelector(`#deskNav [data-p="${id}"]`);if(dm)dm.classList.add('active');
  document.querySelectorAll('#bnav .bni').forEach(b=>b.classList.remove('active'));
  const bm=document.querySelector(`#bnav [data-p="${id}"]`);if(bm)bm.classList.add('active');
  if(id==='markets')loadMarkets();
  if(id==='wallet'){refreshBalance();loadMyPredictions();}
  closeNav();
};

// ═══════════════════════════════════════════
// MOBILE NAV
// ═══════════════════════════════════════════
window.openNav=function(){document.getElementById('mobNav').classList.add('open');};
window.closeNav=function(e){if(e&&e.target!==document.getElementById('mobNav'))return;document.getElementById('mobNav').classList.remove('open');};

function buildMobNav(){
  const body=document.getElementById('mobNavBody');
  if(!body)return;
  body.innerHTML=document.getElementById('deskNav').innerHTML;
  body.querySelectorAll('.ni').forEach(item=>{
    const p=item.getAttribute('data-p');
    if(p)item.setAttribute('onclick',`showPage('${p}',this)`);
  });
}

// ═══════════════════════════════════════════
// THEME
// ═══════════════════════════════════════════
window.toggleTheme=function(){
  const html=document.documentElement;
  const d=html.getAttribute('data-theme')==='dark';
  html.setAttribute('data-theme',d?'light':'dark');
  localStorage.setItem('praxis_theme',d?'light':'dark');
  updateTL();
};
function updateTL(){
  const d=document.documentElement.getAttribute('data-theme')==='dark';
  const lbl=d?'Light mode':'Dark mode';
  ['tlD','tlM'].forEach(id=>{const e=document.getElementById(id);if(e)e.textContent=lbl;});
}
const st=localStorage.getItem('praxis_theme');
if(st)document.documentElement.setAttribute('data-theme',st);
updateTL();

// ═══════════════════════════════════════════
// RPC STATUS
// ═══════════════════════════════════════════
window.checkRPC=async function(){
  try{
    const d=await rpc('/v1/query/height',{});currentHeight=d.height||0;if(d.networkID)currentNetworkID=d.networkID;if(d.chainID)currentChainID=d.chainID;
    ['rpcDot','rpcDotM'].forEach(id=>{const e=document.getElementById(id);if(e)e.className='dot live';});
    const el=document.getElementById('rpcStatus');if(el)el.textContent='live';
    const hb=document.getElementById('hBadge');if(hb)hb.textContent=`block ${currentHeight}`;
    const hm=document.getElementById('hbM');if(hm)hm.textContent=`#${currentHeight}`;
    ['ni_height'].forEach(id=>{const e=document.getElementById(id);if(e)e.textContent=currentHeight;});
    const ns=document.getElementById('ni_status');if(ns)ns.textContent='connected';
    const nr=document.getElementById('ni_rpc');if(nr)nr.textContent=getRPC();
    const sh=document.getElementById('sb_h');if(sh)sh.textContent=currentHeight;
    const ce=document.getElementById('c_expiry');if(ce&&!ce.value)ce.value=currentHeight+1000;
    const nonceEl=document.getElementById('c_nonce');if(nonceEl&&!nonceEl.value)nonceEl.value=BigInt(Date.now())*1000n;
  }catch{
    ['rpcDot','rpcDotM'].forEach(id=>{const e=document.getElementById(id);if(e)e.className='dot';});
    const el=document.getElementById('rpcStatus');if(el)el.textContent='offline';
    const ns=document.getElementById('ni_status');if(ns)ns.textContent='offline';
  }
};
window.applyHost=function(){const h=document.getElementById('ni_host').value.trim();if(h)localStorage.setItem('praxis_rpc_host',h);checkRPC();toast('Connecting to '+h+'…');};

// ═══════════════════════════════════════════
// OUTCOME TOGGLES
// ═══════════════════════════════════════════
window.setOut=function(v){selectedOut=v;document.getElementById('btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('btn_no').className='obtn no'+(!v?' active':'');};
window.setResOut=function(v){resolveOut=v;document.getElementById('rbtn_yes').className='obtn yes'+(v?' active':'');document.getElementById('rbtn_no').className='obtn no'+(!v?' active':'');};
window.setPropOut=function(v){propOut=v;document.getElementById('pbtn_yes').className='obtn yes'+(v?' active':'');document.getElementById('pbtn_no').className='obtn no'+(!v?' active':'');};
window.setRevOut=function(v){revOut=v;document.getElementById('rvbtn_yes').className='obtn yes'+(v?' active':'');document.getElementById('rvbtn_no').className='obtn no'+(!v?' active':'');};

// ═══════════════════════════════════════════
// SIGNER
// ═══════════════════════════════════════════
window.loadKey=async function(){
  const hex=document.getElementById('sk_input').value.trim().toLowerCase();
  if(hex.length!==64)return toast('Private key must be exactly 64 hex chars',true);
  try{
    if(!bls12_381)throw new Error('BLS library not loaded');
    signerPrivKey=h2b(hex);
    signerPubKey=bls12_381.getPublicKey(signerPrivKey);
    const hb=await crypto.subtle.digest('SHA-256',signerPubKey);
    signerAddress=b2h(new Uint8Array(hb).slice(0,20));
    document.getElementById('keyStatus').className='kstat loaded';
    document.getElementById('keyStatus').textContent='✓ loaded — '+signerAddress.slice(0,16)+'…';
    document.getElementById('sk_derived').style.display='block';
    document.getElementById('sk_pub').textContent=b2h(signerPubKey);
    document.getElementById('sk_addr').textContent=signerAddress;
    ['c_creator','p_bettor','r_resolver','cl_addr','s_from','w_addr','ft_addr',
     'reg_addr','prop_resolver','dis_addr','cv_voter','rv_voter','tal_addr','fin_addr','sl_addr'].forEach(id=>{
      const el=document.getElementById(id);if(el&&!el.value)el.value=signerAddress;
    });
    document.getElementById('sk_input').value='';
    refreshBalance();
    loadMyPredictions();
    toast('Key loaded — '+signerAddress);
  }catch(e){signerPrivKey=signerPubKey=signerAddress=null;toast('Key load failed: '+e.message,true);}
};
window.clearKey=function(){
  signerPrivKey=signerPubKey=signerAddress=null;
  document.getElementById('keyStatus').className='kstat';
  document.getElementById('keyStatus').textContent='○ No key loaded';
  document.getElementById('sk_derived').style.display='none';
  document.getElementById('sk_input').value='';
  toast('Key cleared');
};

// ═══════════════════════════════════════════
// ACCOUNT QUERY
// ═══════════════════════════════════════════
window.queryAccount=async function(){
  const addr=document.getElementById('w_addr').value.trim().toLowerCase();
  addr40(addr,'Address');
  try{
    const d=await rpc('/v1/query/account',{address:addr});
    document.getElementById('w_result').style.display='block';
    document.getElementById('w_balance').textContent=Number(d.amount||0).toLocaleString();
    document.getElementById('w_addrD').textContent=addr;
  }catch(e){toast('Query failed: '+e.message,true);}
};

// ═══════════════════════════════════════════
// FAILED TX
// ═══════════════════════════════════════════
window.checkFailedTxs=async function(){
  const addr=document.getElementById('ft_addr').value.trim().toLowerCase();
  addr40(addr,'Address');
  try{
    const d=await rpc('/v1/query/failed-txs',{address:addr,perPage:20});
    const c=d.totalCount||0;const el=document.getElementById('ft_result');el.style.display='block';
    if(c===0){el.innerHTML=`<div class="alert ag">✓ No failed transactions for ${addr.slice(0,12)}…</div>`;return;}
    const rows=(d.results||[]).map(r=>`<div style="margin-bottom:8px;padding:8px;background:var(--bg);border:1px solid var(--border);font-family:'JetBrains Mono',monospace;font-size:10px"><span style="color:var(--red)">${esc(r.error?.msg||'?')} (${r.error?.code})</span><br><span style="color:var(--text3)">${r.txHash?.slice(0,24)}…</span></div>`).join('');
    el.innerHTML=`<div class="alert ar">⚠ ${c} failed tx(s)</div>${rows}`;
  }catch(e){toast('Query failed: '+e.message,true);}
};

// ═══════════════════════════════════════════
// PENDING HELPER
// ═══════════════════════════════════════════
function setPend(btnId,pendId,on){
  const b=document.getElementById(btnId);const p=document.getElementById(pendId);
  if(b)b.disabled=on;if(p)p.style.display=on?'flex':'none';
}

async function doSubmit(msgType,typeUrl,inner,meta,btnId,pendId){
  if(!signerPrivKey)return toast('Load a private key in Signer first',true);
  if(!currentHeight)return toast('Node not connected',true);
  setPend(btnId,pendId,true);
  try{
    const tx=await buildSigned(msgType,typeUrl,inner,meta);
    const hash=await submitTxRPC(tx);
    toast('✓ Submitted — '+(hash.length>20?hash.slice(0,20)+'…':hash));
    checkRPC();
    if(msgType==='create_market')setTimeout(loadMarkets,3000);
  }catch(e){toast('Error: '+e.message,true);}
  finally{setPend(btnId,pendId,false);}
}

function showPL(outId,payId,tx){
  document.getElementById(outId).style.display='block';
  document.getElementById(payId).value=JSON.stringify(tx,null,2);
}

// ═══════════════════════════════════════════
// MY PREDICTIONS
// ═══════════════════════════════════════════
async function refreshBalance(){
  if(!signerAddress)return;
  try{
    const d=await rpc('/v1/query/account',{address:signerAddress});
    const bal=Number(d.amount||0);
    const wbal=document.getElementById('w_balance');if(wbal)wbal.textContent=bal.toLocaleString();
    const wres=document.getElementById('w_result');if(wres)wres.style.display='block';
    const wadr=document.getElementById('w_addrD');if(wadr)wadr.textContent=signerAddress;
    const waddr=document.getElementById('w_addr');if(waddr&&!waddr.value)waddr.value=signerAddress;
  }catch{}
}

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

    el.innerHTML = predictions.map(p =>
      '<div style="background:var(--bg);border:1px solid var(--border);padding:12px;margin-bottom:8px;display:flex;justify-content:space-between;align-items:center">' +
        '<div>' +
          '<div style="font-family:JetBrains Mono,monospace;font-size:10px;color:var(--text3);margin-bottom:4px">MKT ' + p.marketId.slice(0,12) + '…</div>' +
          '<div style="display:flex;gap:12px">' +
            '<span style="font-family:JetBrains Mono,monospace;font-size:11px;color:' + (p.outcome ? 'var(--green)' : 'var(--red)') + '">' + (p.outcome ? 'YES' : 'NO') + '</span>' +
            '<span style="font-family:JetBrains Mono,monospace;font-size:11px;color:var(--text2)">Shares: ' + fmtA(p.shares) + '</span>' +
            '<span style="font-family:JetBrains Mono,monospace;font-size:11px;color:var(--text2)">Max: ' + fmtA(p.maxCost) + ' PRX</span>' +
          '</div>' +
        '</div>' +
        '<span style="font-family:JetBrains Mono,monospace;font-size:9px;color:var(--text3)">#' + p.height + '</span>' +
      '</div>').join('');
  } catch (e) {
    el.innerHTML = '<div style="padding:12px;color:var(--red);font-family:JetBrains Mono,monospace;font-size:10px">Error: ' + esc(e.message) + '</div>';
  }
};

// ═══════════════════════════════════════════
// RENDER MARKET CARDS — Premium Design
// ═══════════════════════════════════════════
function renderMarketCards(markets) {
  var parts = [];
  parts.push('<div class="mgrid">');
  for (var i = 0; i < markets.length; i++) {
    var m = markets[i];
    var open = m.status === 0;
    var expired = m.status === 1;
    var proposed = m.status === 2;
    var disputed = m.status === 5;
    var finalized = m.status === 6;
    var statusClass = open ? 'sp-o' : (expired || proposed) ? 'sp-e' : disputed ? 'sp-d' : 'sp-f';
    var statusLabel = open ? 'Open' : expired ? 'Expired' : proposed ? 'Proposed' : disputed ? 'Disputed' : finalized ? 'Finalized' : 'Closed';
    var mid = m.marketId || m.txHash;
    var total = m.qYes + m.qNo;
    var yesPct = total > 0n ? Number(m.qYes * 100n / total) : 50;
    var noPct = 100 - yesPct;
    var cardClass = 'mcard' + (expired ? ' mexp' : '') + (finalized ? ' mfin' : '');
    var banner = '';
    if (expired) banner = '<div class="mc-banner bnr"><span>⏳</span> Awaiting resolver proposal</div>';
    if (proposed) banner = '<div class="mc-banner bnr"><span>🔎</span> Resolver: ' + (m.resolver ? m.resolver.slice(0,8) + '\u2026' : '?') + ' \u2014 proposed ' + (m.proposedOutcome ? '<span style="color:var(--green)">YES</span>' : '<span style="color:var(--red)">NO</span>') + '</div>';
    if (disputed) banner = '<div class="mc-banner bnd"><span>⚡</span> Dispute active — panel vote in progress</div>';
    if (finalized) banner = '<div class="mc-banner bnf"><span>✓</span> Market finalized</div>';
    var actYes = open ? 'onclick="fillP(\'' + mid + '\', true)"' : 'disabled';
    var actNo  = open ? 'onclick="fillP(\'' + mid + '\', false)"' : 'disabled';
    parts.push('<div class="' + cardClass + '">');
    parts.push('<div class="mc-head"><div class="mc-q" style="cursor:pointer" onclick="showDetail(\'' + mid + '\')">' + esc(m.question) + '</div><div class="spill ' + statusClass + '"><span class="dot"></span>' + statusLabel + '</div></div>');
    if (banner) parts.push(banner);
    parts.push('<div class="mc-prob"><div class="prob-row"><span class="prob-lbl">Implied probability</span><div class="prob-vals"><div style="text-align:center"><span class="pvy">' + yesPct + '%</span><span class="pvl">YES</span></div><div style="text-align:center"><span class="pvn">' + noPct + '%</span><span class="pvl">NO</span></div></div></div><div class="btrack"><div class="byes" style="width:' + yesPct + '%' + (finalized ? ';box-shadow:none;background:#4a4a4a' : '') + '"></div><div class="bno"' + (finalized ? ' style="background:#2a2a2a"' : '') + '></div></div></div>');
    parts.push('<div class="mc-pools"><div class="pc pcy"><div class="pc-lbl">YES Pool</div><div class="pc-val">' + fmtA(m.qYes) + ' PRX</div></div><div class="pc pcn"><div class="pc-lbl">NO Pool</div><div class="pc-val">' + fmtA(m.qNo) + ' PRX</div></div></div>');

    var actYes = open ? 'onclick="fillP(\'' + mid + '\', true)"' : 'disabled';
    var actNo  = open ? 'onclick="fillP(\'' + mid + '\', false)"' : 'disabled';
    parts.push('<div class="mc-acts"><button class="byes2" ' + actYes + '>BET YES</button><button class="bno2" ' + actNo + '>BET NO</button></div>');
    parts.push('<div class="card-foot"><div class="meta">');
    parts.push('<div class="mitem"><span class="mlbl">Total pool</span><span class="mval g">' + fmtA(m.qYes + m.qNo) + ' PRX</span></div>');
    parts.push('<div class="mitem"><span class="mlbl">' + (open ? 'Expires' : expired ? 'Expired' : proposed ? 'Proposed' : finalized ? 'Finalized' : 'Closed') + '</span><span class="mval">blk #' + (m.expiry ? Number(m.expiry) : '?') + '</span></div>');
    parts.push('<div class="mitem"><span class="mlbl">Creator</span><span class="mval">' + (m.creator ? m.creator.slice(0,8) + '…' : '???') + '</span></div>');
    parts.push('</div><span class="market-id">' + mid.slice(0,8) + '…</span></div>');
    parts.push('</div>');
  }
  parts.push('</div>');
  return parts.join('');
}

// store markets globally for detail view
let _allMarkets = [];

window.showDetail = function(marketId) {
  const m = _allMarkets.find(x => x.marketId === marketId || x.txHash === marketId);
  if (!m) return;
  const open = m.status === 0;
  const proposed = m.status === 2;
  const finalized = m.status === 6;
  const total = m.qYes + m.qNo;
  const yesPct = total > 0n ? Number(m.qYes * 100n / total) : 50;
  const noPct = 100 - yesPct;
  const mid = m.marketId || m.txHash;

  document.getElementById('det-question').textContent = m.question;
  document.getElementById('det-qyes').textContent = fmtA(m.qYes) + ' PRX';
  document.getElementById('det-qno').textContent = fmtA(m.qNo) + ' PRX';
  document.getElementById('det-yes-pct').textContent = yesPct + '%';
  document.getElementById('det-no-pct').textContent = noPct + '%';
  document.getElementById('det-bar').style.width = yesPct + '%';
  document.getElementById('det-mid').textContent = mid;
  document.getElementById('det-creator').textContent = m.creator || '—';
  document.getElementById('det-total').textContent = fmtA(m.qYes + m.qNo) + ' PRX';
  document.getElementById('det-expiry').textContent = m.expiry ? 'blk #' + Number(m.expiry) : '—';

  const resolverRow = document.getElementById('det-resolver-row');
  if (m.resolver) {
    resolverRow.style.display = '';
    document.getElementById('det-resolver').textContent = m.resolver + (m.proposedOutcome !== undefined ? ' → proposed ' + (m.proposedOutcome ? 'YES' : 'NO') : '');
  } else {
    resolverRow.style.display = 'none';
  }

  const statusLabels = {0:'Open',1:'Expired',2:'Proposed',5:'Disputed',6:'Finalized'};
  const statusClasses = {0:'sp-o',1:'sp-e',2:'sp-e',5:'sp-d',6:'sp-f'};
  document.getElementById('det-status-pill').innerHTML = '<div class="spill ' + (statusClasses[m.status]||'sp-f') + '"><span class="dot"></span>' + (statusLabels[m.status]||'Closed') + '</div>';

  const yesBtn = document.getElementById('det-bet-yes');
  const noBtn = document.getElementById('det-bet-no');
  if (open) {
    yesBtn.removeAttribute('disabled'); yesBtn.setAttribute('onclick', 'fillP(' + JSON.stringify(mid) + ', true)');
    noBtn.removeAttribute('disabled');  noBtn.setAttribute('onclick', 'fillP(' + JSON.stringify(mid) + ', false)');
  } else {
    yesBtn.setAttribute('disabled',''); noBtn.setAttribute('disabled','');
  }

  const bannerCard = document.getElementById('det-banner-card');
  if (m.status === 1) {
    bannerCard.style.display = '';
    bannerCard.innerHTML = '<div class="mc-banner bnr"><span>⏳</span> Awaiting resolver proposal</div>';
  } else if (m.status === 2) {
    bannerCard.style.display = '';
    bannerCard.innerHTML = '<div class="mc-banner bnr"><span>🔎</span> Resolver: ' + (m.resolver ? m.resolver.slice(0,8) + '…' : '?') + ' — proposed ' + (m.proposedOutcome ? '<span style="color:var(--green)">YES</span>' : '<span style="color:var(--red)">NO</span>') + '</div>';
  } else {
    bannerCard.style.display = 'none';
  }

  showPage('detail', null);
};

window.fillP = (id, outcome) => {
  document.getElementById('p_mid').value = id;
  if (outcome !== undefined) { setOut(outcome); }
  showPage('predict', null);
};
window.fillR = (id, outcome) => {
  document.getElementById('r_mid').value = id;
  if (outcome !== undefined) { setResOut(outcome); }
  showPage('resolve', null);
};
window.fillC = id => { document.getElementById('cl_mid').value = id; showPage('claim', null); };

// ═══════════════════════════════════════════
// MARKETS PAGE
// ═══════════════════════════════════════════
function decVarint(buf,pos){let r=0n,s=0n;while(pos<buf.length){const b=BigInt(buf[pos++]);r|=(b&0x7fn)<<s;s+=7n;if(!(b&0x80n))break;}return{v:r,p:pos};}

window.loadMarkets = async function () {
  const el = document.getElementById('marketsList');
  const countEl = document.getElementById('sb_c');
  el.innerHTML = '<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;loading markets from chain</div>';
  try {
    await checkRPC();

    const heightResp = await rpc('/v1/query/height', {});
    const tipHeight = Number(heightResp.height || currentHeight || 1);
    const BATCH = 100;
    const allTxs = [];

    for (let h = 1; h <= tipHeight; h += BATCH) {
      const batchPromises = [];
      for (let bh = h; bh < h + BATCH && bh <= tipHeight; bh++) {
        batchPromises.push(
          rpc('/v1/query/txs-by-height', { height: bh, perPage: 50 })
            .then(d => allTxs.push(...(d.results || [])))
            .catch(() => {})
        );
      }
      await Promise.all(batchPromises);
    }

    const marketsMap = new Map();

    for (const tx of allTxs) {
      if (tx.messageType !== 'create_market') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const question = msg.question || '';
      const creator  = tx.sender || '';
      const b0       = BigInt(msg.b0 || 0);
      const expiry   = BigInt(msg.expiryTime || msg.expiry_time || 0);
      const nonce    = BigInt(msg.nonce || 0);
      const lmsrSeed = b0 > 50000000n ? b0 - 50000000n : b0;

      let marketId = tx.txHash || (creator + String(nonce));
      try {
        const creatorBytes = /^[0-9a-fA-F]{40}$/.test(creator)
          ? h2b(creator)
          : (() => { const bin = atob(creator); return new Uint8Array([...bin].map(c => c.charCodeAt(0))); })();
        const nonceBytes = new Uint8Array(8);
        let n = nonce;
        for (let i = 7; i >= 0; i--) { nonceBytes[i] = Number(n & 0xffn); n >>= 8n; }
        const input = new Uint8Array(creatorBytes.length + 8);
        input.set(creatorBytes); input.set(nonceBytes, 20);
        const hash = await crypto.subtle.digest('SHA-256', input);
        marketId = b2h(new Uint8Array(hash).slice(0, 20));
      } catch (e) {}

      if (!marketsMap.has(marketId)) {
        marketsMap.set(marketId, {
          txHash: tx.txHash || '',
          marketId,
          question: question || '(no question)',
          creator,
          b0,
          lmsrSeed,
          expiry,
          nonce,
          status: 0,
          qYes: lmsrSeed / 2n,
          qNo:  lmsrSeed / 2n,
        });
      }
    }

    for (const tx of allTxs) {
      if (tx.messageType !== 'submit_prediction') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || msg.market_id || '';
      let marketId = rawMid;
      try { const b = Uint8Array.from(atob(rawMid), c => c.charCodeAt(0)); marketId = b2h(b); } catch(e) {}
      const outcome  = msg.outcome === true || msg.outcome === 'true' || msg.outcome === 1;
      const amount   = BigInt(msg.shares || msg.amount || 0);
      if (!marketId || !marketsMap.has(marketId)) continue;
      const m = marketsMap.get(marketId);
      if (outcome) { m.qYes += amount; } else { m.qNo += amount; }
    }

    for (const tx of allTxs) {
      if (tx.messageType !== 'propose_outcome') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || '';
      let marketId = rawMid;
      try { const b = Uint8Array.from(atob(rawMid), c => c.charCodeAt(0)); marketId = b2h(b); } catch(e) {}
      if (!marketId || !marketsMap.has(marketId)) continue;
      const m = marketsMap.get(marketId);
      let resolver = tx.sender || '';
      try { const rb = Uint8Array.from(atob(msg.resolverAddress || ''), c => c.charCodeAt(0)); resolver = b2h(rb); } catch(e) {}
      m.status = 2;
      m.resolver = resolver;
      m.proposedOutcome = msg.proposedOutcome;
    }

    const markets = [...marketsMap.values()];
    for (const m of markets) {
      if (m.expiry && currentHeight > Number(m.expiry) && m.status === 0) m.status = 1;
    }

    _allMarkets = markets;
    if (countEl) countEl.textContent = markets.length;
    if (markets.length === 0) {
      el.innerHTML = '<div class="alert ay">No markets on-chain yet.<br>Create the first one!</div>';
      return;
    }
    el.innerHTML = renderMarketCards(markets);

  } catch (e) {
    el.innerHTML = '<div class="alert ar">⚠ Cannot reach node at <code>' + getRPC() + '</code><br>' + esc(e.message) + '</div>';
  }
};

// ═══════════════════════════════════════════
// ── SEND
// ═══════════════════════════════════════════
window.build_send=function(){try{
  const from=document.getElementById('s_from').value.trim().toLowerCase();
  const to=document.getElementById('s_to').value.trim().toLowerCase();
  const amt=parseInt(document.getElementById('s_amount').value);
  const fee=parseInt(document.getElementById('s_fee').value)||10000;
  addr40(from,'From');addr40(to,'To');if(!amt||amt<=0)throw new Error('Amount > 0 required');
  showPL('so','sp',buildUnsigned('send','type.googleapis.com/types.MessageSend',encSend(from,to,amt),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_send=async function(){try{
  const from=document.getElementById('s_from').value.trim().toLowerCase();
  const to=document.getElementById('s_to').value.trim().toLowerCase();
  const amt=parseInt(document.getElementById('s_amount').value);
  const fee=parseInt(document.getElementById('s_fee').value)||10000;
  addr40(from,'From');addr40(to,'To');if(!amt||amt<=0)throw new Error('Amount > 0');
  await doSubmit('send','type.googleapis.com/types.MessageSend',encSend(from,to,amt),{fee},'btn_send','pend_send');
}catch(e){toast(e.message,true);}};

// ── CREATE MARKET
window.build_create=function(){try{
  const q=document.getElementById('c_question').value.trim();
  const cr=document.getElementById('c_creator').value.trim().toLowerCase();
  const b0=parseInt(document.getElementById('c_b0').value);
  const exp=parseInt(document.getElementById('c_expiry').value)||currentHeight+1000;
  const fee=parseInt(document.getElementById('c_fee').value)||10000;
  let nonce=document.getElementById('c_nonce').value;
  if(!nonce)nonce=BigInt(Date.now())*1000n;
  else nonce=parseInt(nonce);
  if(!q)throw new Error('Question required');addr40(cr,'Creator');
  showPL('co','cp',buildUnsigned('create_market','type.googleapis.com/types.MessageCreateMarket',encCreate(cr,b0,exp,nonce,q),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_create=async function(){try{
  const q=document.getElementById('c_question').value.trim();
  const cr=document.getElementById('c_creator').value.trim().toLowerCase();
  const b0=parseInt(document.getElementById('c_b0').value);
  const exp=parseInt(document.getElementById('c_expiry').value)||currentHeight+1000;
  const fee=parseInt(document.getElementById('c_fee').value)||10000;
  let nonce=document.getElementById('c_nonce').value;
  if(!nonce)nonce=BigInt(Date.now())*1000n;
  else nonce=parseInt(nonce);
  if(!q)throw new Error('Question required');addr40(cr,'Creator');
  await doSubmit('create_market','type.googleapis.com/types.MessageCreateMarket',encCreate(cr,b0,exp,nonce,q),{fee},'btn_create','pend_create');
}catch(e){toast(e.message,true);}};

// ── SUBMIT PREDICTION
window.build_predict=function(){try{
  const mid=document.getElementById('p_mid').value.trim().toLowerCase();mid40(mid);
  const bettor=document.getElementById('p_bettor').value.trim().toLowerCase();addr40(bettor,'Bettor');
  const shares=parseInt(document.getElementById('p_shares').value);
  const mc=parseInt(document.getElementById('p_maxcost').value);
  const fee=parseInt(document.getElementById('p_fee').value)||10000;
  if(shares<1)throw new Error('Shares min 1 PRX');
  showPL('po','pp',buildUnsigned('submit_prediction','type.googleapis.com/types.MessageSubmitPrediction',encPredict(mid,bettor,selectedOut,shares,mc),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_predict=async function(){try{
  const mid=document.getElementById('p_mid').value.trim().toLowerCase();mid40(mid);
  const bettor=document.getElementById('p_bettor').value.trim().toLowerCase();addr40(bettor,'Bettor');
  const shares=parseInt(document.getElementById('p_shares').value);
  const mc=parseInt(document.getElementById('p_maxcost').value);
  const fee=parseInt(document.getElementById('p_fee').value)||10000;
  if(shares<1)throw new Error('Shares min 1 PRX');
  await doSubmit('submit_prediction','type.googleapis.com/types.MessageSubmitPrediction',encPredict(mid,bettor,selectedOut,shares,mc),{fee},'btn_predict','pend_predict');
}catch(e){toast(e.message,true);}};

// ── RESOLVE MARKET
window.build_resolve=function(){try{
  const mid=document.getElementById('r_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('r_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const fee=parseInt(document.getElementById('r_fee').value)||10000;
  showPL('ro','rp',buildUnsigned('resolve_market','type.googleapis.com/types.MessageResolveMarket',encResolve(mid,res,resolveOut),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_resolve=async function(){try{
  const mid=document.getElementById('r_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('r_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const fee=parseInt(document.getElementById('r_fee').value)||10000;
  await doSubmit('resolve_market','type.googleapis.com/types.MessageResolveMarket',encResolve(mid,res,resolveOut),{fee},'btn_resolve','pend_resolve');
}catch(e){toast(e.message,true);}};

// ── CLAIM WINNINGS
window.build_claim=function(){try{
  const mid=document.getElementById('cl_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('cl_addr').value.trim().toLowerCase();addr40(addr,'Claimant');
  const fee=parseInt(document.getElementById('cl_fee').value)||10000;
  showPL('clo','clp',buildUnsigned('claim_winnings','type.googleapis.com/types.MessageClaimWinnings',encClaim(mid,addr),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_claim=async function(){try{
  const mid=document.getElementById('cl_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('cl_addr').value.trim().toLowerCase();addr40(addr,'Claimant');
  const fee=parseInt(document.getElementById('cl_fee').value)||10000;
  await doSubmit('claim_winnings','type.googleapis.com/types.MessageClaimWinnings',encClaim(mid,addr),{fee},'btn_claim','pend_claim');
}catch(e){toast(e.message,true);}};

// ── REGISTER RESOLVER
window.build_register=function(){try{
  const addr=document.getElementById('reg_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
  const stake=parseInt(document.getElementById('reg_stake').value);
  const fee=parseInt(document.getElementById('reg_fee').value)||10000;
  if(stake<100)throw new Error('Stake min 100 PRX');
  showPL('rego','regp',buildUnsigned('register_resolver','type.googleapis.com/types.MessageRegisterResolver',encRegister(addr,stake),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_register=async function(){try{
  const addr=document.getElementById('reg_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
  const stake=parseInt(document.getElementById('reg_stake').value);
  const fee=parseInt(document.getElementById('reg_fee').value)||10000;
  if(stake<100)throw new Error('Stake min 100 PRX');
  await doSubmit('register_resolver','type.googleapis.com/types.MessageRegisterResolver',encRegister(addr,stake),{fee},'btn_register','pend_register');
}catch(e){toast(e.message,true);}};

// ── PROPOSE OUTCOME
window.build_propose=function(){try{
  const mid=document.getElementById('prop_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('prop_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const bond=parseInt(document.getElementById('prop_bond').value);
  const fee=parseInt(document.getElementById('prop_fee').value)||10000;
  showPL('propo','propp',buildUnsigned('propose_outcome','type.googleapis.com/types.MessageProposeOutcome',encPropose(mid,res,propOut,bond),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_propose=async function(){try{
  const mid=document.getElementById('prop_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('prop_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const bond=parseInt(document.getElementById('prop_bond').value);
  const fee=parseInt(document.getElementById('prop_fee').value)||10000;
  await doSubmit('propose_outcome','type.googleapis.com/types.MessageProposeOutcome',encPropose(mid,res,propOut,bond),{fee},'btn_propose','pend_propose');
}catch(e){toast(e.message,true);}};

// ── FILE DISPUTE
window.build_dispute=function(){try{
  const mid=document.getElementById('dis_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('dis_addr').value.trim().toLowerCase();addr40(addr,'Disputer');
  const bond=parseInt(document.getElementById('dis_bond').value);
  const fee=parseInt(document.getElementById('dis_fee').value)||10000;
  showPL('diso','disp',buildUnsigned('file_dispute','type.googleapis.com/types.MessageFileDispute',encDispute(mid,addr,bond),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_dispute=async function(){try{
  const mid=document.getElementById('dis_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('dis_addr').value.trim().toLowerCase();addr40(addr,'Disputer');
  const bond=parseInt(document.getElementById('dis_bond').value);
  const fee=parseInt(document.getElementById('dis_fee').value)||10000;
  await doSubmit('file_dispute','type.googleapis.com/types.MessageFileDispute',encDispute(mid,addr,bond),{fee},'btn_dispute','pend_dispute');
}catch(e){toast(e.message,true);}};

// ── COMMIT VOTE
window.build_commit=function(){try{
  const mid=document.getElementById('cv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('cv_voter').value.trim().toLowerCase();addr40(voter,'Voter');
  const hash=document.getElementById('cv_hash').value.trim().toLowerCase();if(hash.length!==64)throw new Error('Commit hash must be 64 hex chars');
  const fee=parseInt(document.getElementById('cv_fee').value)||10000;
  showPL('cvo','cvp',buildUnsigned('commit_vote','type.googleapis.com/types.MessageCommitVote',encCommit(mid,voter,hash),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_commit=async function(){try{
  const mid=document.getElementById('cv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('cv_voter').value.trim().toLowerCase();addr40(voter,'Voter');
  const hash=document.getElementById('cv_hash').value.trim().toLowerCase();if(hash.length!==64)throw new Error('Commit hash must be 64 hex chars');
  const fee=parseInt(document.getElementById('cv_fee').value)||10000;
  await doSubmit('commit_vote','type.googleapis.com/types.MessageCommitVote',encCommit(mid,voter,hash),{fee},'btn_commit','pend_commit');
}catch(e){toast(e.message,true);}};

// ── REVEAL VOTE
window.build_reveal=function(){try{
  const mid=document.getElementById('rv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('rv_voter').value.trim().toLowerCase();addr40(voter,'Voter');
  const nonce=document.getElementById('rv_nonce').value.trim().toLowerCase();if(nonce.length!==64)throw new Error('Nonce must be 64 hex chars');
  const fee=parseInt(document.getElementById('rv_fee').value)||10000;
  showPL('rvo','rvp',buildUnsigned('reveal_vote','type.googleapis.com/types.MessageRevealVote',encReveal(mid,voter,revOut,nonce),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_reveal=async function(){try{
  const mid=document.getElementById('rv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('rv_voter').value.trim().toLowerCase();addr40(voter,'Voter');
  const nonce=document.getElementById('rv_nonce').value.trim().toLowerCase();if(nonce.length!==64)throw new Error('Nonce must be 64 hex chars');
  const fee=parseInt(document.getElementById('rv_fee').value)||10000;
  await doSubmit('reveal_vote','type.googleapis.com/types.MessageRevealVote',encReveal(mid,voter,revOut,nonce),{fee},'btn_reveal','pend_reveal');
}catch(e){toast(e.message,true);}};

// ── TALLY VOTES
window.build_tally=function(){try{
  const mid=document.getElementById('tal_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('tal_addr').value.trim().toLowerCase();addr40(addr,'Caller');
  const fee=parseInt(document.getElementById('tal_fee').value)||10000;
  showPL('talo','talp',buildUnsigned('tally_votes','type.googleapis.com/types.MessageTallyVotes',encTally(mid,addr),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_tally=async function(){try{
  const mid=document.getElementById('tal_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('tal_addr').value.trim().toLowerCase();addr40(addr,'Caller');
  const fee=parseInt(document.getElementById('tal_fee').value)||10000;
  await doSubmit('tally_votes','type.googleapis.com/types.MessageTallyVotes',encTally(mid,addr),{fee},'btn_tally','pend_tally');
}catch(e){toast(e.message,true);}};

// ── FINALIZE MARKET
window.build_finalize=function(){try{
  const mid=document.getElementById('fin_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('fin_addr').value.trim().toLowerCase();addr40(addr,'Caller');
  const fee=parseInt(document.getElementById('fin_fee').value)||10000;
  showPL('fino','finp',buildUnsigned('finalize_market','type.googleapis.com/types.MessageFinalizeMarket',encFinalize(mid,addr),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_finalize=async function(){try{
  const mid=document.getElementById('fin_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('fin_addr').value.trim().toLowerCase();addr40(addr,'Caller');
  const fee=parseInt(document.getElementById('fin_fee').value)||10000;
  await doSubmit('finalize_market','type.googleapis.com/types.MessageFinalizeMarket',encFinalize(mid,addr),{fee},'btn_finalize','pend_finalize');
}catch(e){toast(e.message,true);}};

// ── CLAIM SLASH
window.build_slash=function(){try{
  const mid=document.getElementById('sl_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('sl_addr').value.trim().toLowerCase();addr40(addr,'Claimant');
  const fee=parseInt(document.getElementById('sl_fee').value)||10000;
  showPL('slo','slp',buildUnsigned('claim_slash','type.googleapis.com/types.MessageClaimSlash',encSlash(mid,addr),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_slash=async function(){try{
  const mid=document.getElementById('sl_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('sl_addr').value.trim().toLowerCase();addr40(addr,'Claimant');
  const fee=parseInt(document.getElementById('sl_fee').value)||10000;
  await doSubmit('claim_slash','type.googleapis.com/types.MessageClaimSlash',encSlash(mid,addr),{fee},'btn_slash','pend_slash');
}catch(e){toast(e.message,true);}};

// ═══════════════════════════════════════════
// MAINNET POLISH — UI ONLY, NO CHAIN LOGIC
// ═══════════════════════════════════════════

// PRX denomination — 1 PRX = 1 PRX (no micro conversion)

// Copy to clipboard
window.copyText = async function(text, btn) {
  try {
    await navigator.clipboard.writeText(text);
    if (btn) { btn.textContent = '✓'; btn.classList.add('ok'); setTimeout(() => { btn.textContent = '⎘'; btn.classList.remove('ok'); }, 1800); }
    toast('Copied');
  } catch { toast('Copy failed', true); }
};

// Wire copy buttons to derived address and pubkey after key load
function wireCopyBtns() {
  const pairs = [
    ['sk_addr', 'copy_sk_addr'],
    ['sk_pub',  'copy_sk_pub'],
  ];
  pairs.forEach(([srcId, btnId]) => {
    const btn = document.getElementById(btnId);
    if (!btn) return;
    btn.onclick = function() {
      const el = document.getElementById(srcId);
      copyText(el ? el.textContent.trim() : '', this);
    };
  });
  // payload boxes
  document.querySelectorAll('.payload-box textarea').forEach(ta => {
    const box = ta.closest('.payload-box');
    if (!box || box.querySelector('.copy-payload-btn')) return;
    const b = document.createElement('button');
    b.className = 'btn bg bsm copy-payload-btn';
    b.style.cssText = 'margin-top:6px;font-size:10px';
    b.textContent = '⎘ Copy payload';
    b.onclick = function() { copyText(ta.value, this); };
    box.appendChild(b);
  });
}

// Inject copy buttons into derived key display
function injectKeyboardCopyBtns() {
  const addrEl = document.getElementById('sk_addr');
  const pubEl  = document.getElementById('sk_pub');
  if (addrEl && !document.getElementById('copy_sk_addr')) {
    const wrap = document.createElement('div');
    wrap.className = 'cwrap';
    addrEl.parentNode.insertBefore(wrap, addrEl);
    wrap.appendChild(addrEl);
    const btn = document.createElement('button');
    btn.id = 'copy_sk_addr'; btn.className = 'cbtn'; btn.textContent = '⎘';
    btn.title = 'Copy address';
    wrap.appendChild(btn);
  }
  if (pubEl && !document.getElementById('copy_sk_pub')) {
    const wrap = document.createElement('div');
    wrap.className = 'cwrap';
    pubEl.parentNode.insertBefore(wrap, pubEl);
    wrap.appendChild(pubEl);
    const btn = document.createElement('button');
    btn.id = 'copy_sk_pub'; btn.className = 'cbtn'; btn.textContent = '⎘';
    btn.title = 'Copy pubkey';
    wrap.appendChild(btn);
  }
  wireCopyBtns();
}

// Confirm modal
let _confirmResolve = null;
window.closeConfirm = function() {
  document.getElementById('confOverlay').classList.remove('open');
  if (_confirmResolve) { _confirmResolve(false); _confirmResolve = null; }
};
document.getElementById('confOk').onclick = function() {
  document.getElementById('confOverlay').classList.remove('open');
  if (_confirmResolve) { _confirmResolve(true); _confirmResolve = null; }
};
document.getElementById('confOverlay').addEventListener('click', function(e) {
  if (e.target === this) closeConfirm();
});

function showConfirm(title, rows) {
  return new Promise(resolve => {
    _confirmResolve = resolve;
    document.getElementById('confTitle').textContent = title;
    document.getElementById('confSub').textContent = 'review before signing · canopy network';
    const rowsEl = document.getElementById('confRows');
    rowsEl.innerHTML = rows.map(([l, v, cls]) =>
      `<div class="confirm-row"><span class="confirm-lbl">${l}</span><span class="confirm-val ${cls||''}">${v}</span></div>`
    ).join('');
    document.getElementById('confOverlay').classList.add('open');
  });
}

// Patch signAndSubmit_* functions with confirm gate
// We wrap — originals are preserved, just called after confirmation
(function() {
  const v = id => parseInt(document.getElementById(id)?.value)||0;
  const patches = {
    signAndSubmit_create:  () => [
      'Create Market', [
        ['Question',    document.getElementById('c_question')?.value || '—', ''],
        ['B0 Liquidity', v('c_b0').toLocaleString()+' PRX', 'green'],
        ['Fee',         v('c_fee')+' PRX', ''],
      ]
    ],
    signAndSubmit_predict: () => [
      'Submit Prediction', [
        ['Market ID',   (document.getElementById('p_mid')?.value||'').slice(0,16)+'…', ''],
        ['Outcome',     (window._selectedOut!==false?'YES':'NO'), window._selectedOut!==false?'green':'red'],
        ['Shares',      v('p_shares').toLocaleString()+' PRX', ''],
        ['Max Cost',    v('p_maxcost').toLocaleString()+' PRX', ''],
      ]
    ],
    signAndSubmit_resolve: () => [
      'Resolve Market', [
        ['Market ID',   (document.getElementById('r_mid')?.value||'').slice(0,16)+'…', ''],
        ['Outcome',     (window._resolveOut!==false?'YES':'NO'), window._resolveOut!==false?'green':'red'],
      ]
    ],
    signAndSubmit_claim: () => [
      'Claim Winnings', [
        ['Market ID',   (document.getElementById('cl_mid')?.value||'').slice(0,16)+'…', ''],
        ['Claimant',    (document.getElementById('cl_addr')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_register: () => [
      'Register Resolver', [
        ['Address',     (document.getElementById('reg_addr')?.value||'').slice(0,16)+'…', ''],
        ['Stake',       v('reg_stake').toLocaleString()+' PRX', 'green'],
      ]
    ],
    signAndSubmit_propose: () => [
      'Propose Outcome', [
        ['Market ID',   (document.getElementById('prop_mid')?.value||'').slice(0,16)+'…', ''],
        ['Outcome',     (window._propOut!==false?'YES':'NO'), window._propOut!==false?'green':'red'],
        ['Bond',        v('prop_bond').toLocaleString()+' PRX', ''],
      ]
    ],
    signAndSubmit_dispute: () => [
      'File Dispute', [
        ['Market ID',   (document.getElementById('dis_mid')?.value||'').slice(0,16)+'…', ''],
        ['Bond',        v('dis_bond').toLocaleString()+' PRX', ''],
      ]
    ],
    signAndSubmit_commit: () => [
      'Commit Vote', [
        ['Market ID',   (document.getElementById('cv_mid')?.value||'').slice(0,16)+'…', ''],
        ['Commit Hash', (document.getElementById('cv_hash')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_reveal: () => [
      'Reveal Vote', [
        ['Market ID',   (document.getElementById('rv_mid')?.value||'').slice(0,16)+'…', ''],
        ['Vote',        (window._revOut!==false?'YES':'NO'), window._revOut!==false?'green':'red'],
      ]
    ],
    signAndSubmit_tally: () => [
      'Tally Votes', [
        ['Market ID',   (document.getElementById('tal_mid')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_finalize: () => [
      'Finalize Market', [
        ['Market ID',   (document.getElementById('fin_mid')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_slash: () => [
      'Claim Slash', [
        ['Market ID',   (document.getElementById('sl_mid')?.value||'').slice(0,16)+'…', ''],
        ['Claimant',    (document.getElementById('sl_addr')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_send: () => [
      'Send $PRX', [
        ['To',    (document.getElementById('s_to')?.value||'').slice(0,16)+'…', ''],
        ['Amount', v('s_amount').toLocaleString()+' PRX', 'green'],
      ]
    ],
  };

  // Expose outcome vars so patches can read them
  // (they already exist as module-level vars; we shadow-expose via a getter trick)
  Object.defineProperty(window, '_selectedOut', { get: () => typeof selectedOut !== 'undefined' ? selectedOut : true });
  Object.defineProperty(window, '_resolveOut',  { get: () => typeof resolveOut !== 'undefined' ? resolveOut : true });
  Object.defineProperty(window, '_propOut',     { get: () => typeof propOut !== 'undefined' ? propOut : true });
  Object.defineProperty(window, '_revOut',      { get: () => typeof revOut !== 'undefined' ? revOut : true });

  Object.keys(patches).forEach(name => {
    const orig = window[name];
    if (!orig) return;
    window[name] = async function() {
      const [title, rows] = patches[name]();
      const ok = await showConfirm(title, rows);
      if (ok) await orig();
    };
  });
})();

// Offline banner wired to RPC status
const _origCheckRPC = window.checkRPC;
window.checkRPC = async function() {
  try {
    await _origCheckRPC();
    document.getElementById('offBanner').classList.remove('show');
  } catch {
    document.getElementById('offBanner').classList.add('show');
  }
};

// Session badge visibility
const _origLoadKey = window.loadKey;
window.loadKey = async function() {
  await _origLoadKey();
  const badge = document.getElementById('sessBadge');
  if (badge) badge.classList.remove('hidden');
  injectKeyboardCopyBtns();
  setTimeout(wireCopyBtns, 100);
};
const _origClearKey = window.clearKey;
window.clearKey = function() {
  _origClearKey();
  const badge = document.getElementById('sessBadge');
  if (badge) badge.classList.add('hidden');
};

// Wire payload copy buttons when pages are shown
const _origShowPage = window.showPage;
window.showPage = function(id, btn) {
  _origShowPage(id, btn);
  setTimeout(wireCopyBtns, 50);
};

// Init copy btn injection
injectKeyboardCopyBtns();

// ═══════════════════════════════════════════
// INIT
// ═══════════════════════════════════════════
document.getElementById('ni_host').value=getRPCHost();
buildMobNav();
checkRPC();
setInterval(checkRPC,12000);

// ═══════════════════════════════════════════
// KEYSTORE — AES-GCM + PBKDF2
// ═══════════════════════════════════════════
async function deriveKey(password, salt) {
  const enc = new TextEncoder();
  const keyMaterial = await crypto.subtle.importKey(
    'raw', enc.encode(password), 'PBKDF2', false, ['deriveKey']
  );
  return crypto.subtle.deriveKey(
    { name: 'PBKDF2', salt, iterations: 200000, hash: 'SHA-256' },
    keyMaterial,
    { name: 'AES-GCM', length: 256 },
    false,
    ['encrypt', 'decrypt']
  );
}

async function encryptKey(privKeyBytes, password) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv   = crypto.getRandomValues(new Uint8Array(12));
  const key  = await deriveKey(password, salt);
  const enc  = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, privKeyBytes);
  return {
    salt: b2h(salt),
    iv:   b2h(iv),
    encrypted: b2h(new Uint8Array(enc))
  };
}

async function decryptKey(encrypted, iv, salt, password) {
  const key = await deriveKey(password, h2b(salt));
  const dec = await crypto.subtle.decrypt(
    { name: 'AES-GCM', iv: h2b(iv) },
    key,
    h2b(encrypted)
  );
  return new Uint8Array(dec);
}

window.createKeystore = async function() {
  const pw  = document.getElementById('ks_new_pw').value;
  const pw2 = document.getElementById('ks_new_pw2').value;
  if (!pw) return toast('Enter a password', true);
  if (pw !== pw2) return toast('Passwords do not match', true);
  if (pw.length < 8) return toast('Password must be at least 8 characters', true);

  try {
    // generate new BLS private key
    const privBytes = crypto.getRandomValues(new Uint8Array(32));
    const pubKey    = bls12_381.getPublicKey(privBytes);
    const hash      = await crypto.subtle.digest('SHA-256', pubKey);
    const address   = b2h(new Uint8Array(hash).slice(0, 20));

    const { salt, iv, encrypted } = await encryptKey(privBytes, pw);

    const keystore = {
      version: 1,
      publicKey: b2h(pubKey),
      keyAddress: address,
      salt, iv, encrypted
    };

    // download
    const blob = new Blob([JSON.stringify(keystore, null, 2)], { type: 'application/json' });
    const url  = URL.createObjectURL(blob);
    const a    = document.createElement('a');
    a.href = url; a.download = 'praxis-keystore-' + address.slice(0,8) + '.json';
    a.click(); URL.revokeObjectURL(url);

    // auto-load into session
    signerPrivKey = privBytes;
    signerPubKey  = pubKey;
    signerAddress = address;
    updateSignerUI();
    toast('Keystore created and loaded');
    document.getElementById('ks_new_pw').value = '';
    document.getElementById('ks_new_pw2').value = '';
  } catch(e) { toast('Create failed: ' + e.message, true); }
};

window.importKeystore = async function() {
  const pw   = document.getElementById('ks_imp_pw').value;
  const file = document.getElementById('ks_imp_file').files[0];
  if (!file) return toast('Select a keystore file', true);
  if (!pw)   return toast('Enter password', true);

  try {
    const text = await file.text();
    const ks   = JSON.parse(text);
    if (!ks.encrypted || !ks.salt || !ks.iv || !ks.publicKey) throw new Error('Invalid keystore file');

    const privBytes = await decryptKey(ks.encrypted, ks.iv, ks.salt, pw);
    const pubKey    = bls12_381.getPublicKey(privBytes);

    // verify pubkey matches
    if (b2h(pubKey) !== ks.publicKey) throw new Error('Wrong password or corrupted keystore');

    const hash    = await crypto.subtle.digest('SHA-256', pubKey);
    const address = b2h(new Uint8Array(hash).slice(0, 20));

    signerPrivKey = privBytes;
    signerPubKey  = pubKey;
    signerAddress = address;
    updateSignerUI();
    toast('Keystore unlocked — ' + address.slice(0,8) + '…');
    document.getElementById('ks_imp_pw').value = '';
    document.getElementById('ks_imp_file').value = '';
  } catch(e) { toast('Import failed: ' + e.message, true); }
};

function updateSignerUI() {
  document.getElementById('keyStatus').className = 'kstat loaded';
  document.getElementById('keyStatus').textContent = '✓ loaded — ' + signerAddress.slice(0,16) + '…';
  document.getElementById('sk_derived').style.display = 'block';
  document.getElementById('sk_pub').textContent = b2h(signerPubKey);
  document.getElementById('sk_addr').textContent = signerAddress;
  ['c_creator','p_bettor','r_resolver','cl_addr','s_from','w_addr','ft_addr',
   'reg_addr','prop_resolver','dis_addr','cv_voter','rv_voter','tal_addr','fin_addr','sl_addr'].forEach(id => {
    const el = document.getElementById(id); if (el && !el.value) el.value = signerAddress;
  });
  const badge = document.getElementById('sessBadge');
  if (badge) badge.classList.remove('hidden');
  refreshBalance();
  loadMyPredictions();
  injectKeyboardCopyBtns();
  setTimeout(wireCopyBtns, 100);
}

// ═══════════════════════════════════════════
// METAMASK
// ═══════════════════════════════════════════
let mmAddress = null;

window.connectMetaMask = async function() {
  if (!window.ethereum) return toast('MetaMask not detected — install MetaMask first', true);
  try {
    const accounts = await window.ethereum.request({ method: 'eth_requestAccounts' });
    if (!accounts || accounts.length === 0) return toast('No accounts returned', true);
    mmAddress = accounts[0].toLowerCase();
    updateMMUI(true);
    toast('MetaMask connected — ' + mmAddress.slice(0,8) + '…');
  } catch(e) {
    toast('MetaMask connection failed: ' + e.message, true);
  }
};

window.disconnectMetaMask = function() {
  mmAddress = null;
  updateMMUI(false);
  toast('MetaMask disconnected');
};

function updateMMUI(connected) {
  const disc = document.getElementById('mm_disconnected');
  const conn = document.getElementById('mm_connected');
  const addr = document.getElementById('mm_addr');
  const status = document.getElementById('mm_status');
  if (!disc || !conn) return;
  if (connected) {
    disc.style.display = 'none';
    conn.style.display = 'block';
    if (addr) addr.textContent = mmAddress;
    if (status) status.textContent = '✓ connected — ' + mmAddress.slice(0,8) + '…';
  } else {
    disc.style.display = 'block';
    conn.style.display = 'none';
  }
}

// auto-reconnect if already authorized
(async () => {
  if (!window.ethereum) return;
  try {
    const accounts = await window.ethereum.request({ method: 'eth_accounts' });
    if (accounts && accounts.length > 0) {
      mmAddress = accounts[0].toLowerCase();
      updateMMUI(true);
    }
  } catch {}
})();

// listen for account changes
if (window.ethereum) {
  window.ethereum.on('accountsChanged', (accounts) => {
    if (accounts.length === 0) { mmAddress = null; updateMMUI(false); }
    else { mmAddress = accounts[0].toLowerCase(); updateMMUI(true); }
  });
}
