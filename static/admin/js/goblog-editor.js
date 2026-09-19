/* goblog admin: the Markdown editor page (posts/add, pages/add).

   - sets up editor.md for the screen at hand: split preview on wide screens, a
     single pane with a compact toolbar on phones, always as tall as the window
   - keeps a local backup in localStorage while typing, so a closed tab, an
     expired login or a failed save never costs the text; offers to restore it
     (not to be confused with drafts, which are unpublished posts on the server)
   - warns before leaving with unsaved changes, saves with Ctrl/Cmd+S
   - live word count, description length, slug check, tag suggestions */
(function () {
    'use strict';

    var form = document.getElementById('editor-form');
    if (!form || typeof window.editormd !== 'function') { return; }

    var kind = form.getAttribute('data-kind') || 'post';
    var docId = form.getAttribute('data-id') || '';
    var serverDraft = form.getAttribute('data-draft') || ''; // an unpublished post kept on the server
    var DRAFT_KEY = 'goblog:draft:' + kind + ':' + (docId || (serverDraft ? 'draft-' + serverDraft : 'new'));
    var FIELDS = ['title', 'identity', 'page_id', 'category', 'tags', 'description'];
    var statusEl = document.getElementById('editor-status');
    var countEl = document.getElementById('editor-count');
    var banner = document.getElementById('draft-banner');
    var narrow = window.matchMedia('(max-width: 899px)');
    var editor = null;
    var ready = false;
    var dirty = false;
    var submitting = false;
    var saveTimer = null;

    function field(name) { return form.elements[name] || null; }

    function storage(action, value) {
        try {
            if (action === 'get') { return JSON.parse(localStorage.getItem(DRAFT_KEY) || 'null'); }
            if (action === 'set') { localStorage.setItem(DRAFT_KEY, JSON.stringify(value)); return true; }
            localStorage.removeItem(DRAFT_KEY);
        } catch (e) { /* private mode or full storage: autosave is a bonus */ }
        return null;
    }

    function markdown() {
        if (editor && ready) { return editor.getMarkdown(); }
        var ta = form.querySelector('textarea[name="content"]');
        return ta ? ta.value : '';
    }

    function snapshot() {
        var data = {content: markdown()};
        FIELDS.forEach(function (name) {
            var el = field(name);
            if (el) { data[name] = el.value; }
        });
        return data;
    }

    function same(a, b) {
        if (!a || !b) { return false; }
        return ['content'].concat(FIELDS).every(function (key) {
            return (a[key] || '').replace(/\r\n/g, '\n').trim() === (b[key] || '').replace(/\r\n/g, '\n').trim();
        });
    }

    function setStatus(text, tone) {
        if (!statusEl) { return; }
        statusEl.textContent = text;
        statusEl.setAttribute('data-tone', tone || '');
    }

    function clock(ts) {
        var d = new Date(ts);
        function two(n) { return (n < 10 ? '0' : '') + n; }
        var today = new Date().toDateString() === d.toDateString();
        return (today ? '' : (d.getMonth() + 1) + '月' + d.getDate() + '日 ') + two(d.getHours()) + ':' + two(d.getMinutes());
    }

    /* ---------- word count ---------- */
    function countWords(text) {
        var cjk = (text.match(/[\u3400-\u9fff\uf900-\ufaff]/g) || []).length;
        var words = (text.replace(/[\u3400-\u9fff\uf900-\ufaff]/g, ' ').match(/[A-Za-z0-9_\u00c0-\u024f]+/g) || []).length;
        return cjk + words;
    }

    function updateCount() {
        if (!countEl) { return; }
        var n = countWords(markdown());
        countEl.textContent = n + ' 字 · 约 ' + Math.max(1, Math.round(n / 400)) + ' 分钟读完';
    }

    /* ---------- local draft ---------- */
    // What is stored on the server. Stays null when the page shows a rejected
    // save: the form then holds text that exists nowhere else.
    var original = null;
    var rejected = !!form.querySelector('.editor-alert.alert-danger');

    function saveDraft() {
        if (submitting) { return; }
        var now = snapshot();
        if (same(now, original)) {
            storage('remove');
            dirty = false;
            setStatus('', '');
            return;
        }
        now.savedAt = Date.now();
        if (storage('set', now)) { setStatus('已在本机备份 ' + clock(now.savedAt), 'ok'); }
    }

    function touch() {
        if (!ready) { return; }
        dirty = true;
        setStatus('正在输入…', '');
        clearTimeout(saveTimer);
        saveTimer = setTimeout(function () { saveDraft(); updateCount(); }, 1200);
    }

    function offerDraft() {
        var draft = storage('get');
        if (!draft || !banner) { return; }
        if (same(draft, original)) { storage('remove'); return; }
        var text = banner.querySelector('[data-role="text"]');
        if (text) {
            text.textContent = '这台设备上有一份没保存的本机备份（' + clock(draft.savedAt || Date.now()) + '，' + countWords(draft.content || '') + ' 字）。';
        }
        banner.hidden = false;
        banner.querySelector('[data-role="restore"]').addEventListener('click', function () {
            FIELDS.forEach(function (name) {
                var el = field(name);
                if (el && typeof draft[name] === 'string') { el.value = draft[name]; }
            });
            editor.setMarkdown(draft.content || '');
            banner.hidden = true;
            dirty = true;
            refreshFieldHints();
            updateCount();
            setStatus('已恢复，记得保存', 'ok');
        });
        banner.querySelector('[data-role="discard"]').addEventListener('click', function () {
            storage('remove');
            banner.hidden = true;
        });
    }

    /* ---------- field helpers ---------- */
    var slugTouched = false;

    function refreshFieldHints() {
        var desc = field('description');
        var descCount = document.getElementById('description-count');
        if (desc && descCount) {
            var n = desc.value.trim().length;
            descCount.textContent = n ? n + ' 字' + (n > 160 ? '（搜索结果里大约只显示前 160 字）' : '') : '留空时自动取正文开头';
        }

        var slug = field('identity') || field('page_id');
        var hint = document.getElementById('slug-hint');
        if (slug && hint) {
            var value = slug.value.trim();
            var initial = slug.getAttribute('data-initial') || '';
            var taken = [];
            try { taken = JSON.parse(form.getAttribute('data-slugs') || '[]'); } catch (e) { /* none */ }
            var problem = '';
            if (!value) {
                problem = '必填：它就是文章地址的最后一段';
            } else if (value !== initial) {
                if (!/^[a-z0-9][a-z0-9._-]*$/.test(value)) { problem = '只能用小写字母、数字、- _ . ，并以字母或数字开头'; }
                else if (value.length > 100) { problem = '太长了（最多 100 个字符）'; }
                else if (taken.indexOf(value) !== -1) { problem = '这个地址已经被另一篇占用了'; }
            }
            hint.textContent = problem || (slug.getAttribute('data-prefix') || '/posts/') + (value || '…');
            // an untouched empty field is a hint, not yet a mistake
            hint.setAttribute('data-tone', problem && (value || slugTouched) ? 'error' : '');
            slug.setCustomValidity(problem);
        }
    }

    function initTags() {
        var input = field('tags');
        var box = document.getElementById('tag-suggestions');
        if (!input || !box) { return; }
        var all = [];
        try { all = JSON.parse(box.getAttribute('data-tags') || '[]'); } catch (e) { /* none */ }
        if (!all.length) { box.hidden = true; return; }

        function current() {
            return input.value.split(/[,，]/).map(function (t) { return t.trim(); }).filter(Boolean);
        }
        function render() {
            var chosen = current();
            box.innerHTML = '';
            all.forEach(function (name) {
                var btn = document.createElement('button');
                btn.type = 'button';
                btn.className = 'tag-chip' + (chosen.indexOf(name) !== -1 ? ' is-on' : '');
                btn.textContent = name;
                btn.setAttribute('aria-pressed', chosen.indexOf(name) !== -1 ? 'true' : 'false');
                btn.addEventListener('click', function () {
                    var list = current();
                    var at = list.indexOf(name);
                    if (at === -1) { list.push(name); } else { list.splice(at, 1); }
                    input.value = list.join(',');
                    render();
                    touch();
                });
                box.appendChild(btn);
            });
        }
        input.addEventListener('input', render);
        render();
    }

    /* ---------- editor.md ---------- */
    function editorHeight() {
        var bar = document.querySelector('.editor-bar');
        return Math.max(360, window.innerHeight - (bar ? bar.offsetHeight : 56) - 12);
    }

    var COMPACT_TOOLBAR = ['undo', 'redo', '|', 'bold', 'italic', 'quote', '|', 'h2', 'h3', '|', 'list-ul', 'list-ol', '|',
        'link', 'image', 'code', 'code-block', 'table', '|', 'preview', 'fullscreen'];

    function createEditor() {
        var small = narrow.matches;
        var options = {
            width: '100%',
            height: editorHeight(),
            path: form.getAttribute('data-lib'),
            emoji: true,
            taskList: true,
            watch: !small,
            lineNumbers: !small,
            autoFocus: false,
            syncScrolling: 'single',
            toolbarAutoFixed: false,
            placeholder: '用 Markdown 写点什么……',
            htmlDecode: 'style,script,iframe|on*',
            onload: function () {
                ready = true;
                if (rejected) {
                    dirty = true;
                    saveDraft();
                } else {
                    original = snapshot();
                    offerDraft();
                }
                this.cm.on('change', touch);
                updateCount();
            }
        };
        if (small) { options.toolbarIcons = function () { return COMPACT_TOOLBAR; }; }
        return window.editormd('editor-md', options);
    }

    editor = createEditor();

    var resizeTimer = null;
    window.addEventListener('resize', function () {
        clearTimeout(resizeTimer);
        resizeTimer = setTimeout(function () {
            if (editor && ready && !document.querySelector('.editormd-fullscreen')) { editor.resize('100%', editorHeight()); }
        }, 150);
    });

    /* ---------- wiring ---------- */
    form.addEventListener('input', function (e) {
        if (!e.target || !e.target.name || e.target.name === 'content') { return; }
        if (e.target.name === 'identity' || e.target.name === 'page_id') { slugTouched = true; }
        touch();
        refreshFieldHints();
    });
    form.addEventListener('change', function (e) {
        if (e.target && e.target.name) { touch(); }
    });

    form.addEventListener('invalid', function (e) {
        // the browser refused to submit: make sure the offending field is visible
        slugTouched = true;
        refreshFieldHints();
        var meta = document.getElementById('editor-meta');
        if (meta && meta.contains(e.target)) { meta.open = true; }
    }, true);

    form.addEventListener('submit', function () {
        slugTouched = true;
        refreshFieldHints();
        if (!form.checkValidity()) { return; }
        // keep the draft until the list page confirms the save went through
        clearTimeout(saveTimer);
        var last = snapshot();
        last.savedAt = Date.now();
        last.submitted = true;
        storage('set', last);
        submitting = true;
        setStatus('正在保存…', '');
    });

    window.addEventListener('beforeunload', function (e) {
        if (!dirty || submitting) { return; }
        saveDraft();
        e.preventDefault();
        e.returnValue = '';
    });

    document.addEventListener('keydown', function (e) {
        if ((e.ctrlKey || e.metaKey) && !e.altKey && (e.key === 's' || e.key === 'S')) {
            e.preventDefault();
            // an unpublished post is saved as a draft: Ctrl+S must never publish by accident
            var button = document.getElementById('save-draft') || form.querySelector('[type="submit"]');
            button.click();
        }
    });

    // back from "save draft": the server has the text now, the local copy has done its job
    var params = new URLSearchParams(location.search);
    var savedSlug = params.get('saved');
    if (savedSlug) {
        try {
            Object.keys(localStorage).forEach(function (key) {
                if (key.indexOf('goblog:draft:') !== 0) { return; }
                var local = JSON.parse(localStorage.getItem(key) || 'null');
                if (local && local.submitted && local.identity === savedSlug) { localStorage.removeItem(key); }
            });
        } catch (e) { /* storage unavailable */ }
        setStatus('草稿已保存到服务器 ' + clock(Date.now()), 'ok');
        params.delete('saved');
        if (history.replaceState) { history.replaceState(null, '', location.pathname + '?' + params.toString()); }
    }

    // the settings are open while they still need to be filled in (a new post)
    // or fixed (a rejected save); an existing post opens straight into its text
    var meta = document.getElementById('editor-meta');
    if (meta && (!docId || rejected)) { meta.open = true; }

    refreshFieldHints();
    initTags();
})();
