
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
function hexToBytes(hex){const b=new Uint8Array(hex.length/2);for(let i=0;i<hex.length;i+=2)b[i/2]=parseInt(hex.slice(i,i+2),16);return b;}
function bytesToHex(b){return Array.from(b).map(x=>x.toString(16).padStart(2,"0")).join("");}

// ═══════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════
function h2b(hex){hex=hex.trim().toLowerCase();if(hex.length%2)throw new Error('Odd hex');const o=new Uint8Array(hex.length/2);for(let i=0;i<o.length;i++)o[i]=parseInt(hex.slice(i*2,i*2+2),16);return o;}
function b2h(b){return Array.from(b).map(x=>x.toString(16).padStart(2,'0')).join('');}
function fmtA(n){if(!n&&n!==0)return'—';const x=Number(n);if(x>=1e9)return(x/1e9).toFixed(2)+'B';if(x>=1e6)return(x/1e6).toFixed(2)+'M';if(x>=1e3)return(x/1e3).toFixed(1)+'k';return String(x);}
function fmtPRX(n){if(!n&&n!==0)return'—';const x=Number(n)/1_000_000;if(x>=1e9)return(x/1e9).toFixed(2)+'B';if(x>=1e6)return(x/1e6).toFixed(2)+'M';if(x>=1000)return(x/1000).toFixed(2)+'k';if(x>=1)return x.toFixed(2);return x.toFixed(6);}
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
function encCreate(creator,b0,expiry,nonce,question,rules){return cat(bf(1,h2b(creator)),vf(2,b0),vf(3,expiry),vf(4,nonce),sf(5,question),sf(6,rules||''));}
function encPredict(mid,bettor,outcome,shares,maxcost){return cat(bf(1,h2b(mid)),bf(2,h2b(bettor)),boolF(3,outcome),vf(4,shares),vf(5,maxcost));}
function encResolve(mid,resolver,outcome){return cat(bf(1,h2b(mid)),bf(2,h2b(resolver)),boolF(3,outcome));}
function encClaim(mid,claimant){return cat(bf(1,h2b(mid)),bf(2,h2b(claimant)));}
function encReclaim(mid,claimant){return cat(bf(1,h2b(mid)),bf(2,h2b(claimant)));}
function encRegister(addr,stake){return cat(bf(1,h2b(addr)),vf(2,stake));}
function encPropose(mid,resolver,outcome,bond){return cat(bf(1,h2b(mid)),bf(2,h2b(resolver)),boolF(3,outcome),vf(4,bond));}
function encDispute(mid,addr,bond){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)),vf(3,bond));}
function encCommit(mid,voter,hash){return cat(bf(1,h2b(mid)),bf(2,h2b(voter)),bf(3,h2b(hash)));}
function encReveal(mid,voter,vote,nonce){return cat(bf(1,h2b(mid)),bf(2,h2b(voter)),boolF(3,vote),bf(4,h2b(nonce)));}
function encTally(mid,addr){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)));}
function encFinalize(mid,addr){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)));}
function encSlash(mid,addr){return cat(bf(1,h2b(mid)),bf(2,h2b(addr)));}
function encForfeit(mid,resolver){return cat(bf(1,h2b(mid)),bf(2,h2b(resolver)));}
function encUnstakeResolver(addr,amount){return cat(bf(1,h2b(addr)),vf(2,amount));}
function encClaimUnbonded(addr){return cat(bf(1,h2b(addr)));}

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
// BASE64 HELPER (for proto JSON encoding)
// ═══════════════════════════════════════════
function b2b64(bytes){
  let s='';for(let i=0;i<bytes.length;i++)s+=String.fromCharCode(bytes[i]);
  return btoa(s);
}

// ═══════════════════════════════════════════
// BUILD SIGNED TX
// node expects snake_case JSON (proto3 JSON mapping) with base64 bytes
// Transaction fields: message_type, msg{type_url,value}, signature{public_key,signature},
//   created_height, time, fee, memo, network_id, chain_id
// ═══════════════════════════════════════════
async function buildSigned(msgType,typeUrl,inner,meta){
  const txTime=BigInt(Date.now())*1000n;
  const p={txTime,fee:meta.fee||10000,height:meta.height||currentHeight,memo:'',netId:currentNetworkID,chainId:currentChainID};
  const sb=encSignBytes(msgType,typeUrl,inner,p);
  const sig=await blsSign(sb);
  return {
    type: msgType,
    msgTypeUrl: typeUrl,
    msgBytes: b2h(inner),
    signature: { publicKey: b2h(signerPubKey), signature: b2h(sig) },
    createdHeight: p.height,
    time: Number(txTime),
    fee: p.fee,
    memo: '',
    networkID: currentNetworkID,
    chainID: currentChainID,
  };
}

function buildUnsigned(msgType,typeUrl,inner,meta){
  const txTime=BigInt(Date.now())*1000n;
  return {
    message_type: msgType,
    msg: { type_url: typeUrl, value: b2b64(inner) },
    signature: null,
    created_height: meta.height||currentHeight,
    time: Number(txTime),
    fee: meta.fee||10000,
    memo: '',
    network_id: currentNetworkID||1,
    chain_id: currentChainID||1,
  };
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
  if(id==='create')updateCreateBreakdown();
  if(id==='predict')updatePredictBreakdown();
  if(id==='resolvers')loadResolvers();
  closeNav();
};

