
// ═══════════════════════════════════════════
// BLS
// ═══════════════════════════════════════════
let bls12_381 = null;
window.blsReady = (async () => {
  for (const url of ['https://esm.sh/@noble/curves@1.4.2/bls12-381','https://cdn.skypack.dev/@noble/curves@1.4.2/bls12-381']) {
    try { const m = await import(url); bls12_381 = m.bls12_381; break; } catch {}
  }
  if (!bls12_381) toast('BLS library failed to load — check internet', true);
  return bls12_381;
})();

// ═══════════════════════════════════════════
// CONFIG & STATE
// ═══════════════════════════════════════════

window.currentHeight = 0;
window.currentNetworkID = 1;
window.currentChainID   = 266;
let selectedOut   = true;
let propOut       = true;
let revOut        = true;

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

function getSelectedCat() {
  const el = document.querySelector('#c_cat_pick .cpick.active');
  return el ? el.getAttribute('data-cat') : 'other';
}

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
    vf(4,height||window.currentHeight),vf(5,txTime),vf(6,fee||10000),
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
  const p={txTime,fee:meta.fee||10000,height:meta.height||window.currentHeight,memo:'',netId:window.currentNetworkID,chainId:window.currentChainID};
  const sb=encSignBytes(msgType,typeUrl,inner,p);
  const sig=await blsSign(sb);
  const base={
    signature: { publicKey: b2h(signerPubKey), signature: b2h(sig) },
    createdHeight: p.height,
    time: Number(txTime),
    fee: p.fee,
    memo: '',
    networkID: window.currentNetworkID,
    chainID: window.currentChainID,
  };
  if(msgType==='send'){
    const bytes=inner instanceof Uint8Array?inner:h2b(b2h(inner));
    let pos=0,fromB=null,toB=null,amt=0n;
    while(pos<bytes.length){
      const {v:tagV,p:p1}=decVarint(bytes,pos);pos=p1;
      const fn=Number(tagV>>3n),wt=Number(tagV&7n);
      if(wt===2){const {v:ln,p:p2}=decVarint(bytes,pos);pos=p2;const val=bytes.slice(pos,pos+Number(ln));pos+=Number(ln);if(fn===1)fromB=val;else if(fn===2)toB=val;}
      else if(wt===0){const {v,p:p2}=decVarint(bytes,pos);pos=p2;if(fn===3)amt=v;}
    }
    const toHex=b=>Array.from(b).map(x=>x.toString(16).padStart(2,'0')).join('');
    return {...base,type:'send',msg:{fromAddress:toHex(fromB),toAddress:toHex(toB),amount:Number(amt)}};
  }
  return {...base,type:msgType,msgTypeUrl:typeUrl,msgBytes:b2h(inner)};
}

function buildUnsigned(msgType,typeUrl,inner,meta){
  const txTime=BigInt(Date.now())*1000n;
  return {
    message_type: msgType,
    msg: { type_url: typeUrl, value: b2b64(inner) },
    signature: null,
    created_height: meta.height||window.currentHeight,
    time: Number(txTime),
    fee: meta.fee||10000,
    memo: '',
    network_id: window.currentNetworkID||1,
    chain_id: window.currentChainID||1,
  };
}

// ═══════════════════════════════════════════
// RPC
// ═══════════════════════════════════════════

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

// ═════════════════════════════════════════════
// MOBILE NAV
// ═══════════════════════════════════════════
window.openNav=function(){document.getElementById('deskNav').classList.add('open');document.getElementById('mobNav').classList.add('open');};window.closeNav=function(e){if(!e||e.target===document.getElementById('mobNav')||e.currentTarget===document.getElementById('mobNav')){document.getElementById('deskNav').classList.remove('open');document.getElementById('mobNav').classList.remove('open');}};

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

