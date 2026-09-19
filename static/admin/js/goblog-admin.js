/* goblog admin: behaviour shared by the list and form pages. */
(function () {
    'use strict';

    /* sidebar toggle (what SB Admin's scripts.js did) */
    var toggle = document.getElementById('sidebarToggle');
    if (toggle) {
        toggle.addEventListener('click', function (e) {
            e.preventDefault();
            document.body.classList.toggle('sb-sidenav-toggled');
        });
    }

    /* highlight the current section in the menu */
    var path = location.pathname.replace(/\/+$/, '') || '/admin';
    var links = document.querySelectorAll('#sidenavAccordion a.nav-link');
    var best = null;
    Array.prototype.forEach.call(links, function (a) {
        var target = a.getAttribute('data-nav') || a.getAttribute('href').replace(/\/+$/, '');
        var own = a.getAttribute('href').replace(/\/+$/, '');
        if (path === own || path.indexOf(target + '/') === 0 || path === target) {
            if (!best || target.length > best.len) { best = {el: a, len: target.length}; }
        }
    });
    if (best) {
        best.el.classList.add('active');
        best.el.setAttribute('aria-current', 'page');
    }

    /* tables: search, sorting and paging in Chinese */
    var table = document.getElementById('datatablesSimple');
    if (table && window.simpleDatatables) {
        var rows = table.tBodies.length ? table.tBodies[0].rows.length : 0;
        var perPage = parseInt(table.getAttribute('data-per-page'), 10) || 10;
        new window.simpleDatatables.DataTable(table, {
            perPage: perPage,
            perPageSelect: [10, 20, 50, 100],
            searchable: rows > 5,
            paging: rows > perPage,
            labels: {
                placeholder: '搜索…',
                perPage: '每页 {select} 条',
                noRows: '还没有内容',
                info: '第 {start}–{end} 条，共 {rows} 条'
            }
        });
    }

    /* destructive forms ask first (rows are re-rendered by the table, so delegate) */
    document.addEventListener('submit', function (e) {
        var message = e.target.getAttribute && e.target.getAttribute('data-confirm');
        if (message && !window.confirm(message)) { e.preventDefault(); }
    });

    /* after a successful save the editor's local draft has done its job */
    var saved = new URLSearchParams(location.search).get('saved');
    if (saved) {
        try {
            Object.keys(localStorage).forEach(function (key) {
                if (key.indexOf('goblog:draft:') !== 0) { return; }
                var draft = JSON.parse(localStorage.getItem(key) || 'null');
                if (draft && draft.submitted && (draft.identity === saved || draft.page_id === saved)) { localStorage.removeItem(key); }
            });
        } catch (e) { /* storage unavailable */ }

        var note = document.getElementById('saved-note');
        if (note) { note.hidden = false; }
        if (history.replaceState) { history.replaceState(null, '', location.pathname); }
    }
})();