// ═══════════════════════════════════════════
// MOBILE NAV
// ═══════════════════════════════════════════
window.openNav=function(){document.getElementById('deskNav').classList.toggle('open');document.getElementById('mobNav').classList.toggle('open');};
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
    const d=await rpc('/v1/query/height',{});currentHeight=d.height||0;
    currentNetworkID=d.network_id||d.networkID||currentNetworkID;
    currentChainID=d.chain_id||d.chainID||currentChainID;
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
     'reg_addr','prop_resolver','dis_addr','cv_voter','rv_voter','tal_addr','fin_addr','sl_addr',
     'fo_resolver','rc_addr','ccf_addr','can_addr','unst_addr','cub_addr'].forEach(id=>{
      const el=document.getElementById(id);if(el&&!el.value)el.value=signerAddress;
    });
    const _ski=document.getElementById('sk_input');if(_ski)_ski.value='';
    refreshBalance();
    loadMyPredictions();
    toast('Key loaded — '+signerAddress);
  }catch(e){signerPrivKey=signerPubKey=signerAddress=null;toast('Key load failed: '+e.message,true);}
};
window.clearKey=function(){
  localStorage.removeItem('praxis_keystore');
  signerPrivKey=signerPubKey=signerAddress=null;
  document.getElementById('keyStatus').className='kstat';
  document.getElementById('keyStatus').textContent='○ No key loaded';
  document.getElementById('sk_derived').style.display='none';
  const _ski=document.getElementById('sk_input');if(_ski)_ski.value='';
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
    document.getElementById('w_balance').textContent=fmtPRX(d.amount||0);
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
    toast('⏳ Broadcasting — confirming in ~25s…');
    checkRPC();
    if(msgType==='create_market')setTimeout(loadMarkets,3000);
    setTimeout(async()=>{
      try{
        const d=await rpc('/v1/query/failed-txs',{address:signerAddress,perPage:20});
        const failed=(d.results||[]).find(r=>r.txHash===hash);
        if(failed){
          const code=failed.error?.code;
          const msg=failed.error?.msg||'Transaction failed';
          toast('✗ Failed — '+friendlyError(code,msg),true);
        } else {
          toast('✓ Confirmed — '+(hash.length>20?hash.slice(0,20)+'…':hash));
          if(msgType==='create_market'||msgType==='finalize_market')loadMarkets();
        }
      }catch(e){toast('✓ Submitted — could not confirm status',false);}
    },25000);
  }catch(e){toast(friendlyError(null,e.message),true);}
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
    const wbal=document.getElementById('w_balance');if(wbal)wbal.textContent=fmtPRX(bal);
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

    el.innerHTML = predictions.map(p => {
      const m = _allMarkets.find(x => x.id === p.marketId);
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

// ═══════════════════════════════════════════
// RENDER MARKET CARDS — Premium Design
// ═══════════════════════════════════════════
function resolverTier(addr) {
  const r = _resolverRegistry.get(addr);
  if (!r) return null;
  const estRRS = Math.min(r.proposalCount * 10, 999);
  if (estRRS >= 200) return {label:'Gold',   color:'#FFD700', icon:'★'};
  if (estRRS >= 50)  return {label:'Silver', color:'#C0C0C0', icon:'◆'};
  if (estRRS >= 1)   return {label:'Bronze', color:'#CD7F32', icon:'▲'};
  return {label:'Registered', color:'var(--text3)', icon:'○'};
}

window.renderMarketCards = function(markets) {
  if (!markets || markets.length === 0) return '<div class="alert ay">No markets found</div>';

  // category filter
  let filtered = markets;
  if (window._activeCat && window._activeCat !== 'all') {
    const cat = window._activeCat;
    filtered = markets.filter(m => {
      const q = (m.question || '').toLowerCase();
      if (cat === 'crypto')   return /btc|eth|crypto|bitcoin|ethereum|solana|token|defi|nft|blockchain/.test(q);
      if (cat === 'sports')   return /nba|nfl|fifa|soccer|football|tennis|golf|sports|league|match|game|win/.test(q);
      if (cat === 'politics') return /election|president|vote|congress|senate|government|policy|law|bill/.test(q);
      if (cat === 'finance')  return /stock|market|fed|rate|gdp|inflation|s&p|nasdaq|economy|oil|gold/.test(q);
      return true;
    });
  }

  if (filtered.length === 0) return '<div class="alert ay">No markets in this category</div>';

  return '<div class="mgrid">' + filtered.map(m => {
    const total = m.qYes + m.qNo;
    const yesPct = total > 0n ? Number((m.qYes * 100n) / total) : 50;
    const noPct  = 100 - yesPct;
    const mid    = m.marketId || m.txHash || '';
    const vol    = total > 0n ? fmtPRX(total) : '—';
    const exp    = m.expiry ? '#' + m.expiry.toString() : '—';
    const creator = (m.creator || '').slice(0,8) + '…';

    let statusHtml = '';
    let cardClass  = '';
    if (m.status === 0) {
      statusHtml = '<span class="spill sp-live"><span class="sp-dot"></span>LIVE</span>';
    } else if (m.status === 4) {
      statusHtml = '<span class="spill sp-proposed">PROPOSED</span>';
      cardClass = 'mexp';
    } else if (m.status === 5) {
      statusHtml = '<span class="spill sp-disputed">DISPUTED</span>';
      cardClass = 'mexp';
    } else if (m.status === 6) {
      statusHtml = '<span class="spill sp-finalized">FINALIZED</span>';
      cardClass = 'mfin';
    } else if (m.status === 1) {
      statusHtml = '<span class="spill sp-cancelled">CANCELLED</span>';
      cardClass = 'mcan';
    } else if (m.status === 8) {
      statusHtml = '<span class="spill sp-expired">EXPIRED</span>';
      cardClass = 'mexp';
    }

    const showBtns = m.status === 0;

    return `<div class="mcard ${cardClass}" onclick="openDetail(this.dataset.mid)" data-mid="${mid}">
  <div class="mcard-top">
    <div class="mcard-cat"><span class="mcard-cat-dot"></span>PRAXIS MARKET &nbsp;${statusHtml}</div>
    <div class="mcard-q">${esc(m.question || '(no question)')}</div>
    <div class="mcard-probs">
      <div>
        <div class="prob-yes">${yesPct}%</div>
        <div class="prob-labels"><span class="prob-lbl">YES</span></div>
      </div>
      <div class="prob-divider"></div>
      <div>
        <div class="prob-no">${noPct}%</div>
        <div class="prob-labels"><span class="prob-lbl">NO</span></div>
      </div>
    </div>
    <div class="btrack"><div class="byes" style="width:${yesPct}%"></div></div>
  </div>
  <div class="mcard-meta">
    <div class="meta-item"><span class="meta-lbl">Volume</span><span class="meta-val g">${vol}</span></div>
    <div class="meta-item"><span class="meta-lbl">Expires</span><span class="meta-val">${exp}</span></div>
    <div class="meta-item"><span class="meta-lbl">Creator</span><span class="meta-val">${creator}</span></div>
  </div>
  ${showBtns ? `<div class="mcard-actions">
    <button class="mcard-btn mcard-btn-yes" onclick="event.stopPropagation();fillP(${JSON.stringify(mid)},true)">BET YES</button>
    <button class="mcard-btn mcard-btn-no"  onclick="event.stopPropagation();fillP(${JSON.stringify(mid)},false)">BET NO</button>
  </div>` : ''}
</div>`;
  }).join('') + '</div>';
};

// ── Volume chip updater ──

// store markets globally for detail view
let _allMarkets = [];
let _resolverRegistry = new Map();
let _detailMarketId = null; // address -> {stake, proposalCount}

window.showDetail = function(marketId) {
  const m = _allMarkets.find(x => x.marketId === marketId || x.txHash === marketId);
  if (!m) return;
  const open = m.status === 0;
  const expired = m.status === 8;
  const cancelled = m.status === 1;
  const proposed = m.status === 4;
  const disputed = m.status === 5;
  const finalized = m.status === 6;
  const voided = m.status === 7;
  const resolved = m.status === 2;
  const total = m.qYes + m.qNo;
  const yesPct = total > 0n ? Number(m.qYes * 100n / total) : 50;
  const noPct = 100 - yesPct;
  const mid = m.marketId || m.txHash;

  document.getElementById('det-question').textContent = m.question;
  document.getElementById('det-qyes').textContent = fmtPRX(m.qYes) + ' PRX';
  document.getElementById('det-qno').textContent = fmtPRX(m.qNo) + ' PRX';
  document.getElementById('det-yes-pct').textContent = yesPct + '%';
  document.getElementById('det-no-pct').textContent = noPct + '%';
  document.getElementById('det-bar').style.width = yesPct + '%';
  document.getElementById('det-mid').textContent = mid;
  document.getElementById('det-creator').textContent = m.creator || '—';
  document.getElementById('det-total').textContent = fmtPRX(m.qYes + m.qNo) + ' PRX';
  document.getElementById('det-expiry').textContent = m.expiry ? 'blk #' + Number(m.expiry) : '—';
  const rulesRow = document.getElementById('det-rules-row');
  const rulesEl = document.getElementById('det-rules');
  if (rulesRow && rulesEl) {
    if (m.rules) { rulesEl.textContent = m.rules; rulesRow.style.display = ''; }
    else { rulesRow.style.display = 'none'; }
  }

  const resolverRow = document.getElementById('det-resolver-row');
  if (m.resolver) {
    resolverRow.style.display = '';
    const tier = m.resolver ? resolverTier(m.resolver) : null;
  const tierHtml = tier ? ' <span style="color:' + tier.color + '">' + tier.icon + ' ' + tier.label + '</span>' : '';
  document.getElementById('det-resolver').innerHTML = (m.resolver || '—') + tierHtml + (m.proposedOutcome !== undefined ? ' → proposed ' + (m.proposedOutcome ? '<span style="color:var(--green)">YES</span>' : '<span style="color:var(--red)">NO</span>') : '');
  } else {
    resolverRow.style.display = 'none';
  }

  const statusLabels = {0:'Open',1:'Cancelled',2:'Resolved',3:'Expired',4:'Proposed',5:'Disputed',6:'Finalized',7:'Voided',8:'Expired'};
  const statusClasses = {0:'sp-o',1:'sp-d',2:'sp-f',3:'sp-e',4:'sp-e',5:'sp-d',6:'sp-f',7:'sp-e',8:'sp-e'};
  document.getElementById('det-status-pill').innerHTML = '<div class="spill ' + (statusClasses[m.status]||'sp-f') + '"><span class="dot"></span>' + (statusLabels[m.status]||'Closed') + '</div>';

  const yesBtn = document.getElementById('det-bet-yes');
  const noBtn = document.getElementById('det-bet-no');
  if (open) {
    yesBtn.removeAttribute('disabled'); yesBtn.setAttribute('onclick', 'fillP(' + JSON.stringify(mid) + ', true)');
    noBtn.removeAttribute('disabled');  noBtn.setAttribute('onclick', 'fillP(' + JSON.stringify(mid) + ', false)');
  } else {
    yesBtn.setAttribute('disabled',''); noBtn.setAttribute('disabled','');
  }

  const proposeBtn = document.getElementById('det-propose-btn');
  const claimBtn   = document.getElementById('det-claim-btn');
  if (proposeBtn) {
    if (m.status === 8) {
      // COI-1: hide propose if signer is market creator
      const signerIsCreator = signerAddress && m.creator && signerAddress.toLowerCase() === m.creator.toLowerCase();
      // COI-2: hide propose if signer holds a position in this market
      const signerHasPosition = (() => {
        try {
          const txs = JSON.parse(localStorage.getItem('praxis_tx_cache') || '[]');
          return txs.some(tx =>
            tx.messageType === 'submit_prediction' &&
            tx.sender && tx.sender.toLowerCase() === (signerAddress||'').toLowerCase() &&
            tx.transaction && tx.transaction.msg &&
            (() => { try { return b2h(Uint8Array.from(atob(tx.transaction.msg.marketId||''), c=>c.charCodeAt(0))) === mid; } catch { return false; } })()
          );
        } catch { return false; }
      })();
      if (signerIsCreator) {
        proposeBtn.style.display = '';
        proposeBtn.disabled = true;
        proposeBtn.title = 'Market creators cannot propose outcomes for their own markets';
        proposeBtn.textContent = '⚖ Cannot Propose (Creator)';
      } else if (signerHasPosition) {
        proposeBtn.style.display = '';
        proposeBtn.disabled = true;
        proposeBtn.title = 'Forfeit your position before proposing';
        proposeBtn.textContent = '⚖ Forfeit Position First';
      } else {
        proposeBtn.style.display = '';
        proposeBtn.disabled = false;
        proposeBtn.textContent = '⚖ Propose Outcome';
        proposeBtn.setAttribute('onclick', 'fillPropose(' + JSON.stringify(mid) + ')');
      }
    } else {
      proposeBtn.style.display = 'none';
      proposeBtn.disabled = false;
      proposeBtn.textContent = '⚖ Propose Outcome';
    }
  }
  if (claimBtn) {
    if (m.status === 6) {
      claimBtn.style.display = '';
      claimBtn.textContent = '◎ Claim Winnings';
      claimBtn.setAttribute('onclick', 'fillC(' + JSON.stringify(mid) + ')');
    } else if (m.status === 1) {
      claimBtn.style.display = '';
      claimBtn.textContent = '◎ Claim Refund';
      claimBtn.setAttribute('onclick', 'fillC(' + JSON.stringify(mid) + ')');
    }
  }

  const reclaimBtn = document.getElementById('det-reclaim-btn');
  if (reclaimBtn) {
    if (m.status === 8 && currentHeight > Number(m.expiry) + 300) {
      reclaimBtn.style.display = '';
      reclaimBtn.setAttribute('onclick', 'fillReclaim(' + JSON.stringify(mid) + ')');
    } else {
      reclaimBtn.style.display = 'none';
    }
  }

  const forfeitBtn = document.getElementById('det-forfeit-btn');
  if (forfeitBtn) {
    if (m.status === 0 && signerAddress && signerAddress !== m.creator) {
      forfeitBtn.style.display = '';
      forfeitBtn.setAttribute('onclick', 'fillForfeit(' + JSON.stringify(mid) + ')');
    } else {
      forfeitBtn.style.display = 'none';
    }
  }

  const bannerCard = document.getElementById('det-banner-card');
  if (m.status === 8) {
    bannerCard.style.display = '';
    bannerCard.innerHTML = '<div class="mc-banner bnr"><span>⏳</span> Awaiting resolver proposal</div>';
  } else if (m.status === 4) {
    bannerCard.style.display = '';
    bannerCard.innerHTML = '<div class="mc-banner bnr"><span>🔎</span> Resolver: ' + (m.resolver ? m.resolver.slice(0,8) + '…' : '?') + ' — proposed ' + (m.proposedOutcome ? '<span style="color:var(--green)">YES</span>' : '<span style="color:var(--red)">NO</span>') + '</div>';
  } else if (m.status === 1) {
    bannerCard.style.display = '';
    bannerCard.innerHTML = '<div class="mc-banner bnr"><span>✕</span> Market cancelled — reclaim your stake</div>';
  } else if (m.status === 7) {
    bannerCard.style.display = '';
    bannerCard.innerHTML = '<div class="mc-banner bnr"><span>⚠</span> Market voided — full refund available</div>';
  } else {
    bannerCard.style.display = 'none';
  }

  _detailMarketId = mid;
  showPage('detail', null);
  setTimeout(()=>switchDetailTab('activity'), 50);
};
window.openDetail = window.showDetail;

window.fillP = (id, outcome) => {
  document.getElementById('p_mid').value = id;
  if (outcome !== undefined) { setOut(outcome); }
  showPage('predict', null);
};
window.fillC = id => { document.getElementById('cl_mid').value = id; showPage('claim', null); };

// ═══════════════════════════════════════════
// MARKETS PAGE
// ═══════════════════════════════════════════
function decVarint(buf,pos){let r=0n,s=0n;while(pos<buf.length){const b=BigInt(buf[pos++]);r|=(b&0x7fn)<<s;s+=7n;if(!(b&0x80n))break;}return{v:r,p:pos};}

let _activeTab = 'live';
const CLOSED_WINDOW = 20000; // blocks

window.switchTab = function(tab) {
  _activeTab = tab;
  document.querySelectorAll('.mtab').forEach(b => b.classList.remove('active'));
  const btn = document.getElementById('tab-' + tab);
  if (btn) btn.classList.add('active');
  renderCurrentTab();
};

function renderCurrentTab() {
  const el = document.getElementById('marketsList');
  if (!_allMarkets.length) return;

  let markets;
  if (_activeTab === 'live') {
    markets = _allMarkets.filter(m => m.status === 0);
  } else if (_activeTab === 'proposed') {
    markets = _allMarkets.filter(m => m.status === 4 || m.status === 5);
  } else {
    // closed — rolling window of last CLOSED_WINDOW blocks
    markets = _allMarkets.filter(m =>
      (m.status === 8 || m.status === 1 || m.status === 6 || m.status === 7 || m.status === 2 || m.status === 3) &&
      m.expiry && Number(m.expiry) >= (currentHeight - CLOSED_WINDOW)
    );
  }

  const countEl = document.getElementById('sb_c');
  if (countEl) countEl.textContent = _allMarkets.filter(m => m.status === 0).length;

  if (markets.length === 0) {
    const labels = {live:'No open markets yet', proposed:'No markets awaiting resolution', closed:'No recently closed markets'};
    el.innerHTML = '<div class="alert ay">' + (labels[_activeTab] || 'No markets') + '</div>';
    return;
  }
  el.innerHTML = window.renderMarketCards(markets);
}

window.loadMarkets = async function () {
  const el = document.getElementById('marketsList');
  const countEl = document.getElementById('sb_c');
  el.innerHTML = '<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;loading markets from chain</div>';
  try {
    await checkRPC();

    const heightResp = await rpc('/v1/query/height', {});
    const tipHeight = Number(heightResp.height || currentHeight || 1);
    const BATCH = 100;
    const CACHE_KEY = 'praxis_tx_cache';
    const CACHE_HEIGHT_KEY = 'praxis_scan_height';

    // load cache
    let allTxs = [];
    let scanFrom = 1;
    try {
      const cached = localStorage.getItem(CACHE_KEY);
      const cachedHeight = parseInt(localStorage.getItem(CACHE_HEIGHT_KEY) || '0');
      if (cached && cachedHeight > 0) {
        allTxs = JSON.parse(cached);
        scanFrom = cachedHeight + 1;
        el.innerHTML = '<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;Cache loaded to block ' + cachedHeight + ' — scanning new blocks…</div>';
      }
    } catch(e) { allTxs = []; scanFrom = 1; }

    // scan only new blocks
    if (scanFrom <= tipHeight) {
      for (let h = scanFrom; h <= tipHeight; h += BATCH) {
        const pct = Math.round(((h - scanFrom) / Math.max(tipHeight - scanFrom, 1)) * 100);
        el.innerHTML = '<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;Scanning blocks ' + h + ' / ' + tipHeight + ' &nbsp;<span style="color:var(--green)">' + pct + '%</span></div>';
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
      // save cache
      try {
        localStorage.setItem(CACHE_KEY, JSON.stringify(allTxs));
        localStorage.setItem(CACHE_HEIGHT_KEY, String(tipHeight));
      } catch(e) {}
    }

    const marketsMap = new Map();

    for (const tx of allTxs) {
      if (tx.messageType !== 'create_market') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const question = msg.question || '';
      const rules    = msg.rules || '';
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
          rules: rules || '',
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
      m.status = 4;
      m.resolver = resolver;
      m.proposedOutcome = msg.proposedOutcome;
    }

    // Build resolver registry
    for (const tx of allTxs) {
      if (tx.messageType !== 'register_resolver') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const addr = tx.sender || '';
      let stake = BigInt(msg.stakeAmount || msg.stake_amount || 0);
      if (!_resolverRegistry.has(addr)) {
        _resolverRegistry.set(addr, { stake, proposalCount: 0, rrs: 10 });
      } else {
        _resolverRegistry.get(addr).stake = stake;
      }
    }
    for (const tx of allTxs) {
      if (tx.messageType !== 'propose_outcome') continue;
      const addr = tx.sender || '';
      if (_resolverRegistry.has(addr)) {
        _resolverRegistry.get(addr).proposalCount++;
      }
    }
    // sync rrs estimate into registry
    _resolverRegistry.forEach(r => { r.rrs = Math.min(10 + r.proposalCount * 10, 999); });

    for (const tx of allTxs) {
      if (tx.messageType !== 'finalize_market') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || '';
      let marketId = rawMid;
      try { const b = Uint8Array.from(atob(rawMid), c => c.charCodeAt(0)); marketId = b2h(b); } catch(e) {}
      if (!marketId || !marketsMap.has(marketId)) continue;
      marketsMap.get(marketId).status = 6;
    }

    for (const tx of allTxs) {
      if (tx.messageType !== 'file_dispute') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || '';
      let marketId = rawMid;
      try { const b = Uint8Array.from(atob(rawMid), c => c.charCodeAt(0)); marketId = b2h(b); } catch(e) {}
      if (!marketId || !marketsMap.has(marketId)) continue;
      marketsMap.get(marketId).status = 5;
    }

    for (const tx of allTxs) {
      if (tx.messageType !== 'cancel_market') continue;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || '';
      let marketId = rawMid;
      try { const b = Uint8Array.from(atob(rawMid), c => c.charCodeAt(0)); marketId = b2h(b); } catch(e) {}
      if (!marketId || !marketsMap.has(marketId)) continue;
      marketsMap.get(marketId).status = 1;
    }

    const markets = [...marketsMap.values()];
    for (const m of markets) {
      if (m.expiry && currentHeight > Number(m.expiry) && m.status === 0) m.status = 8;
    }

    _allMarkets = markets;
    checkRoles();
    renderCurrentTab();

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
  const amt=parseInt(document.getElementById('s_amount').value)*1000000;
  const fee=parseInt(document.getElementById('s_fee').value)||10000;
  addr40(from,'From');addr40(to,'To');if(!amt||amt<=0)throw new Error('Amount > 0 required');
  showPL('so','sp',buildUnsigned('send','type.googleapis.com/types.MessageSend',encSend(from,to,amt),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_send=async function(){try{
  const from=document.getElementById('s_from').value.trim().toLowerCase();
  const to=document.getElementById('s_to').value.trim().toLowerCase();
  const amt=parseInt(document.getElementById('s_amount').value)*1000000;
  const fee=parseInt(document.getElementById('s_fee').value)||10000;
  addr40(from,'From');addr40(to,'To');if(!amt||amt<=0)throw new Error('Amount > 0');
  await doSubmit('send','type.googleapis.com/types.MessageSend',encSend(from,to,amt),{fee},'btn_send','pend_send');
}catch(e){toast(e.message,true);}};

// ── CREATE MARKET
window.build_create=function(){try{
  const q=document.getElementById('c_question').value.trim();
  const cr=document.getElementById('c_creator').value.trim().toLowerCase();
  const b0=parseInt(document.getElementById('c_b0').value)*1000000;
  const exp=parseInt(document.getElementById('c_expiry').value)||currentHeight+1000;
  const fee=parseInt(document.getElementById('c_fee').value)||10000;
  let nonce=document.getElementById('c_nonce').value;
  if(!nonce)nonce=BigInt(Date.now())*1000n;
  else nonce=parseInt(nonce);
  const rules=document.getElementById('c_rules').value.trim();
  if(!q)throw new Error('Question required');addr40(cr,'Creator');
  showPL('co','cp',buildUnsigned('create_market','type.googleapis.com/types.MessageCreateMarket',encCreate(cr,b0,exp,nonce,q,rules),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
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
window.signAndSubmit_create=async function(){try{
  const q=document.getElementById('c_question').value.trim();
  const cr=document.getElementById('c_creator').value.trim().toLowerCase();
  const b0=parseInt(document.getElementById('c_b0').value)*1000000;
  const exp=parseInt(document.getElementById('c_expiry').value)||currentHeight+1000;
  const fee=parseInt(document.getElementById('c_fee').value)||10000;
  let nonce=document.getElementById('c_nonce').value;
  if(!nonce)nonce=BigInt(Date.now())*1000n;
  else nonce=parseInt(nonce);
  const rules=document.getElementById('c_rules').value.trim();
  if(!q)throw new Error('Question required');addr40(cr,'Creator');
  await doSubmit('create_market','type.googleapis.com/types.MessageCreateMarket',encCreate(cr,b0,exp,nonce,q,rules),{fee},'btn_create','pend_create');
}catch(e){toast(e.message,true);}};

// ── SUBMIT PREDICTION
window.build_predict=function(){try{
  const mid=document.getElementById('p_mid').value.trim().toLowerCase();mid40(mid);
  const bettor=document.getElementById('p_bettor').value.trim().toLowerCase();addr40(bettor,'Bettor');
  const sharesInput=parseInt(document.getElementById("p_shares").value);
  const shares=sharesInput*1000000;
  const mc=parseInt(document.getElementById('p_maxcost').value)*1000000;
  const fee=parseInt(document.getElementById('p_fee').value)||10000;
  if(sharesInput<1)throw new Error("Shares min 1 PRX");
  showPL('po','pp',buildUnsigned('submit_prediction','type.googleapis.com/types.MessageSubmitPrediction',encPredict(mid,bettor,selectedOut,shares,mc),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
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
window.signAndSubmit_predict=async function(){try{
  const mid=document.getElementById('p_mid').value.trim().toLowerCase();mid40(mid);
  const bettor=document.getElementById('p_bettor').value.trim().toLowerCase();addr40(bettor,'Bettor');
  const sharesInput=parseInt(document.getElementById("p_shares").value);
  const shares=sharesInput*1000000;
  const mc=parseInt(document.getElementById('p_maxcost').value)*1000000;
  const fee=parseInt(document.getElementById('p_fee').value)||10000;
  if(sharesInput<1)throw new Error("Shares min 1 PRX");
  await doSubmit('submit_prediction','type.googleapis.com/types.MessageSubmitPrediction',encPredict(mid,bettor,selectedOut,shares,mc),{fee},'btn_predict','pend_predict');
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
  const stake=parseInt(document.getElementById('reg_stake').value)*1000000;
  const fee=parseInt(document.getElementById('reg_fee').value)||10000;
  if(stake<500000000000)throw new Error('Stake min 500,000 PRX');
  showPL('rego','regp',buildUnsigned('register_resolver','type.googleapis.com/types.MessageRegisterResolver',encRegister(addr,stake),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_register=async function(){try{
  const addr=document.getElementById('reg_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
  const stake=parseInt(document.getElementById('reg_stake').value)*1000000;
  const fee=parseInt(document.getElementById('reg_fee').value)||10000;
  if(stake<500000000000)throw new Error('Stake min 500,000 PRX');
  await doSubmit('register_resolver','type.googleapis.com/types.MessageRegisterResolver',encRegister(addr,stake),{fee},'btn_register','pend_register');
}catch(e){toast(e.message,true);}};

// ── PROPOSE OUTCOME
window.build_propose=function(){try{
  const mid=document.getElementById('prop_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('prop_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const bond=parseInt(document.getElementById('prop_bond').value)*1000000;
  const fee=parseInt(document.getElementById('prop_fee').value)||10000;
  showPL('propo','propp',buildUnsigned('propose_outcome','type.googleapis.com/types.MessageProposeOutcome',encPropose(mid,res,propOut,bond),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_propose=async function(){try{
  const mid=document.getElementById('prop_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('prop_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const bond=parseInt(document.getElementById('prop_bond').value)*1000000;
  const fee=parseInt(document.getElementById('prop_fee').value)||10000;
  await doSubmit('propose_outcome','type.googleapis.com/types.MessageProposeOutcome',encPropose(mid,res,propOut,bond),{fee},'btn_propose','pend_propose');
}catch(e){toast(e.message,true);}};

// ── FILE DISPUTE
window.build_dispute=function(){try{
  const mid=document.getElementById('dis_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('dis_addr').value.trim().toLowerCase();addr40(addr,'Disputer');
  const bond=parseInt(document.getElementById('dis_bond').value)*1000000;
  const fee=parseInt(document.getElementById('dis_fee').value)||10000;
  showPL('diso','disp',buildUnsigned('file_dispute','type.googleapis.com/types.MessageFileDispute',encDispute(mid,addr,bond),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};
window.signAndSubmit_dispute=async function(){try{
  const mid=document.getElementById('dis_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('dis_addr').value.trim().toLowerCase();addr40(addr,'Disputer');
  const bond=parseInt(document.getElementById('dis_bond').value)*1000000;
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
      `<div class="cm-row"><span class="cm-l">${l}</span><span class="cm-v ${cls||''}">${v}</span></div>`
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
        ['B0 Liquidity', v('c_b0').toLocaleString()+' PRX', 'g'],
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
    signAndSubmit_claim: () => [
      'Claim Winnings', [
        ['Market ID',   (document.getElementById('cl_mid')?.value||'').slice(0,16)+'…', ''],
        ['Claimant',    (document.getElementById('cl_addr')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_register: () => [
      'Register Resolver', [
        ['Address',     (document.getElementById('reg_addr')?.value||'').slice(0,16)+'…', ''],
        ['Stake', (parseInt(document.getElementById('reg_stake')?.value||0)).toLocaleString()+' PRX', 'g'],
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
    signAndSubmit_claimcreator: () => [
      'Claim Creator Fee', [
        ['Market ID', (document.getElementById('ccf_mid')?.value||'').slice(0,16)+'…', ''],
        ['Creator',   (document.getElementById('ccf_addr')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_cancel: () => [
      'Cancel Market', [
        ['Market ID', (document.getElementById('can_mid')?.value||'').slice(0,16)+'…', ''],
        ['Creator',   (document.getElementById('can_addr')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_unstake_resolver: () => [
      'Unstake Resolver', [
        ['Resolver', (document.getElementById('unst_addr')?.value||'').slice(0,16)+'…', ''],
        ['Amount',   (parseInt(document.getElementById('unst_amount')?.value||0)).toLocaleString()+' PRX (0 = full exit)', ''],
      ]
    ],
    signAndSubmit_claim_unbonded: () => [
      'Claim Unbonded Stake', [
        ['Resolver', (document.getElementById('cub_addr')?.value||'').slice(0,16)+'…', ''],
      ]
    ],
    signAndSubmit_send: () => [
      'Send $PRX', [
        ['To',    (document.getElementById('s_to')?.value||'').slice(0,16)+'…', ''],
        ['Amount', v('s_amount').toLocaleString()+' PRX', 'g'],
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
const _niHost=document.getElementById('ni_host');if(_niHost)_niHost.value=getRPCHost();
buildMobNav();
checkRPC();
setInterval(checkRPC,12000);

// ═══════════════════════════════════════════
// KEYSTORE — AES-GCM + Argon2id (Canopy official format)
// Uses argon2-bundled.min.js (must be served alongside app.js)
// ═══════════════════════════════════════════

// Argon2id params matching Canopy CLI keystore
const ARGON2_TIME    = 3;
const ARGON2_MEM     = 65536; // 64 MB
const ARGON2_THREADS = 4;
const ARGON2_KEYLEN  = 32;

async function deriveKeyArgon2(password, salt) {
  // argon2-bundled exposes window.argon2
  if (!window.argon2) throw new Error('Argon2 library not loaded — ensure argon2-bundled.min.js is present');
  const result = await window.argon2.hash({
    pass: password,
    salt: salt,           // Uint8Array
    time: window._argon2Override?.time || ARGON2_TIME,
    mem:  window._argon2Override?.mem  || ARGON2_MEM,
    hashLen: window._argon2Override?.keylen || ARGON2_KEYLEN,
    parallelism: window._argon2Override?.threads || ARGON2_THREADS,
    type: window.argon2.ArgonType.Argon2id,
  });
  // result.hash is Uint8Array of 32 bytes — import as AES-GCM key
  return crypto.subtle.importKey('raw', result.hash, { name: 'AES-GCM' }, false, ['encrypt', 'decrypt']);
}

async function encryptKey(privKeyBytes, password) {
  const salt = crypto.getRandomValues(new Uint8Array(16));
  const iv   = crypto.getRandomValues(new Uint8Array(12));
  const key  = await deriveKeyArgon2(password, salt);
  const enc  = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, privKeyBytes);
  return {
    kdf:  'argon2id',
    salt: b2h(salt),
    iv:   b2h(iv),
    encrypted: b2h(new Uint8Array(enc)),
    argon2: { time: ARGON2_TIME, mem: ARGON2_MEM, threads: ARGON2_THREADS, keylen: ARGON2_KEYLEN },
  };
}

async function decryptKey(encrypted, iv, salt, password, kdf) {
  let key, nonce;
  if (kdf === 'canopy') {
    // Canopy CLI format: Argon2i (not id), mem=32MB, keyLen=32, nonce=key[:12]
    if (!window.argon2) throw new Error('Argon2 library not loaded');
    const result = await window.argon2.hash({
      pass: password, salt: h2b(salt),
      time: 3, mem: 32768, hashLen: 32,
      parallelism: 4, type: window.argon2.ArgonType.Argon2i,
    });
    const keyBytes = result.hash;  // 32 bytes
    nonce = keyBytes.slice(0, 12); // nonce = key[:12]
    key = await crypto.subtle.importKey('raw', keyBytes, { name: 'AES-GCM' }, false, ['decrypt']);
  } else if (!kdf || kdf === 'argon2id') {
    key = await deriveKeyArgon2(password, h2b(salt));
    nonce = h2b(iv);
  } else {
    // legacy PBKDF2 fallback
    const enc = new TextEncoder();
    const km = await crypto.subtle.importKey('raw', enc.encode(password), 'PBKDF2', false, ['deriveKey']);
    key = await crypto.subtle.deriveKey(
      { name: 'PBKDF2', salt: h2b(salt), iterations: 200000, hash: 'SHA-256' },
      km, { name: 'AES-GCM', length: 256 }, false, ['encrypt', 'decrypt']
    );
    nonce = h2b(iv);
  }
  const dec = await crypto.subtle.decrypt({ name: 'AES-GCM', iv: nonce }, key, h2b(encrypted));
  return new Uint8Array(dec);
}

window.createKeystore = async function() {
  const pw  = document.getElementById('ks_new_pw').value;
  const pw2 = document.getElementById('ks_new_pw2').value;
  if (!pw) return toast('Enter a password', true);
  if (pw !== pw2) return toast('Passwords do not match', true);
  if (pw.length < 8) return toast('Password must be at least 8 characters', true);

  try {
    // generate new BLS private key (valid scalar)
    const privBytes = bls12_381.utils.randomPrivateKey();
    const pubKey    = bls12_381.getPublicKey(privBytes);
    const hash      = await crypto.subtle.digest('SHA-256', pubKey);
    const address   = b2h(new Uint8Array(hash).slice(0, 20));

    const { salt, iv, encrypted } = await encryptKey(privBytes, pw);

    const keystore = {
      version: 1,
      kdf: 'argon2id',
      publicKey: b2h(pubKey),
      keyAddress: address,
      salt, iv, encrypted,
      argon2: { time: ARGON2_TIME, mem: ARGON2_MEM, threads: ARGON2_THREADS, keylen: ARGON2_KEYLEN },
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

window.checkSavedKeystore = function() {
  const saved = localStorage.getItem('praxis_keystore');
  const wrap = document.getElementById('ks_quick_wrap');
  if (!wrap) return;
  if (saved) {
    const raw = JSON.parse(saved);
    const addr = raw.keyAddress || '?';
    document.getElementById('ks_quick_addr').textContent = addr.slice(0,8) + '…' + addr.slice(-6);
    wrap.style.display = '';
  } else {
    wrap.style.display = 'none';
  }
};

window.quickUnlock = async function() {
  const pw = document.getElementById('ks_quick_pw').value;
  if (!pw) return toast('Enter password', true);
  const saved = localStorage.getItem('praxis_keystore');
  if (!saved) return toast('No saved keystore', true);
  try {
    const raw = JSON.parse(saved);
    if (!raw.encrypted || !raw.salt || !raw.iv || !raw.publicKey) throw new Error('Invalid saved keystore');
    if (raw.argon2) { window._argon2Override = raw.argon2; } else { window._argon2Override = null; }
    let privBytes;
    try {
      privBytes = await decryptKey(raw.encrypted, raw.iv, raw.salt, pw, raw.kdf || 'argon2id');
    } catch(e) {
      privBytes = await decryptKey(raw.encrypted, raw.iv, raw.salt, pw, 'pbkdf2');
    }
    let pubKey = bls12_381.getPublicKey(privBytes);
    if (b2h(pubKey) !== raw.publicKey) {
      try { privBytes = await decryptKey(raw.encrypted, raw.iv, raw.salt, pw, 'pbkdf2'); pubKey = bls12_381.getPublicKey(privBytes); } catch(e2) {}
    }
    if (b2h(pubKey) !== raw.publicKey) throw new Error('Wrong password');
    const hash = await crypto.subtle.digest('SHA-256', pubKey);
    const address = b2h(new Uint8Array(hash).slice(0, 20));
    signerPrivKey = privBytes;
    signerPubKey = pubKey;
    signerAddress = address;
    updateSignerUI();
    toast('Session restored — ' + address.slice(0,8) + '…');
    document.getElementById('ks_quick_pw').value = '';
  } catch(e) { toast('Unlock failed: ' + e.message, true); }
};

window.importKeystore = async function() {
  const pw   = document.getElementById('ks_imp_pw').value;
  const file = document.getElementById('ks_imp_file').files[0];
  if (!file) return toast('Select a keystore file', true);
  if (!pw)   return toast('Enter password', true);

  try {
    const text = await file.text();
    const raw  = JSON.parse(text);

    // Praxis flat format
    if (!raw.encrypted || !raw.salt || !raw.iv || !raw.publicKey) throw new Error('Invalid keystore file');
    if (raw.argon2) { window._argon2Override = raw.argon2; } else { window._argon2Override = null; }
    let privBytes;
    try {
      privBytes = await decryptKey(raw.encrypted, raw.iv, raw.salt, pw, raw.kdf || 'argon2id');
    } catch(e) {
      privBytes = await decryptKey(raw.encrypted, raw.iv, raw.salt, pw, 'pbkdf2');
    }
    let pubKey = bls12_381.getPublicKey(privBytes);
    if (b2h(pubKey) !== raw.publicKey) {
      // try pbkdf2 fallback
      try {
        privBytes = await decryptKey(raw.encrypted, raw.iv, raw.salt, pw, 'pbkdf2');
        pubKey = bls12_381.getPublicKey(privBytes);
      } catch(e2) {}
    }
    if (b2h(pubKey) !== raw.publicKey) throw new Error('Wrong password or corrupted keystore');
    const hash    = await crypto.subtle.digest('SHA-256', pubKey);
    const address = b2h(new Uint8Array(hash).slice(0, 20));
    signerPrivKey = privBytes;
    signerPubKey  = pubKey;
    signerAddress = address;
    updateSignerUI();
    toast('Keystore unlocked — ' + address.slice(0,8) + '…');
    localStorage.setItem('praxis_keystore', JSON.stringify(raw));
    document.getElementById('ks_imp_pw').value = '';
    document.getElementById('ks_imp_file').value = '';
    checkSavedKeystore();
  } catch(e) { console.error('Import failed full error:', e); toast('Import failed: ' + e.message, true); }
};

function updateSignerUI() {
  document.getElementById('keyStatus').className = 'kstat loaded';
  document.getElementById('keyStatus').textContent = '✓ loaded — ' + signerAddress.slice(0,16) + '…';
  document.getElementById('sk_derived').style.display = 'block';
  document.getElementById('sk_pub').textContent = b2h(signerPubKey);
  document.getElementById('sk_addr').textContent = signerAddress;
  ['c_creator','p_bettor','r_resolver','cl_addr','s_from','w_addr','ft_addr',
   'reg_addr','prop_resolver','dis_addr','cv_voter','rv_voter','tal_addr','fin_addr','sl_addr',
   'fo_resolver','rc_addr','ccf_addr','can_addr','unst_addr','cub_addr'].forEach(id => {
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

// ═══════════════════════════════════════════
// SCAN CACHE
// ═══════════════════════════════════════════
window.clearScanCache = function() {
  localStorage.removeItem('praxis_tx_cache');
  localStorage.removeItem('praxis_scan_height');
  toast('Cache cleared — full rescan on next refresh');
  loadMarkets();
};

// ═══════════════════════════════════════════
// ERROR CODES
// ═══════════════════════════════════════════
const PRAXIS_ERRORS = {
  124: 'Market has not expired yet — propose_outcome is only callable after expiry.',
  181: 'Cannot finalize — dispute window is still open. Wait for the dispute period to close.',
  4001: 'Resolver has an open position in this market. Use Forfeit Position before proposing.',
  4002: 'Market creator cannot act as resolver for their own market.',
  4003: 'This prediction exceeds the 20% position cap for one side. Try a smaller amount.',
  4010: 'Storage error — please try again or contact support.',
  195: 'Dispute panel could not be formed',
  196: 'This market is not eligible for reclaim',
  197: "Reclaim window hasn't opened yet — wait 300 blocks after expiry",
  198: 'Nothing to reclaim for this wallet',
  199: 'You hold a position in this market and cannot act as resolver. Transfer or forfeit your shares first.',
  200: 'The market creator cannot resolve their own market.',
  201: 'This prediction would exceed the 20% per-address position cap for this market. Try a smaller amount.',
  202: 'Resolver stake below minimum — 500,000 PRX required.',
  203: 'Cooldown period has not elapsed yet.',
  204: 'Pool is empty — nothing to claim.',
  205: 'Market is not finalized.',
  207: 'Resolver RRS is zero — not eligible for rewards.',
  208: 'No successful resolutions in this epoch.',
  210: 'Active proposal exists — unstake not allowed.',
  211: 'Resolver is not active.',
  212: 'No unbonding stake to claim.',
  213: 'Unbonding period not complete.',
  214: 'Resolver record not found.',
  215: 'Market has expired.',
  216: 'Market has positions — cannot cancel.',
  217: 'Unbonding already pending.',
};

function friendlyError(code, msg) {
  if (!code && msg) { const m = msg.match(/"code":(\d+)/); if (m) code = parseInt(m[1]); }
  if (code && PRAXIS_ERRORS[code]) return PRAXIS_ERRORS[code];
  return msg || 'Unknown error';
}

// ═══════════════════════════════════════════
// RECLAIM STAKE
// ═══════════════════════════════════════════
window.build_reclaim = function() {
  try {
    const mid  = document.getElementById('rc_mid').value.trim().toLowerCase();
    const addr = document.getElementById('rc_addr').value.trim().toLowerCase();
    const fee  = parseInt(document.getElementById('rc_fee').value) || 10000;
    addr40(mid, 'Market ID'); addr40(addr, 'Claimant Address');
    showPL('rco','rcp', buildUnsigned('reclaim_stake','type.googleapis.com/types.MessageReclaimStake', encReclaim(mid,addr),{fee}));
    toast('Payload built');
  } catch(e) { toast(e.message, true); }
};

window.signAndSubmit_reclaim = async function() {
  const mid  = document.getElementById('rc_mid').value.trim().toLowerCase();
  const addr = document.getElementById('rc_addr').value.trim().toLowerCase();
  const fee  = parseInt(document.getElementById('rc_fee').value) || 10000;
  try {
    addr40(mid,'Market ID'); addr40(addr,'Claimant Address');
  } catch(e) { return toast(e.message, true); }
  await doSubmit('reclaim_stake','type.googleapis.com/types.MessageReclaimStake', encReclaim(mid,addr),{fee},'btn_reclaim','pend_reclaim');
};

window.fillReclaim = function(id) {
  document.getElementById('rc_mid').value = id;
  if (signerAddress) document.getElementById('rc_addr').value = signerAddress;
  showPage('reclaim', null);
};

// ═══════════════════════════════════════════
// ROLE-BASED SIDEBAR
// ═══════════════════════════════════════════
async function checkRoles() {
  if (!signerAddress) return;

  // Superadmin — full access regardless of role
  const SUPERADMIN = '8e14dc0ce537f1c75036f11d7495d60882aa6731';
  if (signerAddress.toLowerCase() === SUPERADMIN) {
    document.getElementById('nav-resolver-section').style.display = '';
    document.querySelectorAll('.nav-resolver-item').forEach(el => el.style.display = '');
    document.getElementById('nav-admin-section').style.display = '';
    document.querySelectorAll('.nav-admin-item').forEach(el => el.style.display = '');
    return;
  }

  // Admin section — superadmin only (handled above)
  document.getElementById('nav-admin-section').style.display = 'none';
  document.querySelectorAll('.nav-admin-item').forEach(el => el.style.display = 'none');

  // Check RESOLVER — has a register_resolver tx in scanned data
  const isResolver = _allMarkets.length >= 0 && (() => {
    const cache = localStorage.getItem('praxis_tx_cache');
    if (!cache) return false;
    try {
      const txs = JSON.parse(cache);
      return txs.some(tx =>
        tx.messageType === 'register_resolver' &&
        tx.sender && tx.sender.toLowerCase() === signerAddress.toLowerCase()
      );
    } catch { return false; }
  })();

  document.getElementById('nav-resolver-section').style.display = isResolver ? '' : 'none';
  document.querySelectorAll('.nav-resolver-item').forEach(el => el.style.display = isResolver ? '' : 'none');
}

// Run role check after key load and after markets load
const _origUpdateSignerUI = updateSignerUI;
updateSignerUI = function() {
  _origUpdateSignerUI();
  checkRoles();
};

// ═══════════════════════════════════════════
// COI-3 POSITION CAP CHECK
// ═══════════════════════════════════════════
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

  // find market in _allMarkets
  const m = _allMarkets.find(x => x.marketId === mid || x.txHash === mid);
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
window.build_forfeit = function() {
  const mid      = document.getElementById('fo_mid').value.trim().toLowerCase();
  const resolver = document.getElementById('fo_resolver').value.trim().toLowerCase();
  const fee      = parseInt(document.getElementById('fo_fee').value) || 10000;
  mid40(mid); addr40(resolver, 'Resolver Address');
  const inner = encForfeit(mid, resolver);
  showPL('foo','fop', buildUnsigned('forfeit_position','type.googleapis.com/types.MessageForfeitPosition',inner,{fee}));
  toast('Payload built');
};

window.signAndSubmit_forfeit = async function() {
  const mid      = document.getElementById('fo_mid').value.trim().toLowerCase();
  const resolver = document.getElementById('fo_resolver').value.trim().toLowerCase();
  const fee      = parseInt(document.getElementById('fo_fee').value) || 10000;
  try { mid40(mid); addr40(resolver, 'Resolver Address'); } catch(e) { return toast(e.message, true); }
  const inner = encForfeit(mid, resolver);
  await doSubmit('forfeit_position','type.googleapis.com/types.MessageForfeitPosition',inner,{fee},'btn_forfeit','pend_forfeit');
};

window.fillForfeit = function(id) {
  document.getElementById('fo_mid').value = id;
  if (signerAddress) document.getElementById('fo_resolver').value = signerAddress;
  showPage('forfeit', null);
};
window.fillPropose = function(id) {
  document.getElementById('prop_mid').value = id;
  if (signerAddress) document.getElementById('prop_resolver').value = signerAddress;
  showPage('propose', null);
  setTimeout(updateMinBondHint, 50);
};

window.updateMinBondHint = function() {
  const mid = document.getElementById('prop_mid').value.trim().toLowerCase();
  const hint = document.getElementById('prop_bond_hint');
  const bondEl = document.getElementById('prop_bond');
  if (!hint) return;
  if (!mid || mid.length !== 40) {
    hint.textContent = 'Enter Market ID to compute min bond';
    hint.style.color = '';
    return;
  }
  const m = _allMarkets.find(x => x.marketId === mid || x.txHash === mid);
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
  if (bondEl && parseFloat(bondEl.value) < minBond) bondEl.value = Math.ceil(minBond);
};
checkSavedKeystore();

// ═══════════════════════════════════════════
// CLAIM CREATOR FEE
// ═══════════════════════════════════════════
function encClaimCreatorFee(mid,creator){return cat(bf(1,h2b(mid)),bf(2,h2b(creator)));}

window.build_claimcreator=function(){
  try{
    const mid=document.getElementById('ccf_mid').value.trim().toLowerCase();mid40(mid);
    const addr=document.getElementById('ccf_addr').value.trim().toLowerCase();addr40(addr,'Creator');
    const fee=parseInt(document.getElementById('ccf_fee').value)||10000;
    showPL('ccfo','ccfp',buildUnsigned('claim_creator_fee','type.googleapis.com/types.MessageClaimCreatorFee',encClaimCreatorFee(mid,addr),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};
window.signAndSubmit_claimcreator=async function(){
  try{
    const mid=document.getElementById('ccf_mid').value.trim().toLowerCase();mid40(mid);
    const addr=document.getElementById('ccf_addr').value.trim().toLowerCase();addr40(addr,'Creator');
    const fee=parseInt(document.getElementById('ccf_fee').value)||10000;
    await doSubmit('claim_creator_fee','type.googleapis.com/types.MessageClaimCreatorFee',encClaimCreatorFee(mid,addr),{fee},'btn_claimcreator','pend_claimcreator');
  }catch(e){toast(e.message,true);}
};
window.fillClaimCreator=function(id){
  document.getElementById('ccf_mid').value=id;
  if(signerAddress)document.getElementById('ccf_addr').value=signerAddress;
  showPage('claimcreator',null);
};

// ═══════════════════════════════════════════
// CANCEL MARKET
// ═══════════════════════════════════════════
function encCancelMarket(mid,creator){return cat(bf(1,h2b(mid)),bf(2,h2b(creator)));}

window.build_cancel=function(){
  try{
    const mid=document.getElementById('can_mid').value.trim().toLowerCase();mid40(mid);
    const addr=document.getElementById('can_addr').value.trim().toLowerCase();addr40(addr,'Creator');
    const fee=parseInt(document.getElementById('can_fee').value)||10000;
    showPL('cano','canp',buildUnsigned('cancel_market','type.googleapis.com/types.MessageCancelMarket',encCancelMarket(mid,addr),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};
window.signAndSubmit_cancel=async function(){
  try{
    const mid=document.getElementById('can_mid').value.trim().toLowerCase();mid40(mid);
    const addr=document.getElementById('can_addr').value.trim().toLowerCase();addr40(addr,'Creator');
    const fee=parseInt(document.getElementById('can_fee').value)||10000;
    await doSubmit('cancel_market','type.googleapis.com/types.MessageCancelMarket',encCancelMarket(mid,addr),{fee},'btn_cancel','pend_cancel');
  }catch(e){toast(e.message,true);}
};

// ═══════════════════════════════════════════
// UNSTAKE RESOLVER
// ═══════════════════════════════════════════
window.build_unstake_resolver=function(){
  try{
    const addr=document.getElementById('unst_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const amount=parseInt(document.getElementById('unst_amount').value||'0');
    const amountU=BigInt(amount)*1000000n;
    const fee=parseInt(document.getElementById('unst_fee').value)||10000;
    showPL('unsto','unstp',buildUnsigned('unstake_resolver','type.googleapis.com/types.MessageUnstakeResolver',encUnstakeResolver(addr,amountU),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};
window.signAndSubmit_unstake_resolver=async function(){
  try{
    const addr=document.getElementById('unst_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const amount=parseInt(document.getElementById('unst_amount').value||'0');
    const amountU=BigInt(amount)*1000000n;
    const fee=parseInt(document.getElementById('unst_fee').value)||10000;
    await doSubmit('unstake_resolver','type.googleapis.com/types.MessageUnstakeResolver',encUnstakeResolver(addr,amountU),{fee},'btn_unstake_resolver','pend_unstake_resolver');
  }catch(e){toast(e.message,true);}
};

// ═══════════════════════════════════════════
// CLAIM UNBONDED STAKE
// ═══════════════════════════════════════════
window.build_claim_unbonded=function(){
  try{
    const addr=document.getElementById('cub_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const fee=parseInt(document.getElementById('cub_fee').value)||10000;
    showPL('cubo','cubp',buildUnsigned('claim_unbonded_stake','type.googleapis.com/types.MessageClaimUnbondedStake',encClaimUnbonded(addr),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};
window.signAndSubmit_claim_unbonded=async function(){
  try{
    const addr=document.getElementById('cub_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const fee=parseInt(document.getElementById('cub_fee').value)||10000;
    await doSubmit('claim_unbonded_stake','type.googleapis.com/types.MessageClaimUnbondedStake',encClaimUnbonded(addr),{fee},'btn_claim_unbonded','pend_claim_unbonded');
  }catch(e){toast(e.message,true);}
};

// ═══════════════════════════════════════════
// RESOLVER RECORD STATE QUERY
// prefix 0x16 + len + addr bytes
// ═══════════════════════════════════════════
function buildResolverKey(addrHex){
  const addr=h2b(addrHex);
  const key=new Uint8Array(1+1+addr.length);
  key[0]=0x16; key[1]=addr.length; key.set(addr,2);
  return b2h(key);
}

function decodeResolverRecord(hexData){
  const buf=h2b(hexData);
  let pos=0;
  const rec={stake:0n,rrs:0n,registeredAt:0n,successfulResolutions:0n,lastClaimedEpoch:0n};
  while(pos<buf.length){
    const {v:tagV,p:p1}=decVarint(buf,pos);pos=p1;
    const fn=Number(tagV>>3n),wt=Number(tagV&7n);
    if(wt===2){const {v:lenV,p:p2}=decVarint(buf,pos);pos=p2+Number(lenV);}
    else if(wt===0){
      const {v,p:p2}=decVarint(buf,pos);pos=p2;
      if(fn===2)rec.stake=v;
      if(fn===3)rec.rrs=v;
      if(fn===4)rec.registeredAt=v;
      if(fn===5)rec.successfulResolutions=v;
      if(fn===6)rec.lastClaimedEpoch=v;
    } else if(wt===1){pos+=8;} else if(wt===5){pos+=4;} else break;
  }
  return rec;
}

async function fetchResolverRecord(addrHex){
  try{
    const key=buildResolverKey(addrHex);
    const resp=await fetch(getRPC()+'/v1/query/state',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({key})});
    if(!resp.ok)return null;
    const data=await resp.json();
    const hex=data.value||data.result||'';
    if(!hex)return null;
    return decodeResolverRecord(hex);
  }catch{return null;}
}

// ═══════════════════════════════════════════
// BROWSE RESOLVERS
// ═══════════════════════════════════════════
window.loadResolvers=async function(){
  const el=document.getElementById('resolversList');
  if(!el)return;
  el.innerHTML='<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;loading resolvers</div>';
  try{
    // auto-scan if no cache or registry empty
    if(!localStorage.getItem('praxis_tx_cache')||_resolverRegistry.size===0){
      await loadMarkets();
    }
    const cache=localStorage.getItem('praxis_tx_cache');
    if(!cache){el.innerHTML='<div class="alert ay">No chain data — node may be offline</div>';return;}
    const txs=JSON.parse(cache);
    const resolvers=new Map();
    for(const tx of txs){
      if(tx.messageType!=='register_resolver')continue;
      const msg=(tx.transaction&&tx.transaction.msg)||{};
      const addr=tx.sender||'';
      const stake=BigInt(msg.stakeAmount||msg.stake_amount||0);
      if(!resolvers.has(addr)){
        // use rrs from _resolverRegistry if available (already computed in loadMarkets)
        const regEntry=_resolverRegistry.get(addr);
        const rrs=regEntry?regEntry.rrs:10;
        resolvers.set(addr,{addr,stake,proposals:regEntry?regEntry.proposalCount:0,height:tx.height||0,rrs});
      } else {
        const r=resolvers.get(addr);
        r.stake=stake;
        if((tx.height||0)>r.height)r.height=tx.height;
      }
    }
    for(const tx of txs){
      if(tx.messageType!=='propose_outcome')continue;
      const addr=tx.sender||'';
      if(resolvers.has(addr)&&!_resolverRegistry.has(addr))resolvers.get(addr).proposals++;
    }
    if(resolvers.size===0){el.innerHTML='<div class="alert ay">No registered resolvers found</div>';return;}
    const list=[...resolvers.values()];
    // if registry has data, rrs already set; otherwise estimate
    list.forEach(r=>{ if(!_resolverRegistry.has(r.addr)) r.rrs=Math.min(10+r.proposals*10,999); });
    list.sort((a,b)=>b.rrs-a.rrs);
    el.innerHTML=list.map(r=>{
      const rrs=r.rrs||0;
      let tier,tcolor,ticon;
      if(rrs<10){tier='Suspended';tcolor='var(--red)';ticon='✕';}
      else if(rrs>=500){tier='Gold';tcolor='#FFD700';ticon='★';}
      else if(rrs>=200){tier='Silver';tcolor='#C0C0C0';ticon='◆';}
      else{tier='Bronze';tcolor='#CD7F32';ticon='▲';}
      return '<div class="card" style="margin-bottom:10px"><div class="ci">'+
        '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:10px">'+
          '<div class="addr-mono" style="font-size:11px;color:var(--green)">'+r.addr.slice(0,8)+'\u2026'+r.addr.slice(-6)+'</div>'+
          '<span style="font-family:var(--font-mono);font-size:11px;color:'+tcolor+'">'+ticon+' '+tier+'</span>'+
        '</div>'+
        '<div class="igrid" style="grid-template-columns:1fr 1fr 1fr 1fr">'+
          '<div class="icell"><div class="ilbl">Stake</div><div class="ival" style="font-size:11px">'+fmtPRX(r.stake)+' PRX</div></div>'+
          '<div class="icell"><div class="ilbl">Est. RRS</div><div class="ival" style="color:'+tcolor+'">'+rrs+'</div></div>'+
          '<div class="icell"><div class="ilbl">Proposals</div><div class="ival">'+r.proposals+'</div></div>'+
          '<div class="icell"><div class="ilbl">Since block</div><div class="ival">#'+r.height+'</div></div>'+
        '</div>'+
        '<div style="margin-top:10px;font-family:var(--font-mono);font-size:9px;color:var(--text3);word-break:break-all">'+r.addr+'</div>'+
      '</div></div>';
    }).join('');
  }catch(e){el.innerHTML='<div class="alert ar">Error: '+esc(e.message)+'</div>';}
};
// ═══════════════════════════════════════════
// MARKET DETAIL — ACTIVITY FEED + TOP HOLDERS
// ═══════════════════════════════════════════
window.switchDetailTab = function(tab) {
  ['activity','holders'].forEach(t => {
    const btn = document.getElementById('dtab-'+t);
    const pane = document.getElementById('dpane-'+t);
    if(btn) {
      btn.style.borderBottomColor = t===tab ? 'var(--green)' : 'transparent';
      btn.style.color = t===tab ? 'var(--text2)' : 'var(--text3)';
    }
    if(pane) pane.style.display = t===tab ? '' : 'none';
  });
  if(tab==='activity') renderActivityFeed(_detailMarketId);
  if(tab==='holders')  renderTopHolders(_detailMarketId);
};

window.renderActivityFeed = function(mid) {
  const el = document.getElementById('dpane-activity');
  if(!el||!mid) return;
  try {
    const txs = JSON.parse(localStorage.getItem('praxis_tx_cache')||'[]');
    const relevant = txs.filter(tx => {
      const msg = (tx.transaction&&tx.transaction.msg)||{};
      const rawMid = msg.marketId||msg.market_id||'';
      if(!rawMid && tx.messageType==='create_market') {
        // match by derived marketId
        return false; // handled below
      }
      if(!rawMid) return false;
      let txMid = rawMid;
      try { txMid = b2h(Uint8Array.from(atob(rawMid),c=>c.charCodeAt(0))); } catch{}
      return txMid === mid;
    });

    // also include the create_market TX for this market
    const createTx = txs.find(tx => tx.messageType==='create_market' && _allMarkets.find(m=>m.marketId===mid&&m.txHash===tx.txHash));
    if(createTx && !relevant.includes(createTx)) relevant.unshift(createTx);

    relevant.sort((a,b)=>(b.height||0)-(a.height||0));

    if(!relevant.length){
      el.innerHTML='<div style="padding:20px;text-align:center;font-family:var(--font-mono);font-size:11px;color:var(--text3)">No activity found</div>';
      return;
    }

    const typeIcon  = {create_market:'◎',submit_prediction:'⚡',propose_outcome:'⚖',finalize_market:'✓',cancel_market:'✕',claim_winnings:'◈',forfeit_position:'↩',resolve_market:'⚑'};
    const typeColor = {create_market:'var(--text2)',submit_prediction:'var(--green)',propose_outcome:'#FFD700',finalize_market:'var(--green)',cancel_market:'var(--red)',claim_winnings:'var(--green)',forfeit_position:'var(--red)',resolve_market:'#C0C0C0'};

    el.innerHTML = relevant.map(tx => {
      const msg = (tx.transaction&&tx.transaction.msg)||{};
      const type = tx.messageType||'unknown';
      const icon = typeIcon[type]||'▪';
      const color = typeColor[type]||'var(--text3)';
      const sender = tx.sender||'';
      const height = tx.height||0;
      let detail = '';
      if(type==='submit_prediction'){
        const outcome = msg.outcome===true||msg.outcome==='true'||msg.outcome===1;
        const shares = BigInt(msg.shares||msg.amount||0);
        detail = '<span style="color:'+(outcome?'var(--green)':'var(--red)')+'font-weight:700">'+(outcome?'YES':'NO')+'</span> &nbsp;'+fmtPRX(shares)+' PRX';
      } else if(type==='propose_outcome'){
        const outcome = msg.proposedOutcome===true||msg.proposedOutcome==='true'||msg.proposedOutcome===1;
        detail = 'Proposed <span style="color:'+(outcome?'var(--green)':'var(--red)')+'">'+( outcome?'YES':'NO')+'</span>';
      } else if(type==='create_market'){
        const b0=BigInt(msg.b0||0);
        detail='Market created · B0 '+fmtPRX(b0)+' PRX';
      } else if(type==='finalize_market'){detail='Market finalized';}
      else if(type==='cancel_market'){detail='Market cancelled';}
      else if(type==='claim_winnings'){detail='Claimed winnings';}
      else if(type==='forfeit_position'){detail='Position forfeited';}
      return '<div style="display:flex;align-items:flex-start;gap:12px;padding:12px 16px;border-bottom:1px solid var(--border)">'+
        '<div style="font-size:15px;color:'+color+';min-width:18px;margin-top:1px">'+icon+'</div>'+
        '<div style="flex:1;min-width:0">'+
          '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:3px">'+
            '<span style="font-family:var(--font-mono);font-size:10px;color:'+color+';text-transform:uppercase;letter-spacing:.5px">'+type.replace(/_/g,' ')+'</span>'+
            '<span style="font-family:var(--font-mono);font-size:9px;color:var(--text3)">blk #'+height+'</span>'+
          '</div>'+
          '<div style="font-family:var(--font-mono);font-size:9px;color:var(--text3);margin-bottom:3px">'+
            (sender?sender.slice(0,8)+'…'+sender.slice(-6):'')+
          '</div>'+
          (detail?'<div style="font-family:var(--font-mono);font-size:11px;color:var(--text2)">'+detail+'</div>':'')+
        '</div>'+
      '</div>';
    }).join('');
  } catch(e) {
    el.innerHTML='<div style="padding:16px;color:var(--red);font-family:var(--font-mono);font-size:11px">Error: '+esc(e.message)+'</div>';
  }
};

window.renderTopHolders = function(mid) {
  const el = document.getElementById('dpane-holders');
  if(!el||!mid) return;
  try {
    const txs = JSON.parse(localStorage.getItem('praxis_tx_cache')||'[]');
    const holders = new Map();
    for(const tx of txs){
      if(tx.messageType!=='submit_prediction') continue;
      const msg=(tx.transaction&&tx.transaction.msg)||{};
      const rawMid=msg.marketId||msg.market_id||'';
      let txMid=rawMid;
      try{txMid=b2h(Uint8Array.from(atob(rawMid),c=>c.charCodeAt(0)));}catch{}
      if(txMid!==mid) continue;
      const addr=tx.sender||'';
      const outcome=msg.outcome===true||msg.outcome==='true'||msg.outcome===1;
      const shares=BigInt(msg.shares||msg.amount||0);
      if(!holders.has(addr)) holders.set(addr,{addr,yes:0n,no:0n,txCount:0});
      const h=holders.get(addr);
      if(outcome) h.yes+=shares; else h.no+=shares;
      h.txCount++;
    }
    if(!holders.size){
      el.innerHTML='<div style="padding:20px;text-align:center;font-family:var(--font-mono);font-size:11px;color:var(--text3)">No positions yet</div>';
      return;
    }
    const list=[...holders.values()].sort((a,b)=>Number((b.yes+b.no)-(a.yes+a.no)));
    const totalYes=list.reduce((s,h)=>s+h.yes,0n);
    const totalNo=list.reduce((s,h)=>s+h.no,0n);
    const grand=totalYes+totalNo;
    el.innerHTML=
      '<div style="display:grid;grid-template-columns:1fr 1fr;gap:1px;background:var(--border);border-bottom:1px solid var(--border)">'+
        '<div style="padding:12px;text-align:center;background:var(--surface)">'+
          '<div style="font-family:var(--font-mono);font-size:8px;color:var(--text3);margin-bottom:4px;letter-spacing:1px">TOTAL YES</div>'+
          '<div style="font-family:var(--font-mono);font-size:12px;color:var(--green)">'+fmtPRX(totalYes)+' PRX</div></div>'+
        '<div style="padding:12px;text-align:center;background:var(--surface)">'+
          '<div style="font-family:var(--font-mono);font-size:8px;color:var(--text3);margin-bottom:4px;letter-spacing:1px">TOTAL NO</div>'+
          '<div style="font-family:var(--font-mono);font-size:12px;color:var(--red)">'+fmtPRX(totalNo)+' PRX</div></div>'+
      '</div>'+
      list.map((h,i)=>{
        const total=h.yes+h.no;
        const pct=grand>0n?Number(total*100n/grand):0;
        const yesPct=total>0n?Number(h.yes*100n/total):0;
        const bias=h.yes>h.no?'YES':h.no>h.yes?'NO':'EVEN';
        const bc=bias==='YES'?'var(--green)':bias==='NO'?'var(--red)':'var(--text3)';
        return '<div style="padding:12px 16px;border-bottom:1px solid var(--border)">'+
          '<div style="display:flex;justify-content:space-between;align-items:center;margin-bottom:6px">'+
            '<div style="display:flex;align-items:center;gap:8px">'+
              '<span style="font-family:var(--font-mono);font-size:9px;color:var(--text3);min-width:20px">#'+(i+1)+'</span>'+
              '<span style="font-family:var(--font-mono);font-size:10px;color:var(--green)">'+h.addr.slice(0,8)+'…'+h.addr.slice(-6)+'</span>'+
            '</div>'+
            '<span style="font-family:var(--font-mono);font-size:10px;font-weight:700;color:'+bc+'">'+bias+'</span>'+
          '</div>'+
          '<div style="display:grid;grid-template-columns:1fr 1fr 1fr;gap:6px;margin-bottom:8px">'+
            '<div><div style="font-family:var(--font-mono);font-size:8px;color:var(--text3);margin-bottom:2px">YES</div>'+
              '<div style="font-family:var(--font-mono);font-size:10px;color:var(--green)">'+fmtPRX(h.yes)+'</div></div>'+
            '<div><div style="font-family:var(--font-mono);font-size:8px;color:var(--text3);margin-bottom:2px">NO</div>'+
              '<div style="font-family:var(--font-mono);font-size:10px;color:var(--red)">'+fmtPRX(h.no)+'</div></div>'+
            '<div><div style="font-family:var(--font-mono);font-size:8px;color:var(--text3);margin-bottom:2px">SHARE</div>'+
              '<div style="font-family:var(--font-mono);font-size:10px;color:var(--text2)">'+pct+'%</div></div>'+
          '</div>'+
          '<div style="height:3px;background:var(--border);border-radius:2px;overflow:hidden">'+
            '<div style="height:100%;width:'+yesPct+'%;background:var(--green);transition:width .3s"></div>'+
          '</div>'+
        '</div>';
      }).join('');
  } catch(e){
    el.innerHTML='<div style="padding:16px;color:var(--red);font-family:var(--font-mono);font-size:11px">Error: '+esc(e.message)+'</div>';
  }
};
