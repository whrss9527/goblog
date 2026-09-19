/* goblog article features: everything that only makes sense once a Markdown
   body has been rendered (posts and pages). Depends on enhance.js (window.goblog).
   Entry point: window.enhancePost(), called by the renderer bootstrap in
   tpl/default/markdown.html. */
(function () {
    'use strict';

    var G = window.goblog;
    if (!G) { return; }
    var each = G.each;

    /* ---------- Code blocks: language label, copy, collapse ---------- */
    var LANG_NAMES = {
        go: 'Go', golang: 'Go', sh: 'Shell', shell: 'Shell', bash: 'Shell', zsh: 'Shell', cmd: 'CMD', bat: 'CMD',
        powershell: 'PowerShell', yml: 'YAML', yaml: 'YAML', json: 'JSON', sql: 'SQL', java: 'Java',
        js: 'JavaScript', javascript: 'JavaScript', ts: 'TypeScript', typescript: 'TypeScript', py: 'Python',
        python: 'Python', html: 'HTML', xml: 'XML', css: 'CSS', c: 'C', cpp: 'C++', rust: 'Rust', rs: 'Rust',
        php: 'PHP', ruby: 'Ruby', rb: 'Ruby', toml: 'TOML', ini: 'INI', nginx: 'Nginx', dockerfile: 'Dockerfile',
        docker: 'Dockerfile', proto: 'Protobuf', protobuf: 'Protobuf', makefile: 'Makefile', diff: 'Diff',
        md: 'Markdown', markdown: 'Markdown', lua: 'Lua', kotlin: 'Kotlin', swift: 'Swift'
    };
    var COLLAPSE_MIN_LINES = 40;
    var COLLAPSE_SHOW_LINES = 24;

    function codeText(pre) {
        // prettify renders <ol><li> per line; join them so newlines survive.
        var lines = pre.querySelectorAll('ol.linenums > li');
        if (lines.length) {
            return Array.prototype.map.call(lines, function (li) { return li.textContent; }).join('\n');
        }
        var code = pre.querySelector('code');
        return code ? code.textContent : pre.textContent;
    }

    // True when a fenced block without a language holds long, mostly-CJK lines.
    function looksLikeProse(text) {
        return text.split('\n').some(function (line) {
            if (line.length < 40) { return false; }
            var cjk = (line.match(/[\u3400-\u9fff\u3000-\u303f\uff00-\uffef]/g) || []).length;
            return cjk / line.length > 0.3;
        });
    }

    function enhanceCodeBlocks() {
        each(document.querySelectorAll('.article-content pre'), function (pre) {
            if (pre.parentNode.classList.contains('code-block-wrapper')) { return; }
            var wrapper = document.createElement('div');
            wrapper.className = 'code-block-wrapper';
            pre.parentNode.insertBefore(wrapper, pre);
            wrapper.appendChild(pre);

            var code = pre.querySelector('code');
            var match = code && /(?:^|\s)lang(?:uage)?-([\w+#.-]+)/.exec(code.className);
            if (match) {
                var key = match[1].toLowerCase();
                var label = document.createElement('span');
                label.className = 'code-lang';
                label.textContent = LANG_NAMES[key] || key.toUpperCase();
                wrapper.appendChild(label);
                wrapper.classList.add('has-lang');
            } else if (looksLikeProse(codeText(pre))) {
                // Fences are sometimes used as a "note box" for plain Chinese text:
                // wrap those instead of making readers scroll sideways.
                wrapper.classList.add('is-prose');
            }

            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'copy-btn';
            btn.textContent = '复制';
            btn.addEventListener('click', function () {
                G.copyText(codeText(pre)).then(function () {
                    btn.textContent = '已复制';
                    btn.classList.add('copied');
                }, function () {
                    btn.textContent = '复制失败';
                }).then(function () {
                    setTimeout(function () {
                        btn.textContent = '复制';
                        btn.classList.remove('copied');
                    }, 2000);
                });
            });
            wrapper.appendChild(btn);

            // Very long listings start folded so they do not bury the prose.
            var lineCount = pre.querySelectorAll('ol.linenums > li').length || codeText(pre).split('\n').length;
            if (lineCount >= COLLAPSE_MIN_LINES) {
                var first = pre.querySelector('ol.linenums > li');
                var lineHeight = first ? first.getBoundingClientRect().height : 22;
                wrapper.style.setProperty('--collapsed-h', Math.round(lineHeight * COLLAPSE_SHOW_LINES + 28) + 'px');
                wrapper.classList.add('is-collapsible', 'is-collapsed');

                var more = document.createElement('button');
                more.type = 'button';
                more.className = 'code-expand';
                more.setAttribute('aria-expanded', 'false');
                var rest = lineCount - COLLAPSE_SHOW_LINES;
                more.textContent = '展开剩余 ' + rest + ' 行';
                more.addEventListener('click', function () {
                    var collapsed = wrapper.classList.toggle('is-collapsed');
                    more.setAttribute('aria-expanded', collapsed ? 'false' : 'true');
                    more.textContent = collapsed ? '展开剩余 ' + rest + ' 行' : '收起代码';
                    if (collapsed && wrapper.getBoundingClientRect().top < 0) {
                        wrapper.scrollIntoView({block: 'start'});
                    }
                });
                wrapper.appendChild(more);
            }
        });
    }

    /* ---------- Images, tables, links ---------- */
    function addImageLazyLoading() {
        each(document.querySelectorAll('.article-content img'), function (img) {
            img.setAttribute('loading', 'lazy');
            img.setAttribute('decoding', 'async');
        });
    }

    function wrapTables() {
        each(document.querySelectorAll('.article-content table'), function (table) {
            if (table.parentNode.classList.contains('table-wrapper')) { return; }
            var wrapper = document.createElement('div');
            wrapper.className = 'table-wrapper';
            wrapper.setAttribute('tabindex', '0');
            wrapper.setAttribute('role', 'region');
            wrapper.setAttribute('aria-label', '表格（可横向滚动）');
            table.parentNode.insertBefore(wrapper, table);
            wrapper.appendChild(table);
        });
    }

    function markExternalLinks() {
        each(document.querySelectorAll('.article-content a[href]'), function (a) {
            var url;
            try { url = new URL(a.getAttribute('href'), location.href); } catch (e) { return; }
            if (!/^https?:$/.test(url.protocol) || url.origin === location.origin) { return; }
            a.target = '_blank';
            a.rel = 'noopener noreferrer';
        });
    }

    /* ---------- Headings: stable ids + "copy link to section" ---------- */
    function slugify(text) {
        return text.trim().toLowerCase()
            .replace(/\s+/g, '-')
            .replace(/[^\w\u00c0-\u024f\u3040-\u30ff\u3400-\u9fff\uac00-\ud7af-]+/g, '')
            .replace(/-{2,}/g, '-')
            .replace(/^-+|-+$/g, '');
    }

    function prepareHeadings(viewer) {
        var headings = viewer.querySelectorAll('h1, h2, h3, h4, h5');
        var seen = {};
        each(headings, function (heading, i) {
            var base = slugify(heading.textContent) || ('section-' + (i + 1));
            var id = base;
            var n = 2;
            while (seen[id] || (document.getElementById(id) && document.getElementById(id) !== heading)) {
                id = base + '-' + n++;
            }
            seen[id] = true;
            heading.id = id;

            var anchor = document.createElement('a');
            anchor.className = 'heading-anchor';
            anchor.href = '#' + id;
            anchor.setAttribute('aria-label', '复制本节链接');
            anchor.innerHTML = G.icons.link;
            anchor.addEventListener('click', function (e) {
                e.preventDefault();
                var url = location.origin + location.pathname + location.search + '#' + encodeURIComponent(id);
                if (history.replaceState) { history.replaceState(null, '', '#' + encodeURIComponent(id)); }
                G.copyText(url).then(function () { G.toast('已复制本节链接'); }, function () { G.toast('复制失败，请手动复制地址栏'); });
            });
            heading.appendChild(anchor);
        });
        return headings;
    }

    // The body is rendered by JS, so the browser's own jump-to-hash ran too
    // early. Supports the new ids and editor.md's legacy <a name="..."> anchors.
    function scrollToHash() {
        if (!location.hash || location.hash.length < 2) { return false; }
        var raw = location.hash.slice(1);
        var name = raw;
        try { name = decodeURIComponent(raw); } catch (e) { /* keep raw */ }
        var target = document.getElementById(name);
        if (!target) {
            var legacy = document.querySelectorAll('.article-content a[name]');
            for (var i = 0; i < legacy.length; i++) {
                if (legacy[i].getAttribute('name') === name) {
                    target = legacy[i].parentNode;
                    break;
                }
            }
        }
        if (!target || !target.closest('.article-main, .article-content, #comments')) { return false; }
        target.scrollIntoView({block: 'start'});
        return true;
    }

    /* ---------- Table of contents ---------- */
    function buildTOC(headings) {
        // Indent by rank of the heading levels actually used, so an article that
        // jumps from h2 straight to h4 still gets a tidy two-level outline.
        var used = [];
        each(headings, function (h) {
            var level = parseInt(h.tagName.charAt(1), 10);
            if (used.indexOf(level) < 0) { used.push(level); }
        });
        used.sort();

        var toc = document.createElement('nav');
        toc.className = 'toc-container';
        toc.setAttribute('aria-label', '文章目录');
        var handle = document.createElement('div');
        handle.className = 'toc-sheet-handle';
        toc.appendChild(handle);
        var title = document.createElement('div');
        title.className = 'toc-title';
        title.textContent = '目录';
        var progress = document.createElement('span');
        progress.className = 'toc-progress';
        title.appendChild(progress);
        toc.appendChild(title);

        var ul = document.createElement('ul');
        each(headings, function (heading) {
            var level = Math.min(4, used.indexOf(parseInt(heading.tagName.charAt(1), 10)));
            var li = document.createElement('li');
            li.className = 'toc-level-' + level;
            var a = document.createElement('a');
            a.href = '#' + heading.id;
            a.textContent = heading.textContent;
            li.appendChild(a);
            ul.appendChild(li);
        });
        toc.appendChild(ul);

        var layout = document.getElementById('article-layout');
        var aside = document.getElementById('article-aside');
        var overlay = document.createElement('div');
        overlay.className = 'toc-overlay';
        document.body.appendChild(overlay);

        var toggleBtn = document.createElement('button');
        toggleBtn.type = 'button';
        toggleBtn.className = 'fab fab-toc';
        toggleBtn.setAttribute('aria-label', '文章目录');
        toggleBtn.setAttribute('aria-expanded', 'false');
        toggleBtn.innerHTML = G.icons.list;
        var stack = G.fabStack();
        stack.insertBefore(toggleBtn, stack.firstChild);

        function closeSheet() {
            toc.classList.remove('mobile-visible');
            overlay.classList.remove('visible');
            toggleBtn.setAttribute('aria-expanded', 'false');
            document.documentElement.style.overflow = '';
        }

        function openSheet() {
            toc.classList.add('mobile-visible');
            overlay.classList.add('visible');
            toggleBtn.setAttribute('aria-expanded', 'true');
            document.documentElement.style.overflow = 'hidden';
            var active = toc.querySelector('a.active');
            if (active) { active.scrollIntoView({block: 'center'}); }
        }

        // >= 1100px the TOC lives in the sticky sidebar, below it is a bottom sheet.
        var wide = window.matchMedia('(min-width: 1100px)');
        function place() {
            closeSheet();
            if (wide.matches && aside) {
                toc.classList.remove('is-sheet');
                aside.appendChild(toc);
            } else {
                toc.classList.add('is-sheet');
                document.body.appendChild(toc);
            }
        }

        if (layout) { layout.classList.add('has-toc'); }
        place();
        if (wide.addEventListener) { wide.addEventListener('change', place); }
        else if (wide.addListener) { wide.addListener(place); }

        toggleBtn.addEventListener('click', function () {
            if (toc.classList.contains('mobile-visible')) { closeSheet(); } else { openSheet(); }
        });
        overlay.addEventListener('click', closeSheet);
        document.addEventListener('keydown', function (e) {
            if (e.key === 'Escape') { closeSheet(); }
        });

        var tocLinks = toc.querySelectorAll('a');
        toc.addEventListener('click', function (e) {
            var link = e.target.closest ? e.target.closest('a') : null;
            if (!link) { return; }
            e.preventDefault();
            closeSheet();
            var target = document.getElementById(decodeURIComponent(link.getAttribute('href').slice(1)));
            if (target) {
                target.scrollIntoView({behavior: 'smooth', block: 'start'});
                if (history.replaceState) { history.replaceState(null, '', '#' + encodeURIComponent(target.id)); }
            }
        });

        // Scroll spy + reading percentage
        var article = document.querySelector('.article-content');
        var lastActive = null;
        function spy() {
            var current = headings[0].id;
            each(headings, function (heading) {
                if (heading.getBoundingClientRect().top <= 110) { current = heading.id; }
            });
            if (article) {
                var rect = article.getBoundingClientRect();
                var total = rect.height - window.innerHeight * 0.6;
                var ratio = total > 0 ? Math.min(1, Math.max(0, (110 - rect.top) / total)) : 1;
                progress.textContent = Math.round(ratio * 100) + '%';
            }
            if (current === lastActive) { return; }
            lastActive = current;
            each(tocLinks, function (link) {
                var on = decodeURIComponent(link.getAttribute('href').slice(1)) === current;
                link.classList.toggle('active', on);
                if (on && !toc.classList.contains('is-sheet') && toc.scrollHeight > toc.clientHeight) {
                    // keep the active entry visible inside a long sidebar TOC
                    toc.scrollTop = Math.max(0, link.offsetTop - toc.clientHeight / 2);
                }
            });
        }
        G.onScroll(spy);
        spy();
    }

    /* ---------- Image lightbox ---------- */
    function initLightbox() {
        var images = Array.prototype.filter.call(
            document.querySelectorAll('.article-content img:not(.emoji)'),
            function (img) { return !img.closest('a'); }
        );
        if (!images.length) { return; }

        var box = document.createElement('div');
        box.className = 'lightbox';
        box.setAttribute('role', 'dialog');
        box.setAttribute('aria-modal', 'true');
        box.setAttribute('aria-label', '图片查看器');
        box.hidden = true;
        box.innerHTML =
            '<button type="button" class="lightbox-btn lightbox-close" aria-label="关闭">' + G.icons.close + '</button>' +
            '<button type="button" class="lightbox-btn lightbox-prev" aria-label="上一张">' + G.icons.left + '</button>' +
            '<button type="button" class="lightbox-btn lightbox-next" aria-label="下一张">' + G.icons.right + '</button>' +
            '<div class="lightbox-stage"><img alt=""></div>' +
            '<div class="lightbox-caption"><span class="lightbox-counter"></span><span class="lightbox-alt"></span></div>';
        document.body.appendChild(box);

        var stage = box.querySelector('.lightbox-stage');
        var view = stage.querySelector('img');
        var counter = box.querySelector('.lightbox-counter');
        var altEl = box.querySelector('.lightbox-alt');
        var prevBtn = box.querySelector('.lightbox-prev');
        var nextBtn = box.querySelector('.lightbox-next');
        var closeBtn = box.querySelector('.lightbox-close');
        var index = 0;
        var lastFocus = null;

        function meaningfulAlt(alt) {
            return alt && !/^[\w\s.()\-一-鿿]*\.(png|jpe?g|gif|webp|bmp|svg)$/i.test(alt) ? alt : '';
        }

        function show(i) {
            index = (i + images.length) % images.length;
            var src = images[index];
            box.classList.remove('is-zoomed');
            view.src = src.currentSrc || src.src;
            view.alt = src.alt || '';
            counter.textContent = images.length > 1 ? (index + 1) + ' / ' + images.length : '';
            altEl.textContent = meaningfulAlt(src.alt);
            prevBtn.hidden = nextBtn.hidden = images.length < 2;
        }

        function open(i) {
            lastFocus = document.activeElement;
            show(i);
            box.hidden = false;
            document.documentElement.style.overflow = 'hidden';
            void box.offsetWidth;
            box.classList.add('is-open');
            closeBtn.focus();
        }

        function close() {
            box.classList.remove('is-open', 'is-zoomed');
            document.documentElement.style.overflow = '';
            setTimeout(function () { box.hidden = true; view.removeAttribute('src'); }, 200);
            if (lastFocus && lastFocus.focus) { lastFocus.focus({preventScroll: true}); }
        }

        each(images, function (img, i) {
            img.classList.add('is-zoomable');
            img.setAttribute('tabindex', '0');
            img.setAttribute('role', 'button');
            img.setAttribute('aria-label', (img.alt ? img.alt + '，' : '') + '点击放大');
            img.addEventListener('click', function () { open(i); });
            img.addEventListener('keydown', function (e) {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    open(i);
                }
            });
        });

        closeBtn.addEventListener('click', close);
        prevBtn.addEventListener('click', function (e) { e.stopPropagation(); show(index - 1); });
        nextBtn.addEventListener('click', function (e) { e.stopPropagation(); show(index + 1); });
        stage.addEventListener('click', function (e) {
            if (e.target !== view) { close(); return; }
            // Toggle between "fit to screen" and natural size when that reveals more detail.
            if (box.classList.contains('is-zoomed')) {
                box.classList.remove('is-zoomed');
            } else if (view.naturalWidth > view.clientWidth * 1.15) {
                box.classList.add('is-zoomed');
                stage.scrollLeft = (stage.scrollWidth - stage.clientWidth) / 2;
                stage.scrollTop = (stage.scrollHeight - stage.clientHeight) / 2;
            } else {
                close();
            }
        });

        document.addEventListener('keydown', function (e) {
            if (box.hidden) { return; }
            if (e.key === 'Escape') { close(); }
            else if (e.key === 'ArrowLeft') { show(index - 1); }
            else if (e.key === 'ArrowRight') { show(index + 1); }
            else if (e.key === 'Tab') {
                // keep focus inside the dialog
                var focusables = box.querySelectorAll('button:not([hidden])');
                var first = focusables[0];
                var last = focusables[focusables.length - 1];
                if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
                else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
            }
        });

        // Swipe: left/right switches image, down closes. Pinch is left to the browser.
        var startX = 0;
        var startY = 0;
        var tracking = false;
        stage.addEventListener('touchstart', function (e) {
            tracking = e.touches.length === 1 && !box.classList.contains('is-zoomed');
            if (tracking) {
                startX = e.touches[0].clientX;
                startY = e.touches[0].clientY;
            }
        }, {passive: true});
        stage.addEventListener('touchend', function (e) {
            if (!tracking || (window.visualViewport && window.visualViewport.scale > 1.05)) { return; }
            tracking = false;
            var dx = e.changedTouches[0].clientX - startX;
            var dy = e.changedTouches[0].clientY - startY;
            if (Math.abs(dx) > 60 && Math.abs(dx) > Math.abs(dy) * 1.5 && images.length > 1) {
                show(index + (dx < 0 ? 1 : -1));
            } else if (dy > 90 && Math.abs(dy) > Math.abs(dx) * 1.5) {
                close();
            }
        }, {passive: true});
    }

    /* ---------- Share / copy link ---------- */
    function initShare() {
        var permalink = document.getElementById('post-permalink');
        var url = permalink ? permalink.href : location.href.split('#')[0];
        var shareBtn = document.getElementById('share-btn');
        var copyBtn = document.getElementById('copy-link-btn');

        function copy() {
            G.copyText(url).then(function () { G.toast('链接已复制'); }, function () { G.toast('复制失败，请手动复制地址栏'); });
        }

        if (copyBtn) { copyBtn.addEventListener('click', copy); }
        if (shareBtn) {
            if (!navigator.share) {
                shareBtn.hidden = true;
            } else {
                shareBtn.addEventListener('click', function () {
                    navigator.share({title: shareBtn.getAttribute('data-title') || document.title, url: url})
                        .catch(function () { /* dismissed */ });
                });
            }
        }
    }

    /* ---------- Like button ---------- */
    function initLike() {
        var btn = document.getElementById('like-btn');
        if (!btn) { return; }
        var countEl = document.getElementById('like-count');
        var labelEl = btn.querySelector('.like-label');
        var identity = btn.getAttribute('data-identity');
        var busy = false;

        function setLiked(count) {
            btn.classList.add('is-liked');
            btn.setAttribute('aria-pressed', 'true');
            if (labelEl) { labelEl.textContent = '已赞'; }
            if (countEl && typeof count === 'number') { countEl.textContent = count; }
        }

        btn.addEventListener('click', function () {
            if (busy) { return; }
            if (btn.classList.contains('is-liked')) {
                G.toast('已经赞过啦，谢谢你 ❤');
                return;
            }
            busy = true;
            var before = parseInt(countEl ? countEl.textContent : '0', 10) || 0;
            setLiked(before + 1); // optimistic
            if (G.burst) { G.burst(btn, ['❤', '💚', '✨', '👍']); }

            fetch('/api/posts/' + encodeURIComponent(identity) + '/like', {
                method: 'POST',
                credentials: 'same-origin',
                headers: {'X-Requested-With': 'goblog', Accept: 'application/json'}
            }).then(function (res) {
                if (res.status === 429) { throw new Error('rate'); }
                if (!res.ok) { throw new Error('http'); }
                return res.json();
            }).then(function (data) {
                setLiked(data.likes);
            }).catch(function (err) {
                // roll back the optimistic update
                btn.classList.remove('is-liked');
                btn.setAttribute('aria-pressed', 'false');
                if (labelEl) { labelEl.textContent = '点赞'; }
                if (countEl) { countEl.textContent = before; }
                G.toast(err && err.message === 'rate' ? '手速太快了，歇一会儿再点吧' : '点赞没有成功，稍后再试试');
            }).then(function () { busy = false; });
        });
    }

    /* ---------- Remember where the reader stopped ---------- */
    var POS_KEY = 'readpos';
    var POS_MAX_AGE = 30 * 24 * 3600 * 1000;
    var POS_MAX_ENTRIES = 30;

    function loadPositions() {
        try { return JSON.parse(G.storageGet(POS_KEY) || '{}') || {}; } catch (e) { return {}; }
    }

    function initReadingPosition(offerResume) {
        if (!document.getElementById('post-viewer')) { return; }
        var key = location.pathname;
        var positions = loadPositions();
        var saved = positions[key];

        if (offerResume && saved && Date.now() - saved.t < POS_MAX_AGE && saved.r > 0.12 && saved.r < 0.92) {
            var percent = Math.round(saved.r * 100);
            setTimeout(function () {
                if (window.scrollY > 300) { return; } // the reader already moved on
                G.toast('上次读到 ' + percent + '%', {
                    action: '继续阅读',
                    duration: 9000,
                    onAction: function () {
                        var max = document.documentElement.scrollHeight - window.innerHeight;
                        window.scrollTo({top: Math.round(max * saved.r), behavior: 'smooth'});
                    }
                });
            }, 900);
        }

        var lastSave = 0;
        function save() {
            var now = Date.now();
            if (now - lastSave < 1000) { return; }
            lastSave = now;
            var all = loadPositions();
            var r = G.readingRatio();
            if (r > 0.95) {
                delete all[key]; // finished: nothing to resume
            } else if (r > 0.05) {
                all[key] = {r: Math.round(r * 1000) / 1000, t: now};
            }
            var keys = Object.keys(all);
            if (keys.length > POS_MAX_ENTRIES) {
                keys.sort(function (a, b) { return all[a].t - all[b].t; });
                keys.slice(0, keys.length - POS_MAX_ENTRIES).forEach(function (k) { delete all[k]; });
            }
            G.storageSet(POS_KEY, JSON.stringify(all));
        }
        G.onScroll(save);
        window.addEventListener('pagehide', function () { lastSave = 0; save(); });
    }

    /* ---------- Entry point ---------- */
    window.enhancePost = function () {
        enhanceCodeBlocks();
        addImageLazyLoading();
        wrapTables();
        markExternalLinks();
        initLightbox();
        initShare();
        initLike();

        var viewer = document.getElementById('post-viewer') || document.getElementById('page-viewer');
        if (viewer) {
            var headings = prepareHeadings(viewer);
            if (viewer.id === 'post-viewer' && headings.length >= 2) { buildTOC(headings); }
        }
        initReadingPosition(!scrollToHash());
    };
})();
