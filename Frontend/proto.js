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
async function blsSign(msg){
  if(!signerPrivKey)throw new Error('No key loaded — go to Signer');
  if(!bls12_381)throw new Error('BLS library not loaded');
  return await bls12_381.sign(msg,signerPrivKey);
}

function b2b64(bytes){
  let s='';for(let i=0;i<bytes.length;i++)s+=String.fromCharCode(bytes[i]);
  return btoa(s);
}

function decVarint(buf,pos){let r=0n,s=0n;while(pos<buf.length){const b=BigInt(buf[pos++]);r|=(b&0x7fn)<<s;s+=7n;if(!(b&0x80n))break;}return{v:r,p:pos};}
function encClaimCreatorFee(mid,creator){return cat(bf(1,h2b(mid)),bf(2,h2b(creator)));}
function encCancelMarket(mid,creator){return cat(bf(1,h2b(mid)),bf(2,h2b(creator)));}
