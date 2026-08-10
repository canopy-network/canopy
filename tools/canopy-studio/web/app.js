/**
 * Canopy Appchain Studio Client Logic
 */

document.addEventListener('DOMContentLoaded', () => {
  initTabs();
  loadConfig();
  loadHistory();
  initFormListeners();
});

function initTabs() {
  const tabs = document.querySelectorAll('.nav-tab');
  tabs.forEach(tab => {
    tab.addEventListener('click', () => {
      document.querySelectorAll('.nav-tab').forEach(t => t.classList.toggle('active', t === tab));
      document.querySelectorAll('.tab-pane').forEach(p => p.classList.toggle('active', p.id === `tab-${tab.dataset.tab}`));
    });
  });
}

async function loadConfig() {
  try {
    const res = await fetch('/api/config');
    const data = await res.json();

    document.getElementById('header-mesh').textContent = data.network.name;
    document.getElementById('mesh-val-count').textContent = `${data.securityMesh.activeValidatorsCount} Node Operators`;
    document.getElementById('mesh-staked').textContent = data.securityMesh.totalStakedCNPY;

    const grid = document.getElementById('chains-container');
    grid.innerHTML = '';

    data.nestedChains.forEach(c => {
      const card = document.createElement('div');
      card.className = 'chain-card';
      card.innerHTML = `
        <div class="chain-title">${c.name}</div>
        <div class="chain-type">${c.chainType}</div>
        <div class="text-muted" style="font-size: 0.82rem;">Validators: <strong>${c.validators}</strong></div>
        <div class="text-muted mt-1" style="font-size: 0.82rem;">Staked: <strong style="color: #34d399;">${c.stakedCnpy}</strong></div>
      `;
      grid.appendChild(card);
    });
  } catch (e) {
    console.error(e);
  }
}

async function loadHistory() {
  try {
    const res = await fetch('/api/appchain/history');
    const logs = await res.json();
    const container = document.getElementById('launch-history-container');

    if (!logs || logs.length === 0) return;
    container.innerHTML = '';

    logs.forEach(l => {
      const row = document.createElement('div');
      row.className = 'ledger-row';
      row.innerHTML = `
        <div>
          <div style="font-weight: 700; color: #fff;">${l.appchain.name}</div>
          <div class="mono text-muted" style="font-size: 0.72rem;">Genesis: ${l.appchain.genesisHash.slice(0, 16)}...</div>
        </div>
        <div style="text-align: right;">
          <div style="color: #34d399; font-weight: 700; font-size: 0.8rem;">${l.appchain.validators} Validators</div>
          <div class="text-muted" style="font-size: 0.75rem;">${new Date(l.timestamp).toLocaleTimeString()}</div>
        </div>
      `;
      container.appendChild(row);
    });
  } catch (e) {
    console.warn(e);
  }
}

function initFormListeners() {
  document.getElementById('launch-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    const btn = document.getElementById('btn-deploy-chain');
    const resultBox = document.getElementById('launch-result-box');

    const name = document.getElementById('app-name').value;
    const category = document.getElementById('app-category').value;
    const initialValidatorCount = document.getElementById('app-validators').value;
    const minStakeAmount = document.getElementById('app-stake').value;

    btn.disabled = true;
    btn.textContent = '⏳ Registering Recursive Appchain on Canopy...';

    try {
      const res = await fetch('/api/appchain/launch', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name, category, initialValidatorCount, minStakeAmount }),
      });
      const data = await res.json();

      if (data.success) {
        resultBox.innerHTML = `
          <div class="card" style="border-color: #10b981; background: rgba(16, 185, 129, 0.08);">
            <strong style="color: #34d399;">🌲 Nested Sovereign Appchain Deployed!</strong>
            <p class="mt-2" style="font-size: 0.9rem;">Appchain ID: <strong class="mono">${data.appchain.id}</strong></p>
            <div class="mono text-muted mt-1" style="font-size: 0.75rem;">Genesis TX: ${data.appchain.registerTx}</div>
          </div>
        `;
        loadConfig();
        loadHistory();
      }
    } catch (err) {
      resultBox.innerHTML = `<div class="badge red">Deployment error: ${err.message}</div>`;
    } finally {
      btn.disabled = false;
      btn.textContent = '🌲 Deploy & Register on Canopy Seed';
    }
  });
}
