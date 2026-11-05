(function () {
  'use strict';

  var params = new URLSearchParams(location.search);
  if (params.has('overlay')) document.body.classList.add('overlay');

  // Settings survive reloads; storage can be unavailable (e.g. some embedded browsers), so never rely on it.
  function load(key, fallback) {
    try { var v = localStorage.getItem(key); return v === null ? fallback : v; } catch (e) { return fallback; }
  }
  function save(key, value) {
    try { localStorage.setItem(key, value); } catch (e) { /* ignore */ }
  }

  var view = params.get('view') === 'enemy' ? 'enemy' : params.get('view') === 'item' ? 'item' : load('view2', 'item');  // 'item' | 'enemy'
  var foundOnly = load('foundOnly', '0') === '1';
  var filter = '';
  var state = null;
  var serverUp = false;

  function $(id) { return document.getElementById(id); }
  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text !== undefined) e.textContent = text;
    return e;
  }
  function clear(node) { while (node.firstChild) node.removeChild(node.firstChild); }

  // Every sprite is shown at its natural size in a fixed-size box (the box sizes are in style.css). The files are
  // scaled ahead of time so all item icons are one size and all enemy icons another,
  // and so nothing is scaled in the browser. A missing sprite leaves an empty box.
  function sprite(kind, file) {
    var box = el('span', 'sprite-box ' + kind + '-sprite');
    if (!file) return box;
    var img = new Image();
    img.alt = '';
    img.src = 'img/' + file;
    box.appendChild(img);
    return box;
  }

  // ---- rendering ---------------------------------------------------------------------------------------

  function renderStatus() {
    var dot = $('dot'), text = $('connText'), notice = $('notice');
    var cls, msg;
    if (!serverUp) {
      cls = 'bad'; msg = 'lost contact with mudora-clipboard';
    } else if (!state) {
      cls = 'wait'; msg = 'loading...';
    } else if (state.ready) {
      cls = 'ok'; msg = 'tracking ' + state.device;
    } else if (state.device) {
      cls = 'wait'; msg = state.device;
    } else {
      cls = 'bad'; msg = 'not connected';
    }
    dot.className = cls;
    text.textContent = msg;
    document.body.classList.toggle('stale', !serverUp || !state || !state.ready);

    var note = state && !state.ready && state.notice ? state.notice : '';
    notice.hidden = note === '';
    notice.textContent = note;
  }

  function renderProgress() {
    if (!state) return;
    var pct = state.total ? (100 * state.found / state.total) : 0;
    $('barFill').style.width = pct.toFixed(1) + '%';
    $('count').textContent = state.found + ' / ' + state.total + ' enemies found';
  }

  // The feed: newest first.
  function renderRecent() {
    var list = $('recentList');
    clear(list);
    var items = state && state.recent ? state.recent : [];
    $('recentEmpty').hidden = items.length !== 0;
    items.forEach(function (r) {
      var li = el('li');
      li.appendChild(sprite('enemy', r.enemySprite));
      var text = el('span', 'text');
      text.appendChild(el('span', 'name', r.name));
      text.appendChild(el('span', 'item', r.itemName));
      li.appendChild(text);
      li.appendChild(sprite('item', r.sprite));
      li.appendChild(el('span', 'at', r.at));
      list.appendChild(li);
    });
  }

  function has(haystack, needle) { return haystack.toLowerCase().indexOf(needle) !== -1; }

  function renderItems(root) {
    root.className = 'items';
    var f = filter.toLowerCase();
    var shown = 0;
    state.items.forEach(function (it) {
      var any = it.enemies.length > 0;
      if (foundOnly && !any) return;
      if (f && !has(it.name, f) && !it.enemies.some(function (e) { return has(e.name, f); })) return;
      shown++;

      var card = el('div', 'item-card ' + (any ? 'has' : 'none'));
      var head = el('div', 'item-head');
      head.appendChild(sprite('item', it.sprite));
      head.appendChild(el('span', 'item-name', it.name));
      if (any) head.appendChild(el('span', 'n', String(it.enemies.length)));
      card.appendChild(head);

      if (any) {
        var list = el('ul', 'enemy-list');
        it.enemies.forEach(function (e) {
          var li = el('li', e.found ? '' : 'unfound');
          li.appendChild(sprite('enemy', e.sprite));
          li.appendChild(el('span', 'ename', e.name));
          list.appendChild(li);
        });
        card.appendChild(list);
      } else {
        card.appendChild(el('div', 'none-note', 'not found yet'));
      }
      root.appendChild(card);
    });
    if (shown === 0) root.appendChild(el('div', 'empty', 'Nothing to show.'));
  }

  function renderEnemies(root) {
    root.className = 'enemies';
    var f = filter.toLowerCase();
    var shown = 0;
    state.enemies.forEach(function (e) {
      if (foundOnly && !e.found) return;
      if (f && !has(e.name, f) && !(e.itemName && has(e.itemName, f))) return;
      shown++;
      var revealedItem = !e.found && e.itemName;
      var card = el('div', 'card' + (e.found ? ' found' : '') + (revealedItem ? ' revealed' : '') +
        (!e.found && !revealedItem ? ' hidden-item' : ''));
      card.appendChild(sprite('enemy', e.sprite));
      var text = el('span', 'text');
      text.appendChild(el('span', 'name', e.name));
      text.appendChild(el('span', 'item', e.itemName ? e.itemName : '???'));
      card.appendChild(text);
      if (e.itemSprite) card.appendChild(sprite('item', e.itemSprite));
      root.appendChild(card);
    });
    if (shown === 0) root.appendChild(el('div', 'empty', 'Nothing to show.'));
  }

  function renderContent() {
    var root = $('content');
    clear(root);
    if (!state) return;
    if (view === 'enemy') renderEnemies(root); else renderItems(root);
  }

  function renderControls() {
    $('viewItem').classList.toggle('on', view === 'item');
    $('viewEnemy').classList.toggle('on', view === 'enemy');
    $('foundOnly').checked = foundOnly;
  }

  function render() {
    renderStatus();
    renderProgress();
    renderRecent();
    renderControls();
    renderContent();
  }

  // ---- controls ----------------------------------------------------------------------------------------

  $('viewItem').addEventListener('click', function () { view = 'item'; save('view2', view); render(); });
  $('viewEnemy').addEventListener('click', function () { view = 'enemy'; save('view2', view); render(); });
  $('foundOnly').addEventListener('change', function (ev) {
    foundOnly = ev.target.checked; save('foundOnly', foundOnly ? '1' : '0'); render();
  });
  $('filter').addEventListener('input', function (ev) { filter = ev.target.value.trim(); renderContent(); });

  // ---- data --------------------------------------------------------------------------------------------

  function apply(s) {
    if (state && s.version < state.version) return;   // an older response arrived late
    state = s;
    serverUp = true;
    render();
  }

  function connect() {
    var es = new EventSource('api/events');
    es.onmessage = function (ev) { apply(JSON.parse(ev.data)); };
    es.onerror = function () { serverUp = false; renderStatus(); };   // EventSource reconnects by itself
  }

  fetch('api/state', { cache: 'no-store' })
    .then(function (r) { return r.json(); })
    .then(apply)
    .catch(function () { serverUp = false; renderStatus(); });
  connect();
  render();
})();
