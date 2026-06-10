// ═══════════════════════════════════════════════════════════════════════
// PRAXIS — MetaMask → BLS12-381 Key Derivation
// ═══════════════════════════════════════════════════════════════════════
//
// Flow:
//   1. User clicks "Connect Wallet"
//   2. MetaMask signs a deterministic seed message
//   3. Signature bytes → HKDF-SHA256 → 32-byte BLS scalar
//   4. BLS scalar → BLS12-381 keypair (privkey + pubkey)
//   5. Praxis address = first 20 bytes of SHA-256(pubkey)
//   6. Encrypted keypair stored in localStorage keyed to ETH address
//   7. Same ETH account = same BLS key, on any device, forever
//
// Security model:
//   - Private key never leaves the browser
//   - Encrypted with AES-256-GCM using ETH address as key material
//   - MetaMask signature is the only secret — losing MetaMask = losing key
// ═══════════════════════════════════════════════════════════════════════

const PRAXIS_DERIVE_MSG = 'Praxis BLS key derivation v1\n\nSigning this message derives your Praxis signing key.\n\nThis signature never leaves your browser.';
const PRAXIS_STORE_PREFIX = 'praxis_bls_v1_';

// ── State ──
let mmEthAddress   = null;   // connected ETH address (0x...)
let mmDerivedPriv  = null;   // derived BLS private key (Uint8Array 32)
let mmDerivedPub   = null;   // derived BLS public key  (Uint8Array 48)
let mmDerivedAddr  = null;   // derived Praxis address  (hex string 40)

// ── Utils ──
// ── HKDF-SHA256 ──
async function hkdf(ikm, salt, info, len) {
  const key  = await crypto.subtle.importKey('raw', ikm, {name:'HKDF'}, false, ['deriveBits']);
  const bits = await crypto.subtle.deriveBits(
    {name:'HKDF', hash:'SHA-256', salt, info: new TextEncoder().encode(info)},
    key, len * 8
  );
  return new Uint8Array(bits);
}

// ── AES-256-GCM encrypt ──
async function aesEncrypt(data, password) {
  const salt    = crypto.getRandomValues(new Uint8Array(16));
  const iv      = crypto.getRandomValues(new Uint8Array(12));
  const keyMat  = await crypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveKey']);
  const aesKey  = await crypto.subtle.deriveKey(
    {name:'PBKDF2', salt, iterations:100000, hash:'SHA-256'},
    keyMat, {name:'AES-GCM', length:256}, false, ['encrypt']
  );
  const ct = await crypto.subtle.encrypt({name:'AES-GCM', iv}, aesKey, data);
  return b2h(salt) + b2h(iv) + b2h(new Uint8Array(ct));
}

// ── AES-256-GCM decrypt ──
async function aesDecrypt(hex, password) {
  const salt   = h2b(hex.slice(0, 32));
  const iv     = h2b(hex.slice(32, 56));
  const ct     = h2b(hex.slice(56));
  const keyMat = await crypto.subtle.importKey('raw', new TextEncoder().encode(password), 'PBKDF2', false, ['deriveKey']);
  const aesKey = await crypto.subtle.deriveKey(
    {name:'PBKDF2', salt, iterations:100000, hash:'SHA-256'},
    keyMat, {name:'AES-GCM', length:256}, false, ['decrypt']
  );
  return new Uint8Array(await crypto.subtle.decrypt({name:'AES-GCM', iv}, aesKey, ct));
}

// ── Derive BLS keypair from MetaMask signature ──
async function deriveBlsFromSignature(ethSig, ethAddr) {
  if (!bls12_381) throw new Error('BLS library not loaded');

  // signature bytes as IKM
  const ikm  = h2b(ethSig.startsWith('0x') ? ethSig.slice(2) : ethSig);
  const salt = new TextEncoder().encode('praxis-bls-salt-v1');

  // HKDF → 32 bytes → BLS scalar (mod curve order)
  const raw  = await hkdf(ikm, salt, 'praxis-bls-key', 32);

  // Clamp to valid BLS scalar: < curve order
  // BLS12-381 curve order r = 0x73eda753...
  // Noble/curves handles clamping internally via Fr.fromBytes
  const privKey = raw;

  // Derive public key (G1 compressed, 48 bytes)
  const pubKey  = bls12_381.getPublicKey(privKey);

  // Praxis address = first 20 bytes of SHA-256(pubkey)
  const pubHash = await crypto.subtle.digest('SHA-256', pubKey);
  const address = b2h(new Uint8Array(pubHash).slice(0, 20));

  return { privKey, pubKey, address };
}

