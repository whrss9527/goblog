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

    /* "?done=merged&name=…" has been rendered by the server: a reload should not repeat it */
    if (new URLSearchParams(location.search).get('done') && history.replaceState) {
        history.replaceState(null, '', location.pathname);
    }

    /* renaming a tag to a name that exists merges the two: say so before the button is pressed */
    var tagForm = document.getElementById('tag-form');
    if (tagForm) {
        var others = [];
        try { others = JSON.parse(tagForm.getAttribute('data-other-names') || '[]'); } catch (e) { /* keep [] */ }
        var nameInput = document.getElementById('tag-name');
        var hint = document.getElementById('merge-hint');
        var submit = document.getElementById('tag-submit');
        var update = function () {
            var name = nameInput.value.trim();
            var merges = name !== '' && others.indexOf(name) !== -1;
            hint.hidden = !merges;
            hint.textContent = merges ? '已经有一个叫「' + name + '」的标签：保存后两个标签会合并成它。' : '';
            submit.textContent = merges ? '合并' : '保存';
        };
        nameInput.addEventListener('input', update);
        update();
    }
})();
