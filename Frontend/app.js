
// ═══════════════════════════════════════════
// BLS
// ═══════════════════════════════════════════
let bls12_381 = null;

// ═══════════════════════════════════════════
// CONFIG & STATE
// ═══════════════════════════════════════════

let selectedOut   = true;
let propOut       = true;
let revOut        = true;

// ═══════════════════════════════════════════
// PROTO ENCODER
// ═══════════════════════════════════════════

// ═══════════════════════════════════════════
// BLS SIGN
// ═══════════════════════════════════════════
// ═══════════════════════════════════════════
// BASE64 HELPER (for proto JSON encoding)
// ═══════════════════════════════════════════
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

// ═══════════════════════════════════════════
// NAVIGATION
// ═══════════════════════════════════════════

// ═════════════════════════════════════════════
// MOBILE NAV
// ═══════════════════════════════════════════

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


// ═══════════════════════════════════════════
// RENDER MARKET CARDS — Premium Design
// ═══════════════════════════════════════════

// ── Volume chip updater ──

// store markets globally for detail view


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





// ═══════════════════════════════════════════
// ── SEND
// ═══════════════════════════════════════════

// ── CREATE MARKET

// ── SUBMIT PREDICTION

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
