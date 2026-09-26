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

    var getJSON = function (url) {
        return fetch(url, {credentials: 'same-origin', headers: {'Accept': 'application/json'}})
            .then(function (res) { return res.json(); });
    };

    /* project list: the owner's recent GitHub repositories that are not projects yet */
    var suggestions = document.getElementById('github-suggestions');
    if (suggestions && window.fetch) {
        var suggestionList = document.getElementById('suggestion-list');
        var suggestionStatus = document.getElementById('suggestion-status');
        getJSON('/admin/projects/github/suggestions').then(function (data) {
            if (data.error) { suggestionStatus.textContent = data.error; return; }
            if (!data.repos || !data.repos.length) {
                suggestionStatus.textContent = '最近的公开仓库都已经加进来了。';
                return;
            }
            suggestionList.removeChild(suggestionStatus);
            data.repos.forEach(function (repo) {
                var item = document.createElement('li');
                item.className = 'list-group-item admin-draft';
                var text = document.createElement('div');
                text.className = 'admin-draft-text';
                var title = document.createElement('a');
                title.className = 'admin-title';
                title.href = repo.url;
                title.target = '_blank';
                title.rel = 'noopener';
                title.textContent = repo.name;
                var sub = document.createElement('span');
                sub.className = 'admin-sub';
                sub.textContent = [repo.description, repo.language, repo.stars ? '★ ' + repo.stars : '',
                    repo.pushed_at ? '最近提交 ' + repo.pushed_at.slice(0, 10) : ''].filter(Boolean).join(' · ');
                text.appendChild(title);
                text.appendChild(sub);
                var add = document.createElement('a');
                add.className = 'btn btn-sm btn-outline-primary';
                add.href = '/admin/projects/add?repo=' + encodeURIComponent(repo.url);
                add.textContent = '添加';
                item.appendChild(text);
                item.appendChild(add);
                suggestionList.appendChild(item);
            });
        }).catch(function () {
            suggestionStatus.textContent = '读取 GitHub 仓库失败，稍后刷新页面再试。';
        });
    }

    /* project form: fill in the details of a GitHub repository (only fields that are still empty) */
    var projectForm = document.getElementById('project-form');
    if (projectForm && window.fetch) {
        var field = function (id) { return document.getElementById(id); };
        var repoInput = field('project-repo');
        var importButton = field('github-import');
        var importStatus = field('import-status');
        var fillEmpty = function (id, value) {
            var el = field(id);
            if (el && value && !el.value.trim()) {
                el.value = value;
                return true;
            }
            return false;
        };
        var runImport = function () {
            var repo = repoInput.value.trim();
            if (!repo) {
                repoInput.focus();
                return;
            }
            importButton.disabled = true;
            importStatus.textContent = '正在读取…';
            getJSON('/admin/projects/github?repo=' + encodeURIComponent(repo)).then(function (data) {
                if (data.error) {
                    importStatus.textContent = data.error;
                    return;
                }
                repoInput.value = data.repo;
                var filled = 0;
                [['project-name', data.name], ['project-description', data.description], ['project-url', data.url],
                    ['project-tech', (data.tech || []).join(', ')], ['project-started', data.started]].forEach(function (pair) {
                    if (fillEmpty(pair[0], pair[1])) { filled++; }
                });
                if (!projectForm.querySelector('[name="id"]').value && data.status) {
                    field('project-status').value = String(data.status);
                }
                var notes = ['已读取 ' + data.repo.replace(/^https:\/\/github\.com\//, '') + (data.stars ? '（★ ' + data.stars + '）' : '') +
                    (filled ? '，填入了 ' + filled + ' 项。' : '，要填的项都已经有内容了。')];
                if (data.existing) { notes.push('注意：这个仓库已经在项目「' + data.existing + '」里了。'); }
                if (data['private']) { notes.push('这是私有仓库：访客打不开源码链接，建议把源码地址留空。'); }
                importStatus.textContent = notes.join(' ');
            }).catch(function () {
                importStatus.textContent = '读取失败，请稍后再试（也可以直接手动填写）。';
            }).then(function () {
                importButton.disabled = false;
            });
        };
        importButton.addEventListener('click', runImport);
        repoInput.addEventListener('keydown', function (e) {
            if (e.key === 'Enter' && !field('project-name').value.trim()) { // nothing to save yet: Enter reads the repository
                e.preventDefault();
                runImport();
            }
        });
        if (projectForm.getAttribute('data-import') === '1') { runImport(); }

        var cover = field('project-cover');
        var coverPreview = field('project-cover-preview');
        cover.addEventListener('input', function () {
            var url = cover.value.trim();
            if (/^(https?:\/\/|\/[^/])/.test(url)) {
                coverPreview.src = url;
                coverPreview.hidden = false;
            } else {
                coverPreview.hidden = true;
            }
        });
    }
})();
