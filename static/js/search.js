/* goblog instant search palette. Loaded on demand by enhance.js the first time
   search is opened; talks to GET /api/search. Exposes window.goblogSearch.open(). */
(function () {
    'use strict';

    var G = window.goblog;
    if (!G || window.goblogSearch) { return; }

    var RECENT_KEY = 'recent-searches';
    var RECENT_MAX = 6;
    var ICON_SEARCH = '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>';

    var root = document.createElement('div');
    root.className = 'palette';
    root.hidden = true;
    root.innerHTML =
        '<div class="palette-backdrop" data-close="1"></div>' +
        '<div class="palette-panel" role="dialog" aria-modal="true" aria-label="搜索文章">' +
        '  <div class="palette-input-row">' + ICON_SEARCH +
        '    <input id="palette-input" type="search" placeholder="搜索文章、标签、分类…" maxlength="50" autocomplete="off" autocapitalize="off"' +
        '           spellcheck="false" enterkeyhint="search" role="combobox" aria-expanded="true" aria-controls="palette-results" aria-autocomplete="list">' +
        '    <button type="button" class="palette-esc" data-close="1" aria-label="关闭搜索"><kbd>Esc</kbd><span>取消</span></button>' +
        '  </div>' +
        '  <div class="palette-results" id="palette-results" role="listbox" aria-label="搜索结果"></div>' +
        '  <div class="palette-footer">' +
        '    <span class="palette-keys"><kbd>↑</kbd><kbd>↓</kbd> 选择 <kbd>Enter</kbd> 打开</span>' +
        '    <a class="palette-all" hidden></a>' +
        '  </div>' +
        '</div>';
    document.body.appendChild(root);

    var input = root.querySelector('#palette-input');
    var results = root.querySelector('#palette-results');
    var allLink = root.querySelector('.palette-all');
    var cache = {};
    var hotCache = null;
    var controller = null;
    var debounceTimer = null;
    var composing = false;
    var active = -1;
    var lastFocus = null;
    var seq = 0;

    function esc(s) {
        var d = document.createElement('div');
        d.appendChild(document.createTextNode(s == null ? '' : String(s)));
        return d.innerHTML;
    }

    function recent() {
        try { return JSON.parse(G.storageGet(RECENT_KEY) || '[]') || []; } catch (e) { return []; }
    }

    function remember(q) {
        q = (q || '').trim();
        if (!q) { return; }
        var list = recent().filter(function (x) { return x !== q; });
        list.unshift(q);
        G.storageSet(RECENT_KEY, JSON.stringify(list.slice(0, RECENT_MAX)));
    }

    function options() {
        return results.querySelectorAll('[role="option"]');
    }

    function setActive(i, scroll) {
        var opts = options();
        if (!opts.length) { active = -1; input.removeAttribute('aria-activedescendant'); return; }
        active = (i + opts.length) % opts.length;
        G.each(opts, function (el, idx) {
            var on = idx === active;
            el.classList.toggle('is-active', on);
            el.setAttribute('aria-selected', on ? 'true' : 'false');
        });
        input.setAttribute('aria-activedescendant', opts[active].id);
        if (scroll) { opts[active].scrollIntoView({block: 'nearest'}); }
    }

    function itemHTML(item, i) {
        var meta = [item.category, item.date].filter(Boolean).map(esc).join(' · ');
        return '<a class="palette-item" role="option" id="palette-opt-' + i + '" aria-selected="false" href="' + esc(item.url) + '">' +
            '<span class="palette-item-title">' + item.title_html + '</span>' +
            (item.snippet_html ? '<span class="palette-item-snippet">' + item.snippet_html + '</span>' : '') +
            '<span class="palette-item-meta">' + meta + '</span></a>';
    }

    function renderHome(hot) {
        var html = '';
        var rec = recent();
        if (rec.length) {
            html += '<div class="palette-section"><div class="palette-section-title">最近搜索<button type="button" class="palette-clear">清除</button></div><div class="palette-chips">' +
                rec.map(function (q) { return '<button type="button" class="chip palette-recent">' + esc(q) + '</button>'; }).join('') + '</div></div>';
        }
        html += '<div class="palette-section"><div class="palette-section-title">快速前往</div><div class="palette-chips">' +
            '<a class="chip" href="/random">🎲 随便看看</a><a class="chip" href="/archive">归档</a><a class="chip" href="/tags">标签</a><a class="chip" href="/reading">阅读清单</a></div></div>';
        if (hot && hot.length) {
            html += '<div class="palette-section"><div class="palette-section-title">热门文章</div>' +
                hot.map(function (item, i) { return itemHTML(item, i); }).join('') + '</div>';
        }
        results.innerHTML = html;
        allLink.hidden = true;
        active = -1;
        input.removeAttribute('aria-activedescendant');
    }

    function renderResults(data) {
        if (!data.items.length) {
            results.innerHTML = '<div class="palette-empty">没有找到与「' + esc(data.query) + '」相关的内容<small>试试更短的关键词，或者用空格分隔多个词</small></div>';
            allLink.hidden = true;
            active = -1;
            return;
        }
        results.innerHTML = data.items.map(itemHTML).join('');
        allLink.hidden = false;
        allLink.href = '/?keyword=' + encodeURIComponent(data.query);
        allLink.textContent = data.total > data.items.length ? '查看全部 ' + data.total + ' 条结果' : '在列表中查看 ' + data.total + ' 条结果';
        setActive(0, false);
    }

    function request(q) {
        var my = ++seq;
        if (controller) { controller.abort(); }
        controller = window.AbortController ? new AbortController() : null;
        return fetch('/api/search?q=' + encodeURIComponent(q), {
            headers: {Accept: 'application/json'},
            signal: controller ? controller.signal : undefined
        }).then(function (res) {
            if (!res.ok) { throw new Error('HTTP ' + res.status); }
            return res.json();
        }).then(function (data) {
            return {stale: my !== seq, data: data};
        });
    }

    function search() {
        var q = input.value.trim();
        if (!q) {
            if (hotCache) { renderHome(hotCache); return; }
            renderHome(null);
            request('').then(function (r) {
                hotCache = r.data.hot || [];
                if (!r.stale && !input.value.trim()) { renderHome(hotCache); }
            }).catch(function () { /* the static part of the home view is enough */ });
            return;
        }
        if (cache[q]) { renderResults(cache[q]); return; }
        results.setAttribute('aria-busy', 'true');
        request(q).then(function (r) {
            cache[q] = r.data;
            if (!r.stale) {
                results.removeAttribute('aria-busy');
                renderResults(r.data);
            }
        }).catch(function (err) {
            if (err && err.name === 'AbortError') { return; }
            results.removeAttribute('aria-busy');
            results.innerHTML = '<div class="palette-empty">搜索暂时不可用<small>按 Enter 改用普通搜索</small></div>';
            allLink.hidden = true;
            active = -1;
        });
    }

    function schedule() {
        clearTimeout(debounceTimer);
        debounceTimer = setTimeout(search, 140);
    }

    function open(initial) {
        if (!root.hidden) { input.focus(); return; }
        lastFocus = document.activeElement;
        root.hidden = false;
        document.documentElement.style.overflow = 'hidden';
        void root.offsetWidth;
        root.classList.add('is-open');
        input.value = initial || '';
        search();
        input.focus();
        input.select();
    }

    function close() {
        if (root.hidden) { return; }
        root.classList.remove('is-open');
        document.documentElement.style.overflow = '';
        clearTimeout(debounceTimer);
        if (controller) { controller.abort(); }
        setTimeout(function () { root.hidden = true; }, 160);
        if (lastFocus && lastFocus.focus && lastFocus !== input) {
            try { lastFocus.blur(); } catch (e) { /* ignore */ }
        }
    }

    // Input handling. While an IME composition (pinyin…) is in progress the
    // field holds half-typed latin letters — wait for the committed text.
    input.addEventListener('compositionstart', function () { composing = true; });
    input.addEventListener('compositionend', function () { composing = false; schedule(); });
    input.addEventListener('input', function () { if (!composing) { schedule(); } });

    input.addEventListener('keydown', function (e) {
        if (composing || e.isComposing) { return; }
        if (e.key === 'ArrowDown') { e.preventDefault(); setActive(active + 1, true); }
        else if (e.key === 'ArrowUp') { e.preventDefault(); setActive(active - 1, true); }
        else if (e.key === 'Enter') {
            e.preventDefault();
            var q = input.value.trim();
            var opts = options();
            remember(q);
            if (active >= 0 && opts[active]) { location.href = opts[active].href; }
            else if (q) { location.href = '/?keyword=' + encodeURIComponent(q); }
        }
    });

    root.addEventListener('keydown', function (e) {
        if (e.key === 'Escape') { e.preventDefault(); close(); }
        else if (e.key === 'Tab') {
            // a dialog with one text field: keep Tab inside the panel
            var focusables = root.querySelectorAll('input, a[href], button');
            var first = focusables[0];
            var last = focusables[focusables.length - 1];
            if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
            else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
        }
    });

    root.addEventListener('click', function (e) {
        var t = e.target;
        if (t.closest('[data-close]')) { close(); return; }
        var chip = t.closest('.palette-recent');
        if (chip) {
            input.value = chip.textContent;
            input.focus();
            search();
            return;
        }
        if (t.closest('.palette-clear')) {
            G.storageSet(RECENT_KEY, '[]');
            renderHome(hotCache);
            return;
        }
        var item = t.closest('.palette-item');
        if (item) { remember(input.value); }
    });

    results.addEventListener('mousemove', function (e) {
        var item = e.target.closest('[role="option"]');
        if (!item) { return; }
        var idx = Array.prototype.indexOf.call(options(), item);
        if (idx !== active) { setActive(idx, false); }
    });

    window.goblogSearch = {open: open, close: close};
})();