// ═══════════════════════════════════════════
// OUTCOME TOGGLES
// ═══════════════════════════════════════════
window.setOut=function(v){selectedOut=v;document.getElementById('btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('btn_no').className='obtn no'+(!v?' active':'');};
window.setPropOut=function(v){propOut=v;document.getElementById('pr_btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('pr_btn_no').className='obtn no'+(!v?' active':'');};
window.setRevOut=function(v){revOut=v;document.getElementById('rv_btn_yes').className='obtn yes'+(v?' active':'');document.getElementById('rv_btn_no').className='obtn no'+(!v?' active':'');};

// ═══════════════════════════════════════════
// SIGNER
// ═══════════════════════════════════════════

// ═════════════════════════════════════════════
// ACCOUNT QUERY
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// FAILED TX
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// PENDING HELPER
// ═══════════════════════════════════════════
function setPend(btnId,pendId,on){
  const b=document.getElementById(btnId);const p=document.getElementById(pendId);
  if(b)b.disabled=on;if(p)p.style.display=on?'flex':'none';
}

async function doSubmit(msgType,typeUrl,inner,meta,btnId,pendId){
  if(!signerPrivKey)return toast('Load a private key in Signer first',true);
  if(!window.currentHeight)return toast('Node not connected',true);
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

// ═══════════════════════════════════════════
// RENDER MARKET CARDS — Premium Design
// ═══════════════════════════════════════════

// ── Volume chip updater ──

// store markets globally for detail view
window._allMarkets = [];

window.openDetail = window.showDetail;

// ═════════════════════════════════════════════
// CLIENT-SIDE ROUTING
// ═══════════════════════════════════════════
const _VALID_PAGE_IDS = ['cancel', 'claim', 'claimcreator', 'commit', 'create', 'detail', 'dispute', 'finalize', 'forfeit', 'markets', 'node', 'profile', 'propose', 'reclaim', 'register', 'resolvers', 'reveal', 'search', 'slash', 'tally'];

function _routeFromPath(path) {
  if (path === '/' || path === '') return { id: 'markets' };
  const detailMatch = path.match(/^\/detail\/(.+)$/);
  if (detailMatch) return { id: 'detail', mid: decodeURIComponent(detailMatch[1]) };
  const id = path.replace(/^\//, '');
  if (_VALID_PAGE_IDS.includes(id)) return { id };
  return { id: 'markets' };
}

window.addEventListener('popstate', function(e) {
  const route = _routeFromPath(location.pathname);
  if (route.id === 'detail' && route.mid) {
    showDetail(route.mid);
  } else {
    showPage(route.id, null, true);
  }
});

window.addEventListener('DOMContentLoaded', async function() {
  const route = _routeFromPath(location.pathname);
  if (route.id === 'detail' && route.mid) {
    await loadMarkets();
    showDetail(route.mid);
  } else if (route.id !== 'markets') {
    showPage(route.id, null, true);
  }
  // markets is already the default-visible page div in the HTML, and
  // loadMarkets() already runs on its own startup path elsewhere, so no
  // extra action needed when route.id === 'markets'.
});



// ═══════════════════════════════════════════
// MARKETS PAGE
// ═══════════════════════════════════════════
function decVarint(buf,pos){let r=0n,s=0n;while(pos<buf.length){const b=BigInt(buf[pos++]);r|=(b&0x7fn)<<s;s+=7n;if(!(b&0x80n))break;}return{v:r,p:pos};}

window._activeTab = 'live';

window.switchTab = function(tab) {
  window._activeTab = tab;
  window._activeTab = tab;
  document.querySelectorAll('.mtab').forEach(b => b.classList.remove('active'));
  const btn = document.getElementById('tab-' + tab);
  if (btn) btn.classList.add('active');
  renderCurrentTab();
};



// ═══════════════════════════════════════════
// ── SEND
// ═══════════════════════════════════════════

// ── CREATE MARKET
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

// ── SUBMIT PREDICTION
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

// ── CLAIM WINNINGS

// ── REGISTER RESOLVER

// ── PROPOSE OUTCOME

// ── FILE DISPUTE

// ── COMMIT VOTE

// ── REVEAL VOTE

// ── TALLY VOTES

// ── FINALIZE MARKET

// ── CLAIM SLASH

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
        ['Market ID',   (document.getElementById('pr_mid')?.value||'').slice(0,16)+'…', ''],
        ['Outcome',     (window._propOut!==false?'YES':'NO'), window._propOut!==false?'green':'red'],
        ['Bond',        v('prop_bond').toLocaleString()+' PRX', ''],
      ]
    ],
    signAndSubmit_dispute: () => [
      'File Dispute', [
        ['Market ID',   (document.getElementById('di_mid')?.value||'').slice(0,16)+'…', ''],
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
        ['Market ID',   (document.getElementById('ta_mid')?.value||'').slice(0,16)+'…', ''],
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
        ['Market ID', (document.getElementById('cf_mid')?.value||'').slice(0,16)+'…', ''],
        ['Creator',   (document.getElementById('cf_addr')?.value||'').slice(0,16)+'…', ''],
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
        ['Resolver', (document.getElementById('un_addr')?.value||'').slice(0,16)+'…', ''],
        ['Amount',   (parseInt(document.getElementById('un_amount')?.value||0)).toLocaleString()+' PRX (0 = full exit)', ''],
      ]
    ],
    signAndSubmit_claim_unbonded: () => [
      'Claim Unbonded Stake', [
        ['Resolver', (document.getElementById('ub_addr')?.value||'').slice(0,16)+'…', ''],
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

// Init copy btn injection
injectKeyboardCopyBtns();

// ═══════════════════════════════════════════
// INIT
// ═══════════════════════════════════════════
const _niHost=document.getElementById('rpc_url');if(_niHost)_niHost.value=getRPCHost();
buildMobNav();
checkRPC();
setTimeout(loadMarkets, 0);
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






// ═══════════════════════════════════════════════
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



// ═══════════════════════════════════════════
// ROLE-BASED SIDEBAR
// ═══════════════════════════════════════════

// Run role check after key load and after markets load
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
function encClaimCreatorFee(mid,creator){return cat(bf(1,h2b(mid)),bf(2,h2b(creator)));}


// ═══════════════════════════════════════════
// CANCEL MARKET
// ═══════════════════════════════════════════
function encCancelMarket(mid,creator){return cat(bf(1,h2b(mid)),bf(2,h2b(creator)));}


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
let _srchCat = 'all';

window.srchCat = function(el) {
  document.querySelectorAll('#srch-cats .cpick').forEach(e => e.classList.remove('active'));
  el.classList.add('active');
  _srchCat = el.getAttribute('data-cat') || 'all';
  runSearch();
};

window.runSearch = function() {
  const q = (document.getElementById('srch-input')?.value || '').trim().toLowerCase();
  const out = document.getElementById('srch-results');
  if (!out) return;
  const markets = window._allMarkets || [];
  if (!q && _srchCat === 'all') {
    out.innerHTML = '<div style="color:var(--text3);font-family:var(--mono);font-size:11px;text-align:center;padding:40px 0">Type to search markets</div>';
    return;
  }
  let filtered = markets.filter(m => {
    const catMatch = _srchCat === 'all' || extractCat(m.rules || '') === _srchCat;
    const textMatch = !q ||
      (m.question || '').toLowerCase().includes(q) ||
      (m.marketId || '').toLowerCase().includes(q) ||
      (m.creator || '').toLowerCase().includes(q) ||
      stripCatPrefix(m.rules || '').toLowerCase().includes(q);
    return catMatch && textMatch;
  });
  if (filtered.length === 0) {
    out.innerHTML = '<div style="color:var(--text3);font-family:var(--mono);font-size:11px;text-align:center;padding:40px 0">No markets found</div>';
    return;
  }
  let bookmarks = [];
  try { bookmarks = JSON.parse(localStorage.getItem('praxis_bookmarks') || '[]'); } catch {}
  out.innerHTML = '<div class="mgrid-2col">' + filtered.map(m => buildPraxisCard(m, bookmarks, false)).join('') + '</div>';
};

// ═══════════════════════════════════════════
// NEW DETAIL PAGE FUNCTIONS
// ═══════════════════════════════════════════

window.setHolderTab = function(side, el) {
  document.querySelectorAll('.det-htab').forEach(b => b.classList.remove('active'));
  el?.classList.add('active');
  renderHoldersSidebar(window._detailMarketId, side);
};

window.setChartRange = function(range, el) {
  document.querySelectorAll('.det-ctab').forEach(b => b.classList.remove('active'));
  el?.classList.add('active');
  renderDetailChart(window._detailMarketId, range);
};

window.renderDetailChart = function(mid, range) {
  const canvas  = document.getElementById('det-chart');
  const emptyEl = document.getElementById('det-chart-empty');
  if (!canvas) return;
  try {
    const txs = JSON.parse(localStorage.getItem('praxis_tx_cache') || '[]');
    const sorted = txs.filter(tx => {
      if (tx.messageType !== 'submit_prediction') return false;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || msg.market_id || '';
      let txMid = rawMid;
      try { txMid = b2h(Uint8Array.from(atob(rawMid), c=>c.charCodeAt(0))); } catch {}
      return txMid === mid;
    }).sort((a,b) => (a.height||0) - (b.height||0));

    if (sorted.length < 2) {
      if (emptyEl) emptyEl.style.display = '';
      canvas.style.opacity = '0';
      return;
    }

    let qYes = 0, qNo = 0;
    const points = [];
    for (const tx of sorted) {
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const outcome = msg.outcome === true || msg.outcome === 'true' || msg.outcome === 1;
      const shares = Number(BigInt(msg.shares || msg.amount || 0)) / 1e6;
      if (outcome) qYes += shares; else qNo += shares;
      const total = qYes + qNo;
      if (total > 0) points.push({ h: tx.height || 0, pct: qYes / total * 100 });
    }

    if (points.length < 2) { if (emptyEl) emptyEl.style.display = ''; canvas.style.opacity = '0'; return; }
    if (emptyEl) emptyEl.style.display = 'none';
    canvas.style.opacity = '1';

    const ctx = canvas.getContext('2d');
    const W = canvas.offsetWidth || 560, H = canvas.offsetHeight || 140;
    canvas.width = W; canvas.height = H;
    ctx.clearRect(0, 0, W, H);
    const pad = {t:10,b:10,l:10,r:10};
    const iW = W-pad.l-pad.r, iH = H-pad.t-pad.b;
    const minH = points[0].h, maxH = points[points.length-1].h;
    const rangeH = Math.max(maxH-minH, 1);

    ctx.strokeStyle = 'rgba(255,255,255,.04)'; ctx.lineWidth = 1;
    [25,50,75].forEach(p => {
      const y = pad.t + iH - (p/100)*iH;
      ctx.beginPath(); ctx.moveTo(pad.l,y); ctx.lineTo(pad.l+iW,y); ctx.stroke();
    });
    ctx.strokeStyle = 'rgba(255,255,255,.12)'; ctx.setLineDash([4,4]);
    const y50 = pad.t + iH*0.5;
    ctx.beginPath(); ctx.moveTo(pad.l,y50); ctx.lineTo(pad.l+iW,y50); ctx.stroke();
    ctx.setLineDash([]);

    ctx.beginPath();
    points.forEach((pt,i) => {
      const x = pad.l + ((pt.h-minH)/rangeH)*iW;
      const y = pad.t + iH - (pt.pct/100)*iH;
      i===0 ? ctx.moveTo(x,y) : ctx.lineTo(x,y);
    });
    ctx.strokeStyle = '#00e87a'; ctx.lineWidth = 2; ctx.stroke();

    const lastPt = points[points.length-1];
    const lastX = pad.l + ((lastPt.h-minH)/rangeH)*iW;
    ctx.lineTo(lastX, pad.t+iH); ctx.lineTo(pad.l, pad.t+iH); ctx.closePath();
    const grad = ctx.createLinearGradient(0,pad.t,0,pad.t+iH);
    grad.addColorStop(0,'rgba(0,232,122,.25)'); grad.addColorStop(1,'rgba(0,232,122,0)');
    ctx.fillStyle = grad; ctx.fill();

    const lastY = pad.t + iH - (lastPt.pct/100)*iH;
    ctx.beginPath(); ctx.arc(lastX,lastY,5,0,Math.PI*2);
    ctx.fillStyle = '#00e87a'; ctx.fill();
    ctx.strokeStyle = '#fff'; ctx.lineWidth = 2; ctx.stroke();
  } catch(e) { if (emptyEl) emptyEl.style.display = ''; }
};

window.renderDisputeCountdown = async function(mid) {
  const row = document.getElementById('det-dispute-row');
  const el  = document.getElementById('det-dispute-val');
  if (!row || !el || !mid) return;
  row.style.display = 'none';
  try {
    const resp = await fetch(getPluginRPC() + '/v1/query/dispute-context?market=' + encodeURIComponent(mid));
    if (!resp.ok) throw new Error('dispute-context query returned ' + resp.status);
    const d = await resp.json();
    const hasProposal = !!d.proposal;
    if (!hasProposal) return;

    const dw = d.dispute_window || {};
    if (dw.open) {
      const blocksLeft = Math.max(0, dw.deadline_block - dw.current_height);
      const msLeft = blocksLeft * 5000;
      const deadline = new Date(Date.now() + msLeft);
      const dateStr = deadline.toLocaleDateString('en-US', {month:'short', day:'numeric', year:'numeric'});
      const timeStr = deadline.toLocaleTimeString('en-US', {hour:'2-digit', minute:'2-digit'});
      const hoursLeft = (msLeft / 3600000).toFixed(1);
      el.innerHTML = '<span style="color:var(--green)">' + hoursLeft + 'h left</span> — closes ' + dateStr + ' ' + timeStr + ' (blk #' + dw.deadline_block + ')';
    } else {
      const reason = d.should_dispute_reason || 'window closed';
      el.innerHTML = '<span style="color:var(--text3)">Closed</span> — ' + reason;
    }
    row.style.display = '';
  } catch(e) {
    console.warn('dispute countdown failed', e);
  }
};

window.renderHoldersSidebar = function(mid, side) {
  side = side || 'yes';
  const el = document.getElementById('det-holders-list');
  if (!el) return;
  try {
    const txs = JSON.parse(localStorage.getItem('praxis_tx_cache') || '[]');
    const holders = {};
    txs.filter(tx => {
      if (tx.messageType !== 'submit_prediction') return false;
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const rawMid = msg.marketId || msg.market_id || '';
      let txMid = rawMid;
      try { txMid = b2h(Uint8Array.from(atob(rawMid), c=>c.charCodeAt(0))); } catch {}
      return txMid === mid;
    }).forEach(tx => {
      const msg = (tx.transaction && tx.transaction.msg) || {};
      const outcome = msg.outcome === true || msg.outcome === 'true' || msg.outcome === 1;
      if ((side==='yes') !== outcome) return;
      const addr = tx.sender || '?';
      const shares = Number(BigInt(msg.shares || msg.amount || 0)) / 1e6;
      holders[addr] = (holders[addr] || 0) + shares;
    });
    const sorted = Object.entries(holders).sort((a,b) => b[1]-a[1]).slice(0,10);
    if (!sorted.length) { el.innerHTML = '<div class="det-holders-empty">No holders yet</div>'; return; }
    el.innerHTML = sorted.map(([addr,amt],i) => `
      <div class="det-holder-row">
        <span class="det-holder-rank">#${i+1}</span>
        <div class="det-holder-avatar">${addr.slice(0,2).toUpperCase()}</div>
        <span class="det-holder-addr">${addr.slice(0,8)}…${addr.slice(-4)}</span>
        <span class="det-holder-amt ${side}-side">${amt.toFixed(2)} PRX</span>
      </div>`).join('');
  } catch(e) { el.innerHTML = '<div class="det-holders-empty">No holders yet</div>'; }
};

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