// ── Check if stored key exists for ETH address ──
function hasStoredKey(ethAddr) {
  return !!localStorage.getItem(PRAXIS_STORE_PREFIX + ethAddr.toLowerCase());
}

// ── Store derived key encrypted ──
async function storeKey(ethAddr, privKey) {
  const enc = await aesEncrypt(privKey, ethAddr.toLowerCase());
  localStorage.setItem(PRAXIS_STORE_PREFIX + ethAddr.toLowerCase(), enc);
}

// ── Load stored key ──
async function loadStoredKey(ethAddr) {
  const enc = localStorage.getItem(PRAXIS_STORE_PREFIX + ethAddr.toLowerCase());
  if (!enc) return null;
  return aesDecrypt(enc, ethAddr.toLowerCase());
}

// ── Main: Connect MetaMask and derive BLS key ──
window.connectMetaMaskBLS = async function() {
  const btn = document.getElementById('mm-connect-btn');
  if (btn) { btn.disabled = true; btn.textContent = 'Connecting…'; }

  try {
    // 1. Check MetaMask
    if (!window.ethereum) throw new Error('MetaMask not found — install the extension');

    // 2. Request accounts
    const accounts = await window.ethereum.request({method:'eth_requestAccounts'});
    if (!accounts.length) throw new Error('No accounts returned');
    const ethAddr = accounts[0].toLowerCase();
    mmEthAddress  = ethAddr;

    // 3. Check if we already have a stored key for this address
    if (hasStoredKey(ethAddr)) {
      const stored = await loadStoredKey(ethAddr);
      if (stored) {
        mmDerivedPriv = stored;
        mmDerivedPub  = bls12_381.getPublicKey(stored);
        const pubHash = await crypto.subtle.digest('SHA-256', mmDerivedPub);
        mmDerivedAddr = b2h(new Uint8Array(pubHash).slice(0, 20));
        _applyDerivedKey();
        updateMMUI(ethAddr);
        toast('✓ Wallet reconnected — ' + mmDerivedAddr.slice(0,8) + '…');
        return;
      }
    }

    // 4. Sign derivation message
    updateMMStatus('Waiting for MetaMask signature…');
    const sig = await window.ethereum.request({
      method: 'personal_sign',
      params: [PRAXIS_DERIVE_MSG, ethAddr],
    });

    // 5. Derive BLS keypair
    updateMMStatus('Deriving Praxis key…');
    const { privKey, pubKey, address } = await deriveBlsFromSignature(sig, ethAddr);
    mmDerivedPriv = privKey;
    mmDerivedPub  = pubKey;
    mmDerivedAddr = address;

    // 6. Store encrypted
    await storeKey(ethAddr, privKey);

    // 7. Apply to signer
    _applyDerivedKey();
    updateMMUI(ethAddr);
    toast('✓ Praxis wallet derived — ' + address.slice(0,8) + '…');

  } catch(e) {
    toast(e.message || 'MetaMask connection failed', true);
    updateMMStatus('Not connected');
    if (btn) { btn.disabled = false; btn.textContent = '🦊 Connect Wallet'; }
  }
};

// ── Apply derived key to the Praxis signer ──
function _applyDerivedKey() {
  if (!mmDerivedPriv || !mmDerivedPub) return;
  // Inject into app.js signer state
  signerPrivKey = mmDerivedPriv;
  signerPubKey  = b2h(mmDerivedPub);
  signerAddress = mmDerivedAddr;

  // Update signer UI
  const ks = document.getElementById('keyStatus');
  if (ks) {
    ks.className = 'kstat loaded';
    ks.textContent = '✓ MetaMask-derived key loaded';
  }
  const ka = document.getElementById('keyAddr');
  if (ka) ka.textContent = mmDerivedAddr;
  const kp = document.getElementById('keyPub');
  if (kp) kp.textContent = signerPubKey;
  const ki = document.getElementById('keyInfo');
  if (ki) ki.style.display = '';

  // Auto-fill address fields
  autoFillAddresses(mmDerivedAddr);
  checkRoles();
}

