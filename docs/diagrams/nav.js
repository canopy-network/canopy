(function () {
  var PAGES = [
    { file: 'oracle-order-lifecycle.html', title: 'Oracle Order Lifecycle', blurb: 'sell order path through anvil + oracle' },
  ];

  var current = location.pathname.split('/').pop();

  var style = document.createElement('style');
  style.textContent = [
    '.diagram-nav {',
    '  position: fixed; top: 0; left: 0; height: 100vh; width: 220px;',
    '  overflow-y: auto; background: #f5f5f5; border-right: 1px solid rgba(45,49,66,0.12);',
    '  padding: 1.5rem 1rem; box-sizing: border-box; z-index: 1000;',
    '  font-family: "Geist", system-ui, sans-serif;',
    '}',
    '.diagram-nav .nav-eyebrow {',
    '  font-family: "Geist Mono", ui-monospace, monospace; font-size: 0.62rem;',
    '  letter-spacing: 0.18em; text-transform: uppercase; color: #4f5d75;',
    '  margin-bottom: 0.75rem;',
    '}',
    '.diagram-nav a {',
    '  display: block; text-decoration: none; color: #2d3142;',
    '  padding: 0.5rem 0.5rem; border-radius: 6px; margin-bottom: 0.25rem;',
    '}',
    '.diagram-nav a:hover { background: rgba(45,49,66,0.06); }',
    '.diagram-nav a.active { background: rgba(235,108,54,0.08); border: 1px solid #eb6c36; }',
    '.diagram-nav .nav-title { font-size: 0.82rem; font-weight: 600; }',
    '.diagram-nav .nav-blurb {',
    '  font-family: "Geist Mono", ui-monospace, monospace; font-size: 0.66rem;',
    '  color: #4f5d75; margin-top: 0.15rem;',
    '}',
    'body { margin-left: 220px; }',
  ].join('\n');
  document.head.appendChild(style);

  var aside = document.createElement('aside');
  aside.className = 'diagram-nav';

  var eyebrow = document.createElement('p');
  eyebrow.className = 'nav-eyebrow';
  eyebrow.textContent = 'Diagrams · ' + PAGES.length;
  aside.appendChild(eyebrow);

  PAGES.forEach(function (page) {
    var a = document.createElement('a');
    a.href = page.file;
    if (page.file === current) a.className = 'active';
    var title = document.createElement('div');
    title.className = 'nav-title';
    title.textContent = page.title;
    var blurb = document.createElement('div');
    blurb.className = 'nav-blurb';
    blurb.textContent = page.blurb;
    a.appendChild(title);
    a.appendChild(blurb);
    aside.appendChild(a);
  });

  document.body.insertBefore(aside, document.body.firstChild);
})();
