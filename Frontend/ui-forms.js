window.fillP = (id, outcome) => {
  document.getElementById('p_mid').value = id;
  if (outcome !== undefined) { setOut(outcome); }
};

window.fillC = id => { document.getElementById('cl_mid').value = id; showPage('claim', null); };

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

window.build_create=function(){try{
  const _cat=getSelectedCat();
  const q=document.getElementById('c_question').value.trim();
  const cr=document.getElementById('c_creator').value.trim().toLowerCase();
  const b0=parseInt(document.getElementById('c_b0').value)*1000000;
  const exp=parseInt(document.getElementById('c_expiry').value)||window.currentHeight+1000;
  const fee=parseInt(document.getElementById('c_fee').value)||10000;
  let nonce=document.getElementById('c_nonce').value;
  if(!nonce)nonce=BigInt(Date.now())*1000n;
  else nonce=parseInt(nonce);
  const rules=document.getElementById('c_rules').value.trim();
  const _imgUrl=document.getElementById('c_img')?.value.trim()||'';
  if(!q)throw new Error('Question required');addr40(cr,'Creator');
  showPL('co','cp',buildUnsigned('create_market','type.googleapis.com/types.MessageCreateMarket',encCreate(cr,b0,exp,nonce,q,buildRulesWithOutcomes(buildRulesWithImg(buildRulesWithCat(_cat,rules),_imgUrl),document.getElementById('c_out_yes')?.value.trim()||'',document.getElementById('c_out_no')?.value.trim()||'')),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};

window.signAndSubmit_create=async function(){try{
  const _cat=getSelectedCat();
  const q=document.getElementById('c_question').value.trim();
  const cr=document.getElementById('c_creator').value.trim().toLowerCase();
  const b0=parseInt(document.getElementById('c_b0').value)*1000000;
  const exp=parseInt(document.getElementById('c_expiry').value)||window.currentHeight+1000;
  const fee=parseInt(document.getElementById('c_fee').value)||10000;
  let nonce=document.getElementById('c_nonce').value;
  if(!nonce)nonce=BigInt(Date.now())*1000n;
  else nonce=parseInt(nonce);
  const rules=document.getElementById('c_rules').value.trim();
  const _imgUrl=document.getElementById('c_img')?.value.trim()||'';
  if(!q)throw new Error('Question required');addr40(cr,'Creator');
  await doSubmit('create_market','type.googleapis.com/types.MessageCreateMarket',encCreate(cr,b0,exp,nonce,q,buildRulesWithOutcomes(buildRulesWithImg(buildRulesWithCat(_cat,rules),_imgUrl),document.getElementById('c_out_yes')?.value.trim()||'',document.getElementById('c_out_no')?.value.trim()||'')),{fee},'btn_create','pend_create');
}catch(e){toast(e.message,true);}};

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

window.build_propose=function(){try{
  const mid=document.getElementById('pr_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('pr_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const bond=parseInt(document.getElementById('pr_bond').value)*1000000;
  const fee=parseInt(document.getElementById('pr_fee').value)||10000;
  showPL('propo','propp',buildUnsigned('propose_outcome','type.googleapis.com/types.MessageProposeOutcome',encPropose(mid,res,propOut,bond),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};

window.signAndSubmit_propose=async function(){try{
  const mid=document.getElementById('pr_mid').value.trim().toLowerCase();mid40(mid);
  const res=document.getElementById('pr_resolver').value.trim().toLowerCase();addr40(res,'Resolver');
  const bond=parseInt(document.getElementById('pr_bond').value)*1000000;
  const fee=parseInt(document.getElementById('pr_fee').value)||10000;
  await doSubmit('propose_outcome','type.googleapis.com/types.MessageProposeOutcome',encPropose(mid,res,propOut,bond),{fee},'btn_propose','pend_propose');
}catch(e){toast(e.message,true);}};

window.build_dispute=function(){try{
  const mid=document.getElementById('di_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('di_addr').value.trim().toLowerCase();addr40(addr,'Disputer');
  const bond=parseInt(document.getElementById('dis_bond').value)*1000000;
  const fee=parseInt(document.getElementById('di_fee').value)||10000;
  showPL('diso','disp',buildUnsigned('file_dispute','type.googleapis.com/types.MessageFileDispute',encDispute(mid,addr,bond),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};

window.signAndSubmit_dispute=async function(){try{
  const mid=document.getElementById('di_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('di_addr').value.trim().toLowerCase();addr40(addr,'Disputer');
  const bond=parseInt(document.getElementById('dis_bond').value)*1000000;
  const fee=parseInt(document.getElementById('di_fee').value)||10000;
  await doSubmit('file_dispute','type.googleapis.com/types.MessageFileDispute',encDispute(mid,addr,bond),{fee},'btn_dispute','pend_dispute');
}catch(e){toast(e.message,true);}};

window.build_commit=function(){try{
  const mid=document.getElementById('cv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('cv_addr').value.trim().toLowerCase();addr40(voter,'Voter');
  const hash=document.getElementById('cv_hash').value.trim().toLowerCase();if(hash.length!==64)throw new Error('Commit hash must be 64 hex chars');
  const fee=parseInt(document.getElementById('cv_fee').value)||10000;
  showPL('cvo','cvp',buildUnsigned('commit_vote','type.googleapis.com/types.MessageCommitVote',encCommit(mid,voter,hash),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};

window.signAndSubmit_commit=async function(){try{
  const mid=document.getElementById('cv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('cv_addr').value.trim().toLowerCase();addr40(voter,'Voter');
  const hash=document.getElementById('cv_hash').value.trim().toLowerCase();if(hash.length!==64)throw new Error('Commit hash must be 64 hex chars');
  const fee=parseInt(document.getElementById('cv_fee').value)||10000;
  await doSubmit('commit_vote','type.googleapis.com/types.MessageCommitVote',encCommit(mid,voter,hash),{fee},'btn_commit','pend_commit');
}catch(e){toast(e.message,true);}};

window.build_reveal=function(){try{
  const mid=document.getElementById('rv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('rv_addr').value.trim().toLowerCase();addr40(voter,'Voter');
  const nonce=document.getElementById('rv_salt').value.trim().toLowerCase();if(nonce.length!==64)throw new Error('Nonce must be 64 hex chars');
  const fee=parseInt(document.getElementById('rv_fee').value)||10000;
  showPL('rvo','rvp',buildUnsigned('reveal_vote','type.googleapis.com/types.MessageRevealVote',encReveal(mid,voter,revOut,nonce),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};

window.signAndSubmit_reveal=async function(){try{
  const mid=document.getElementById('rv_mid').value.trim().toLowerCase();mid40(mid);
  const voter=document.getElementById('rv_addr').value.trim().toLowerCase();addr40(voter,'Voter');
  const nonce=document.getElementById('rv_salt').value.trim().toLowerCase();if(nonce.length!==64)throw new Error('Nonce must be 64 hex chars');
  const fee=parseInt(document.getElementById('rv_fee').value)||10000;
  await doSubmit('reveal_vote','type.googleapis.com/types.MessageRevealVote',encReveal(mid,voter,revOut,nonce),{fee},'btn_reveal','pend_reveal');
}catch(e){toast(e.message,true);}};

window.build_tally=function(){try{
  const mid=document.getElementById('ta_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('ta_addr').value.trim().toLowerCase();addr40(addr,'Caller');
  const fee=parseInt(document.getElementById('ta_fee').value)||10000;
  showPL('talo','talp',buildUnsigned('tally_votes','type.googleapis.com/types.MessageTallyVotes',encTally(mid,addr),{fee}));toast('Payload built');
}catch(e){toast(e.message,true);}};

window.signAndSubmit_tally=async function(){try{
  const mid=document.getElementById('ta_mid').value.trim().toLowerCase();mid40(mid);
  const addr=document.getElementById('ta_addr').value.trim().toLowerCase();addr40(addr,'Caller');
  const fee=parseInt(document.getElementById('ta_fee').value)||10000;
  await doSubmit('tally_votes','type.googleapis.com/types.MessageTallyVotes',encTally(mid,addr),{fee},'btn_tally','pend_tally');
}catch(e){toast(e.message,true);}};

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
  document.getElementById('pr_mid').value = id;
  if (signerAddress) document.getElementById('pr_resolver').value = signerAddress;
  showPage('propose', null);
  setTimeout(() => { updateMinBondHint(); updateProposeRisk(); }, 50);
};

window.build_claimcreator=function(){
  try{
    const mid=document.getElementById('cf_mid').value.trim().toLowerCase();mid40(mid);
    const addr=document.getElementById('cf_addr').value.trim().toLowerCase();addr40(addr,'Creator');
    const fee=parseInt(document.getElementById('cf_fee').value)||10000;
    showPL('ccfo','ccfp',buildUnsigned('claim_creator_fee','type.googleapis.com/types.MessageClaimCreatorFee',encClaimCreatorFee(mid,addr),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};

window.signAndSubmit_claimcreator=async function(){
  try{
    const mid=document.getElementById('cf_mid').value.trim().toLowerCase();mid40(mid);
    const addr=document.getElementById('cf_addr').value.trim().toLowerCase();addr40(addr,'Creator');
    const fee=parseInt(document.getElementById('cf_fee').value)||10000;
    await doSubmit('claim_creator_fee','type.googleapis.com/types.MessageClaimCreatorFee',encClaimCreatorFee(mid,addr),{fee},'btn_claimcreator','pend_claimcreator');
  }catch(e){toast(e.message,true);}
};

window.fillClaimCreator=function(id){
  document.getElementById('cf_mid').value=id;
  if(signerAddress)document.getElementById('cf_addr').value=signerAddress;
  showPage('claimcreator',null);
};

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

window.build_unstake_resolver=function(){
  try{
    const addr=document.getElementById('un_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const amount=parseInt(document.getElementById('un_amount').value||'0');
    const amountU=BigInt(amount)*1000000n;
    const fee=parseInt(document.getElementById('un_fee').value)||10000;
    showPL('unsto','unstp',buildUnsigned('unstake_resolver','type.googleapis.com/types.MessageUnstakeResolver',encUnstakeResolver(addr,amountU),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};

window.signAndSubmit_unstake_resolver=async function(){
  try{
    const addr=document.getElementById('un_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const amount=parseInt(document.getElementById('un_amount').value||'0');
    const amountU=BigInt(amount)*1000000n;
    const fee=parseInt(document.getElementById('un_fee').value)||10000;
    await doSubmit('unstake_resolver','type.googleapis.com/types.MessageUnstakeResolver',encUnstakeResolver(addr,amountU),{fee},'btn_unstake','pend_unstake');
  }catch(e){toast(e.message,true);}
};

window.build_claim_unbonded=function(){
  try{
    const addr=document.getElementById('ub_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const fee=parseInt(document.getElementById('ub_fee').value)||10000;
    showPL('cubo','cubp',buildUnsigned('claim_unbonded_stake','type.googleapis.com/types.MessageClaimUnbondedStake',encClaimUnbonded(addr),{fee}));
    toast('Payload built');
  }catch(e){toast(e.message,true);}
};

window.signAndSubmit_claim_unbonded=async function(){
  try{
    const addr=document.getElementById('ub_addr').value.trim().toLowerCase();addr40(addr,'Resolver');
    const fee=parseInt(document.getElementById('ub_fee').value)||10000;
    await doSubmit('claim_unbonded_stake','type.googleapis.com/types.MessageClaimUnbondedStake',encClaimUnbonded(addr),{fee},'btn_ub','pend_ub');
  }catch(e){toast(e.message,true);}
};