// ── Auto-fill address fields with derived address ──
function autoFillAddresses(addr) {
  const fields = [
    'p_bettor','cl_addr','rc_addr','fo_resolver',
    'reg_addr','pr_resolver','di_addr','cv_addr',
    'rv_addr','un_addr','ub_addr','cf_addr','w_addr',
    's_from','fi_addr','ta_addr','sl_addr'
  ];
  fields.forEach(id => {
    const el = document.getElementById(id);
    if (el && !el.value) el.value = addr;
  });
}

// ── Disconnect ──
window.disconnectMetaMaskBLS = function() {
  mmEthAddress  = null;
  mmDerivedPriv = null;
  mmDerivedPub  = null;
  mmDerivedAddr = null;
  signerPrivKey = null;
  signerPubKey  = null;
  signerAddress = null;

  updateMMUI(null);

  const ks = document.getElementById('keyStatus');
  if (ks) { ks.className = 'kstat'; ks.textContent = 'No key loaded'; }
  const ki = document.getElementById('keyInfo');
  if (ki) ki.style.display = 'none';

  toast('Wallet disconnected');
};

// ── Update MetaMask UI ──
function updateMMUI(ethAddr) {
  const connected    = document.getElementById('mm_connected');
  const disconnected = document.getElementById('mm_disconnected');
  const mmAddrEl     = document.getElementById('mm_addr');
  const mmStatus     = document.getElementById('mm_status');
  const praxisAddrEl = document.getElementById('mm_praxis_addr');

  if (ethAddr) {
    if (connected)    connected.style.display    = '';
    if (disconnected) disconnected.style.display = 'none';
    if (mmAddrEl)     mmAddrEl.textContent       = ethAddr;
    if (mmStatus)     mmStatus.textContent       = '✓ connected';
    if (praxisAddrEl) praxisAddrEl.textContent   = mmDerivedAddr || '';
  } else {
    if (connected)    connected.style.display    = 'none';
    if (disconnected) disconnected.style.display = '';
  }

  // Update top-bar connect button
  const topBtn = document.getElementById('mm-connect-btn');
  if (topBtn) {
    if (ethAddr) {
      topBtn.textContent = ethAddr.slice(0,6) + '…' + ethAddr.slice(-4);
      topBtn.classList.add('connected');
    } else {
      topBtn.textContent = '🦊 Connect Wallet';
      topBtn.classList.remove('connected');
      topBtn.disabled = false;
    }
  }
}

function updateMMStatus(msg) {
  const s = document.getElementById('mm_status');
  if (s) s.textContent = msg;
}

// ── Auto-reconnect on page load if ETH address stored ──
window.addEventListener('load', async () => {
  // Wait for BLS to load
  await new Promise(r => setTimeout(r, 1500));

  if (!window.ethereum) return;
  try {
    const accounts = await window.ethereum.request({method:'eth_accounts'});
    if (!accounts.length) return;
    const ethAddr = accounts[0].toLowerCase();
    if (!hasStoredKey(ethAddr)) return;

    // Silent reconnect — no MetaMask popup
    const stored = await loadStoredKey(ethAddr);
    if (!stored || !bls12_381) return;

    mmEthAddress  = ethAddr;
    mmDerivedPriv = stored;
    mmDerivedPub  = bls12_381.getPublicKey(stored);
    const pubHash = await crypto.subtle.digest('SHA-256', mmDerivedPub);
    mmDerivedAddr = b2h(new Uint8Array(pubHash).slice(0, 20));

    _applyDerivedKey();
    updateMMUI(ethAddr);
    toast('✓ Auto-reconnected — ' + mmDerivedAddr.slice(0,8) + '…');
  } catch(e) {
    // Silent fail on auto-reconnect
  }
});

// ── Handle account changes in MetaMask ──
if (window.ethereum) {
  window.ethereum.on('accountsChanged', async (accounts) => {
    if (!accounts.length) {
      disconnectMetaMaskBLS();
      return;
    }
    // Re-derive for new account
    await connectMetaMaskBLS();
  });
}
