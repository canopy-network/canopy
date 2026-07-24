function resolverTier(addr) {
  const r = _resolverRegistry.get(addr);
  if (!r) return null;
  const estRRS = Math.min(r.proposalCount * 10, 999);
  if (estRRS >= 200) return {label:'Gold',   color:'#FFD700', icon:'★'};
  if (estRRS >= 50)  return {label:'Silver', color:'#C0C0C0', icon:'◆'};
  if (estRRS >= 1)   return {label:'Bronze', color:'#CD7F32', icon:'▲'};
  return {label:'Registered', color:'var(--text3)', icon:'○'};
}

let _resolverRegistry = new Map();

async function checkRoles() {
  // DEVNET: show all nav sections/items to everyone regardless of role
  document.getElementById('nav-admin-section').style.display = '';
  document.querySelectorAll('.nav-admin-item').forEach(el => el.style.display = '');
  document.getElementById('nav-resolver-section').style.display = '';
  document.querySelectorAll('.nav-resolver-item').forEach(el => el.style.display = '');
}

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

window.loadResolvers=async function(){
  const el=document.getElementById('resolversList');
  if(!el)return;
  el.innerHTML='<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;loading resolvers</div>';
  try{
    const resp = await fetch(getPluginRPC() + '/v1/query/resolvers');
    if(!resp.ok) throw new Error('resolvers query returned ' + resp.status);
    const raw = await resp.json();
    if(!raw || raw.length===0){el.innerHTML='<div class="alert ay">No registered resolvers found</div>';return;}
    const list = raw.map(r => ({
      addr: r.resolver_address ? b2h(Uint8Array.from(atob(r.resolver_address),c=>c.charCodeAt(0))) : '',
      stake: BigInt(r.stake_amount||0),
      proposals: Number(r.successful_resolutions||0),
      height: Number(r.registered_at||0),
      rrs: Number(r.rrs_score||0),
    }));
    list.sort((a,b)=>b.rrs-a.rrs);
    window._resolvers = list;
    el.innerHTML=list.map(r=>{
      const rrs=r.rrs||0;
      let tier,tcolor,ticon;
      if(rrs<10){tier='Suspended';tcolor='var(--red)';ticon='✕';}
      else if(rrs>=100){tier='Gold';tcolor='#FFD700';ticon='★';}
      else if(rrs>=50){tier='Silver';tcolor='#C0C0C0';ticon='◆';}
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

window.renderMyResolverStatus = async function(which) {
  // which: 'unstake' or 'claim-unbonded'
  const cardId = which === 'unstake' ? 'un_status_card' : 'ub_status_card';
  const formId = which === 'unstake' ? 'un_form_card' : 'ub_form_card';
  const cardEl = document.getElementById(cardId);
  const formEl = document.getElementById(formId);
  if (!cardEl || !formEl) return;

  if (!signerAddress) {
    cardEl.innerHTML = '<div class="alert ay">Connect a signer to see your resolver status.</div>';
    formEl.style.display = 'none';
    return;
  }

  cardEl.innerHTML = '<div class="loading"><span class="blink">▪ ▪ ▪</span>&nbsp;&nbsp;loading resolver status</div>';

  try {
    const resp = await fetch(getPluginRPC() + '/v1/query/resolvers');
    if (!resp.ok) throw new Error('resolvers query returned ' + resp.status);
    const raw = await resp.json();
    const mine = (raw || []).find(r => {
      const addr = r.resolver_address ? b2h(Uint8Array.from(atob(r.resolver_address), c => c.charCodeAt(0))) : '';
      return addr.toLowerCase() === signerAddress.toLowerCase();
    });

    if (!mine) {
      cardEl.innerHTML = '<div class="alert ay">No resolver record found for this address — not currently staked.</div>';
      formEl.style.display = 'none';
      return;
    }

    const stake = BigInt(mine.stake_amount || 0);
    const unbondingAmt = BigInt(mine.unbonding_amount || 0);
    const releaseHeight = Number(mine.unbonding_release_height || 0);
    const curHeight = Number(window.currentHeight || 0);

    if (unbondingAmt === 0n) {
      // Staked, not unbonding
      cardEl.innerHTML = '<div class="card" style="margin-bottom:16px"><div class="ci">' +
        '<div class="ct">// your_resolver_status</div>' +
        '<div class="igrid" style="grid-template-columns:1fr 1fr">' +
        '<div class="icell"><div class="ilbl">Staked</div><div class="ival" style="color:var(--green)">' + fmtPRX(stake) + ' PRX</div></div>' +
        '<div class="icell"><div class="ilbl">Status</div><div class="ival">Active</div></div>' +
        '</div></div></div>';
      formEl.style.display = '';
    } else if (curHeight > 0 && curHeight < releaseHeight) {
      // Unbonding — countdown
      const blocksLeft = releaseHeight - curHeight;
      const msLeft = blocksLeft * BLOCK_TIME_MS;
      cardEl.innerHTML = '<div class="card" style="margin-bottom:16px"><div class="ci">' +
        '<div class="ct">// your_resolver_status</div>' +
        '<div class="igrid" style="grid-template-columns:1fr 1fr">' +
        '<div class="icell"><div class="ilbl">Unbonding</div><div class="ival" style="color:var(--amber)">' + fmtPRX(unbondingAmt) + ' PRX</div></div>' +
        '<div class="icell"><div class="ilbl">Unlocks in</div><div class="ival">' + fmtDuration(msLeft) + '</div></div>' +
        '</div>' +
        '<div style="margin-top:10px;font-family:var(--mono);font-size:10px;color:var(--text3)">Releases at block #' + releaseHeight + '</div>' +
        '</div></div>';
      formEl.style.display = which === 'unstake' ? 'none' : 'none';
      if (which === 'claim-unbonded') formEl.style.display = 'none';
    } else {
      // Unbonding complete — ready to claim
      cardEl.innerHTML = '<div class="card" style="margin-bottom:16px;border-color:rgba(0,232,122,.3)"><div class="ci">' +
        '<div class="ct">// your_resolver_status</div>' +
        '<div class="igrid" style="grid-template-columns:1fr 1fr">' +
        '<div class="icell"><div class="ilbl">Ready to claim</div><div class="ival" style="color:var(--green)">' + fmtPRX(unbondingAmt) + ' PRX</div></div>' +
        '<div class="icell"><div class="ilbl">Status</div><div class="ival" style="color:var(--green)">Unlocked</div></div>' +
        '</div>' +
        (which === 'unstake' ? '<div style="margin-top:10px"><a href="#" onclick="showPage(\'claim-unbonded\',null);return false" class="btn bp" style="display:inline-block;text-decoration:none">Go to Claim Unbonded →</a></div>' : '') +
        '</div></div>';
      formEl.style.display = which === 'claim-unbonded' ? '' : 'none';
    }
  } catch (e) {
    cardEl.innerHTML = '<div class="alert ar">Error loading resolver status: ' + esc(e.message) + '</div>';
    formEl.style.display = '';
  }
};

function encRewardResolver(addr, epoch){ return cat(bf(1,h2b(addr)), vf(2, BigInt(epoch))); }

function encRewardBuilder(addr){         return cat(bf(1,h2b(addr))); }

function encRewardCommunity(addr){       return cat(bf(1,h2b(addr))); }

function encRewardInvestor(addr){        return cat(bf(1,h2b(addr))); }

function encRewardProtocol(addr){        return cat(bf(1,h2b(addr))); }

function checkRewardAuth(pageId, contentId, unauthId, authorizedAddr) {
  const authed = !authorizedAddr || (signerAddress && signerAddress.toLowerCase() === authorizedAddr.toLowerCase());
  document.getElementById(contentId).style.display = authed ? '' : 'none';
  document.getElementById(unauthId).style.display  = authed ? 'none' : '';
}

async function loadPoolStat(elId, key) {
  try {
    const el = document.getElementById(elId);
    if (!el) return;
    // Pool data comes from plugin state — show epoch estimate for now
    const epoch = window.currentHeight ? Math.floor(window.currentHeight / EPOCH_BLOCKS) : 0;
    el.textContent = 'Epoch #' + epoch;
  } catch(e) {}
}

async function loadResolverRewardData() {
  try {
    const epoch = window.currentHeight ? Math.floor(window.currentHeight / EPOCH_BLOCKS) : 0;
    document.getElementById('rrw-pool').textContent = 'Epoch #' + epoch;

    // Pull resolver info from the resolvers map if already loaded
    if (signerAddress && window._resolvers) {
      const r = window._resolvers.get(signerAddress.toLowerCase());
      if (r) {
        const rrs = r.rrs || 10;
        const proposals = r.proposalCount || 0;
        document.getElementById('rrw-rrs').textContent  = rrs;
        document.getElementById('rrw-rrs2').textContent = rrs;
        document.getElementById('rrw-resolutions').textContent = proposals;

        // Tier
        let tier = 'bronze', tierLabel = '🥉 Bronze', tierClass = 'rrs-bronze';
        if (rrs >= 100) { tier = 'gold';   tierLabel = 'Gold';   tierClass = 'rrs-gold'; }
        else if (rrs >= 50) { tier = 'silver'; tierLabel = 'Silver'; tierClass = 'rrs-silver'; }
        const weight = rrs >= 100 ? 3 : rrs >= 50 ? 2 : 1;

        const badge = document.getElementById('rrw-tier-badge');
        badge.className = 'rrs-badge ' + tierClass;
        badge.innerHTML = tierLabel + ' — RRS <span id="rrw-rrs">' + rrs + '</span>';
        document.getElementById('rrw-share').textContent = weight + '× weight';

        // Epoch history table (last 5 epochs)
        let rows = '';
        for (let i = Math.max(0, epoch - 4); i <= epoch; i++) {
          const isCurrent = i === epoch;
          rows += `<tr>
            <td>#${i}</td>
            <td class="d">${isCurrent ? 'In progress' : '—'}</td>
            <td class="d">—</td>
            <td class="${isCurrent ? 'g' : 'd'}">${isCurrent ? 'Current' : 'Claimable'}</td>
          </tr>`;
        }
        document.querySelector('#rrw-history tbody').innerHTML = rows;
      }
    }
  } catch(e) {}
}

async function loadBuilderRewardData() {
  try {
    const epoch = window.currentHeight ? Math.floor(window.currentHeight / EPOCH_BLOCKS) : 0;
    document.getElementById('brw-pool').textContent = 'Epoch #' + epoch;
    let rows = '';
    for (let i = Math.max(0, epoch - 4); i <= epoch; i++) {
      const isCurrent = i === epoch;
      rows += `<tr>
        <td>#${i}</td>
        <td class="d">${isCurrent ? 'In progress' : '—'}</td>
        <td class="d">—</td>
        <td class="${isCurrent ? 'g' : 'd'}">${isCurrent ? 'Current' : 'Claimable'}</td>
      </tr>`;
    }
    document.querySelector('#brw-history tbody').innerHTML = rows;
  } catch(e) {}
}

window.signAndSubmit_rewardResolver = async function() {
  try {
    const addr = document.getElementById('rrw-addr').value.trim();
    const epochVal = document.getElementById('rrw-epoch').value.trim();
    const epoch = epochVal ? parseInt(epochVal) : Math.floor((window.currentHeight||0) / EPOCH_BLOCKS);
    const fee = BigInt(document.getElementById('rrw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid resolver address', true);
    await doSubmit('claim_resolver_reward','type.googleapis.com/types.MessageClaimResolverReward',encRewardResolver(addr,epoch),{fee},'btn_rrw','pend_rrw');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.build_rewardResolver = function() {
  try {
    const addr = document.getElementById('rrw-addr').value.trim();
    const epochVal = document.getElementById('rrw-epoch').value.trim();
    const epoch = epochVal ? parseInt(epochVal) : Math.floor((window.currentHeight||0) / EPOCH_BLOCKS);
    const fee = BigInt(document.getElementById('rrw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid resolver address', true);
    showPL('rrwo','rrwp',buildUnsigned('claim_resolver_reward','type.googleapis.com/types.MessageClaimResolverReward',encRewardResolver(addr,epoch),{fee}));
    toast('Payload built');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.signAndSubmit_rewardBuilder = async function() {
  try {
    const addr = document.getElementById('brw-addr').value.trim();
    const fee = BigInt(document.getElementById('brw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    await doSubmit('claim_builder_reward','type.googleapis.com/types.MessageClaimBuilderReward',encRewardBuilder(addr),{fee},'btn_brw','pend_brw');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.build_rewardBuilder = function() {
  try {
    const addr = document.getElementById('brw-addr').value.trim();
    const fee = BigInt(document.getElementById('brw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    showPL('brwo','brwp',buildUnsigned('claim_builder_reward','type.googleapis.com/types.MessageClaimBuilderReward',encRewardBuilder(addr),{fee}));
    toast('Payload built');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.signAndSubmit_rewardCommunity = async function() {
  try {
    const addr = document.getElementById('crw-addr').value.trim();
    const fee = BigInt(document.getElementById('crw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    await doSubmit('claim_community_reward','type.googleapis.com/types.MessageClaimCommunityReward',encRewardCommunity(addr),{fee},'btn_crw','pend_crw');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.build_rewardCommunity = function() {
  try {
    const addr = document.getElementById('crw-addr').value.trim();
    const fee = BigInt(document.getElementById('crw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    showPL('crwo','crwp',buildUnsigned('claim_community_reward','type.googleapis.com/types.MessageClaimCommunityReward',encRewardCommunity(addr),{fee}));
    toast('Payload built');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.signAndSubmit_rewardInvestor = async function() {
  try {
    const addr = document.getElementById('irw-addr').value.trim();
    const fee = BigInt(document.getElementById('irw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    await doSubmit('claim_investor_reward','type.googleapis.com/types.MessageClaimInvestorReward',encRewardInvestor(addr),{fee},'btn_irw','pend_irw');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.build_rewardInvestor = function() {
  try {
    const addr = document.getElementById('irw-addr').value.trim();
    const fee = BigInt(document.getElementById('irw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    showPL('irwo','irwp',buildUnsigned('claim_investor_reward','type.googleapis.com/types.MessageClaimInvestorReward',encRewardInvestor(addr),{fee}));
    toast('Payload built');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.signAndSubmit_rewardProtocol = async function() {
  try {
    const addr = document.getElementById('prw-addr').value.trim();
    const fee = BigInt(document.getElementById('prw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    await doSubmit('claim_protocol_reward','type.googleapis.com/types.MessageClaimProtocolReward',encRewardProtocol(addr),{fee},'btn_prw','pend_prw');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};

window.build_rewardProtocol = function() {
  try {
    const addr = document.getElementById('prw-addr').value.trim();
    const fee = BigInt(document.getElementById('prw-fee').value||10000);
    if (!addr || addr.length !== 40) return toast('Invalid address', true);
    showPL('prwo','prwp',buildUnsigned('claim_protocol_reward','type.googleapis.com/types.MessageClaimProtocolReward',encRewardProtocol(addr),{fee}));
    toast('Payload built');
  } catch(e) { toast(friendlyError(null,e.message),true); }
};
