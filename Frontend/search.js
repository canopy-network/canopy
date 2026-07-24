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
