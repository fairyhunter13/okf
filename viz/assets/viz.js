(function () {
  var D = window.__OKF__;
  var byId = {};
  D.nodes.forEach(function (n) { byId[n.id] = n; });

  // The twelve types the okf-knowledge-bundle skill tabulates, each with its
  // own colour. The reference viewer palettes three and greys the rest, which
  // renders Metric, Policy and Skill as the same thing.
  var PALETTE = {
    'Decision': '#2563eb', 'Component': '#0891b2', 'Interface': '#7c3aed',
    'Constraint': '#dc2626', 'Policy': '#db2777', 'Runbook': '#ea580c',
    'Skill': '#65a30d', 'Glossary Term': '#0d9488', 'Attested Computation': '#4f46e5',
    'Scenario': '#c026d3', 'Defect': '#b91c1c', 'Reference': '#475569'
  };
  var EXTRA = ['#059669', '#9333ea', '#e11d48', '#0284c7', '#a16207'];
  var assigned = {}, next = 0;
  function colour(t) {
    if (PALETTE[t]) return PALETTE[t];
    if (!(t in assigned)) { assigned[t] = EXTRA[next % EXTRA.length]; next++; }
    return assigned[t];
  }

  var svg = document.getElementById('svg');
  var root = document.getElementById('root');
  var NS = 'http://www.w3.org/2000/svg';
  function el(name, attrs) {
    var e = document.createElementNS(NS, name);
    for (var k in attrs) e.setAttribute(k, attrs[k]);
    return e;
  }

  D.edges.forEach(function (e) {
    var a = byId[e.f], b = byId[e.t];
    if (!a || !b) return;
    var l = el('line', { x1: a.x, y1: a.y, x2: b.x, y2: b.y, class: 'edge' });
    l.dataset.f = e.f; l.dataset.t = e.t;
    root.appendChild(l);
  });

  D.nodes.forEach(function (n) {
    var g = el('g', { class: 'node' + (n.stale ? ' stale' : '') });
    g.dataset.id = n.id;
    g.appendChild(el('circle', { cx: n.x, cy: n.y, r: n.r, fill: colour(n.type) }));
    var t = el('text', { x: n.x, y: n.y + n.r + 13 });
    t.textContent = n.title.length > 34 ? n.title.slice(0, 33) + '…' : n.title;
    g.appendChild(t);
    g.addEventListener('click', function () { select(n.id); });
    root.appendChild(g);
  });

  // Pan and zoom over the whole layer, so the SVG stays a fixed viewBox and the
  // node coordinates the generator computed are never recomputed here.
  var tx = 0, ty = 0, scale = 1, dragging = false, lx = 0, ly = 0;
  function apply() { root.setAttribute('transform', 'translate(' + tx + ',' + ty + ') scale(' + scale + ')'); }
  svg.addEventListener('mousedown', function (e) { dragging = true; lx = e.clientX; ly = e.clientY; svg.classList.add('drag'); });
  window.addEventListener('mouseup', function () { dragging = false; svg.classList.remove('drag'); });
  window.addEventListener('mousemove', function (e) {
    if (!dragging) return;
    tx += e.clientX - lx; ty += e.clientY - ly; lx = e.clientX; ly = e.clientY; apply();
  });
  svg.addEventListener('wheel', function (e) {
    e.preventDefault();
    var f = e.deltaY < 0 ? 1.12 : 1 / 1.12;
    var r = svg.getBoundingClientRect(), mx = e.clientX - r.left, my = e.clientY - r.top;
    tx = mx - (mx - tx) * f; ty = my - (my - ty) * f; scale *= f; apply();
  }, { passive: false });

  function reset() { tx = 0; ty = 0; scale = 1; apply(); }
  document.getElementById('reset').addEventListener('click', function () {
    reset(); document.getElementById('q').value = ''; document.getElementById('type').value = '';
    filter(); select(null);
  });

  function filter() {
    var q = document.getElementById('q').value.trim().toLowerCase();
    var ty2 = document.getElementById('type').value;
    var shown = 0;
    D.nodes.forEach(function (n) {
      var hay = (n.title + ' ' + n.id + ' ' + n.type + ' ' + (n.tags || []).join(' ') + ' ' + (n.description || '')).toLowerCase();
      var ok = (!q || hay.indexOf(q) >= 0) && (!ty2 || n.type === ty2);
      var g = root.querySelector('.node[data-id="' + CSS.escape(n.id) + '"]');
      if (g) g.classList.toggle('dim', !ok);
      if (ok) shown++;
    });
    document.getElementById('count').textContent =
      shown + ' of ' + D.nodes.length + ' concepts · ' + D.edges.length + ' links';
  }
  document.getElementById('q').addEventListener('input', filter);
  document.getElementById('type').addEventListener('change', filter);

  function esc(s) {
    return String(s == null ? '' : s).replace(/[&<>"]/g, function (c) {
      return { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c];
    });
  }

  function row(label, value) {
    if (!value || (value.length === 0)) return '';
    return '<dt>' + esc(label) + '</dt><dd>' + value + '</dd>';
  }

  function select(id) {
    root.querySelectorAll('.node').forEach(function (g) { g.classList.toggle('sel', g.dataset.id === id); });
    root.querySelectorAll('.edge').forEach(function (l) {
      l.classList.toggle('hot', !!id && (l.dataset.f === id || l.dataset.t === id));
    });
    var d = document.getElementById('detail');
    var n = id && byId[id];
    if (!n) { d.innerHTML = '<p class="empty">Pick a concept to read it. Drag to pan, scroll to zoom.</p>'; return; }

    var badges = '<span class="badge ' +
      (n.trust === 'human-reviewed' ? 'human' : n.trust === 'machine-confirmed' ? 'machine' : '') +
      '">' + esc(n.trust) + '</span>';
    if (n.status) badges += '<span class="badge">' + esc(n.status) + '</span>';
    if (n.stale) badges += '<span class="badge stale">stale</span>';

    var dl = row('Description', esc(n.description)) +
      row('Resource', esc(n.resource)) +
      row('Tags', (n.tags || []).map(esc).join(', ')) +
      row('Generated', esc(n.generated)) +
      row('Verified', (n.verified || []).map(esc).join('<br>')) +
      row('Stale after', esc(n.stale_after)) +
      row('Sources', (n.sources || []).map(function (s) {
        var label = esc(s.title || s.resource);
        var body = /^https?:\/\//.test(s.resource)
          ? '<a href="' + esc(s.resource) + '" rel="noreferrer">' + label + '</a>'
          : label;
        if (s.title && s.resource) body += '<br><code>' + esc(s.resource) + '</code>';
        if (s.last_modified) body += ' <span class="empty">' + esc(s.last_modified) + '</span>';
        return body;
      }).join('<br>'));

    var cites = (n.cited_by || []).map(function (c) {
      return '<li><a data-go="' + esc(c) + '">' + esc(byId[c] ? byId[c].title : c) + '</a></li>';
    }).join('');

    d.innerHTML =
      '<span class="chip" style="background:' + colour(n.type) + '">' + esc(n.type || 'untyped') + '</span>' +
      '<h1>' + esc(n.title) + '</h1>' +
      '<div class="id">' + esc(n.id) + '</div>' +
      '<div style="margin-top:8px">' + badges + '</div>' +
      (dl ? '<dl>' + dl + '</dl>' : '') +
      '<div class="body">' + n.body + '</div>' +
      (cites ? '<h3>Cited by</h3><ul class="cites">' + cites + '</ul>' : '');

    // An intra-bundle link selects the node instead of navigating: the page is
    // one file, so following the href would only ever 404.
    d.querySelectorAll('a[href]').forEach(function (a) {
      var href = a.getAttribute('href');
      var target = href.charAt(0) === '/' ? href.slice(1) : join(dir(n.id), href.split('#')[0]);
      if (byId[target]) {
        a.removeAttribute('href');
        a.dataset.go = target;
      } else if (!/^https?:/.test(href)) {
        a.removeAttribute('href');
      }
    });
    d.querySelectorAll('[data-go]').forEach(function (a) {
      a.addEventListener('click', function () { select(a.dataset.go); });
    });
    d.scrollTop = 0;
  }

  function dir(p) { var i = p.lastIndexOf('/'); return i < 0 ? '' : p.slice(0, i); }
  function join(base, rel) {
    var parts = (base ? base.split('/') : []).concat(rel.split('/'));
    var out = [];
    parts.forEach(function (p) {
      if (p === '.' || p === '') return;
      if (p === '..') out.pop(); else out.push(p);
    });
    return out.join('/');
  }

  D.types.forEach(function (t) {
    var o = document.createElement('option');
    o.value = t; o.textContent = t;
    document.getElementById('type').appendChild(o);
  });
  filter();
  select(null);
})();
