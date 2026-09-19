/* goblog front-end enhancements (vanilla JS, no dependencies). */
(function () {
    'use strict';

    var root = document.documentElement;

    /* ---------- Theme ---------- */
    // The initial theme is applied by an inline script in <head> (no flash).
    function currentTheme() {
        return root.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
    }

    function savedTheme() {
        try {
            var t = localStorage.getItem('theme');
            return (t === 'dark' || t === 'light') ? t : null;
        } catch (e) {
            return null;
        }
    }

    function syncToggleState() {
        var toggle = document.getElementById('dark-toggle');
        if (toggle) { toggle.setAttribute('aria-checked', currentTheme() === 'dark' ? 'true' : 'false'); }
    }

    function applyTheme(theme, persist) {
        root.setAttribute('data-theme', theme);
        syncToggleState();
        if (persist) {
            try { localStorage.setItem('theme', theme); } catch (e) { /* ignore */ }
        }
        var giscusFrame = document.querySelector('iframe.giscus-frame');
        if (giscusFrame && giscusFrame.contentWindow) {
            giscusFrame.contentWindow.postMessage({
                giscus: {setConfig: {theme: theme === 'dark' ? 'dark' : 'light'}}
            }, 'https://giscus.app');
        }
    }

    function initThemeToggle() {
        var toggle = document.getElementById('dark-toggle');
        if (toggle) {
            syncToggleState();
            toggle.addEventListener('click', function () {
                applyTheme(currentTheme() === 'dark' ? 'light' : 'dark', true);
            });
        }
        // Follow OS changes live as long as the visitor never chose explicitly.
        if (window.matchMedia) {
            var mq = window.matchMedia('(prefers-color-scheme: dark)');
            var onChange = function (e) {
                if (!savedTheme()) { applyTheme(e.matches ? 'dark' : 'light', false); }
            };
            if (mq.addEventListener) { mq.addEventListener('change', onChange); }
            else if (mq.addListener) { mq.addListener(onChange); }
        }
    }

    /* ---------- Header: mobile menu, shadow, hide-on-scroll ---------- */
    function initHeader() {
        var header = document.getElementById('site-header');
        var toggle = document.getElementById('nav-toggle');
        var panel = document.getElementById('nav-panel');
        if (!header) { return; }

        function setOpen(open) {
            header.classList.toggle('nav-open', open);
            if (toggle) {
                toggle.setAttribute('aria-expanded', open ? 'true' : 'false');
                toggle.setAttribute('aria-label', open ? '关闭菜单' : '菜单');
            }
        }

        if (toggle && panel) {
            toggle.addEventListener('click', function () {
                setOpen(!header.classList.contains('nav-open'));
            });
            document.addEventListener('click', function (e) {
                if (header.classList.contains('nav-open') && !header.contains(e.target)) { setOpen(false); }
            });
            document.addEventListener('keydown', function (e) {
                if (e.key === 'Escape' && header.classList.contains('nav-open')) {
                    setOpen(false);
                    toggle.focus();
                }
            });
            window.addEventListener('resize', function () {
                if (window.innerWidth > 860) { setOpen(false); }
            });
        }

        // Shadow once scrolled; on small screens slide away while reading down
        // and come back on the first scroll up (CSS scopes this to <= 860px).
        var lastY = window.scrollY;
        var ticking = false;
        function update() {
            ticking = false;
            var y = window.scrollY;
            header.classList.toggle('is-scrolled', y > 4);
            if (Math.abs(y - lastY) > 6) {
                header.classList.toggle('is-hidden', y > lastY && y > 160);
                lastY = y;
            }
        }
        window.addEventListener('scroll', function () {
            if (!ticking) {
                ticking = true;
                window.requestAnimationFrame(update);
            }
        }, {passive: true});
        update();
    }

    /* ---------- Floating buttons ---------- */
    var ICON_UP = '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>';
    var ICON_LIST = '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>';

    function fabStack() {
        var stack = document.getElementById('fab-stack');
        if (!stack) {
            stack = document.createElement('div');
            stack.id = 'fab-stack';
            stack.className = 'fab-stack';
            document.body.appendChild(stack);
        }
        return stack;
    }

    function initBackToTop() {
        var btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'fab fab-top is-off';
        btn.setAttribute('aria-label', '回到顶部');
        btn.innerHTML = ICON_UP;
        btn.addEventListener('click', function () {
            window.scrollTo({top: 0, behavior: 'smooth'});
        });
        fabStack().appendChild(btn);

        var ticking = false;
        window.addEventListener('scroll', function () {
            if (ticking) { return; }
            ticking = true;
            window.requestAnimationFrame(function () {
                ticking = false;
                btn.classList.toggle('is-off', window.scrollY < 600);
            });
        }, {passive: true});
    }

    /* ---------- Reading progress bar ---------- */
    function initReadingProgress() {
        var bar = document.getElementById('reading-progress');
        if (!bar) { return; }
        var ticking = false;
        function update() {
            ticking = false;
            var docHeight = root.scrollHeight - window.innerHeight;
            bar.style.width = docHeight > 0 ? (Math.min(1, window.scrollY / docHeight) * 100) + '%' : '0';
        }
        window.addEventListener('scroll', function () {
            if (!ticking) {
                ticking = true;
                window.requestAnimationFrame(update);
            }
        }, {passive: true});
        update();
    }

    /* ---------- "Don't leave" tab title gimmick ---------- */
    function initTitleGimmick() {
        var originalTitle = document.title;
        var timer = null;
        document.addEventListener('visibilitychange', function () {
            clearTimeout(timer);
            if (document.visibilityState === 'hidden') {
                document.title = '你别走吖 Σ(っ °д° ;)っ';
            } else {
                document.title = '你可算回来了٩(๛ ˘ ³˘)۶';
                timer = setTimeout(function () { document.title = originalTitle; }, 2000);
            }
        });
    }

    /* ---------- Prefetch on hover / touch ---------- */
    // Replaces the render-blocking third-party quicklink script. Only the link
    // the visitor is about to open is fetched, not every link in the viewport.
    function initPrefetch() {
        var conn = navigator.connection;
        if (conn && (conn.saveData || /2g/.test(conn.effectiveType || ''))) { return; }
        var support = document.createElement('link').relList;
        if (!support || !support.supports || !support.supports('prefetch')) { return; }

        var done = {};
        var hoverTimer = null;

        function eligible(a) {
            if (!a || !a.href || a.target === '_blank' || a.hasAttribute('download')) { return false; }
            var url;
            try { url = new URL(a.href, location.href); } catch (e) { return false; }
            if (url.origin !== location.origin) { return false; }
            if (url.pathname === location.pathname && url.search === location.search) { return false; }
            if (/^\/(admin|feed|static|covers|random)/.test(url.pathname) || /\.xml$/.test(url.pathname)) { return false; }
            return !done[url.href] && url.href;
        }

        function prefetch(href) {
            done[href] = true;
            var link = document.createElement('link');
            link.rel = 'prefetch';
            link.href = href;
            document.head.appendChild(link);
        }

        document.addEventListener('mouseover', function (e) {
            var a = e.target.closest ? e.target.closest('a') : null;
            var href = eligible(a);
            if (!href) { return; }
            clearTimeout(hoverTimer);
            hoverTimer = setTimeout(function () { prefetch(href); }, 80);
            a.addEventListener('mouseout', function () { clearTimeout(hoverTimer); }, {once: true});
        }, {passive: true});

        document.addEventListener('touchstart', function (e) {
            var a = e.target.closest ? e.target.closest('a') : null;
            var href = eligible(a);
            if (href) { prefetch(href); }
        }, {passive: true});
    }

    /* ---------- Code blocks ---------- */
    function copyText(text) {
        if (navigator.clipboard && window.isSecureContext) {
            return navigator.clipboard.writeText(text);
        }
        return new Promise(function (resolve, reject) {
            var ta = document.createElement('textarea');
            ta.value = text;
            ta.setAttribute('readonly', '');
            ta.style.position = 'fixed';
            ta.style.opacity = '0';
            document.body.appendChild(ta);
            ta.select();
            try {
                document.execCommand('copy') ? resolve() : reject(new Error('copy failed'));
            } catch (err) {
                reject(err);
            }
            document.body.removeChild(ta);
        });
    }

    function addCodeCopyButtons() {
        var codeBlocks = document.querySelectorAll('.article-content pre, .markdown-body pre');
        Array.prototype.forEach.call(codeBlocks, function (pre) {
            if (pre.parentNode.classList.contains('code-block-wrapper')) { return; }
            var wrapper = document.createElement('div');
            wrapper.className = 'code-block-wrapper';
            pre.parentNode.insertBefore(wrapper, pre);
            wrapper.appendChild(pre);

            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'copy-btn';
            btn.textContent = '复制';
            btn.addEventListener('click', function () {
                // prettify renders <ol><li> per line; join them so newlines survive.
                var lines = pre.querySelectorAll('ol.linenums > li');
                var text;
                if (lines.length) {
                    text = Array.prototype.map.call(lines, function (li) { return li.textContent; }).join('\n');
                } else {
                    var code = pre.querySelector('code');
                    text = code ? code.textContent : pre.textContent;
                }
                copyText(text).then(function () {
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
        });
    }

    function addImageLazyLoading() {
        var images = document.querySelectorAll('.article-content img, .markdown-body img');
        Array.prototype.forEach.call(images, function (img) {
            img.setAttribute('loading', 'lazy');
            img.setAttribute('decoding', 'async');
        });
    }

    /* ---------- Article content helpers ---------- */
    function wrapTables() {
        var tables = document.querySelectorAll('.article-content table');
        Array.prototype.forEach.call(tables, function (table) {
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
        var links = document.querySelectorAll('.article-content a[href]');
        Array.prototype.forEach.call(links, function (a) {
            var url;
            try { url = new URL(a.getAttribute('href'), location.href); } catch (e) { return; }
            if (!/^https?:$/.test(url.protocol) || url.origin === location.origin) { return; }
            a.target = '_blank';
            a.rel = 'noopener noreferrer';
        });
    }

    /* ---------- Table of contents ---------- */
    function buildTOC(headings) {
        // Indent by rank of the heading levels actually used, so an article that
        // jumps from h2 straight to h4 still gets a tidy two-level outline.
        var used = [];
        Array.prototype.forEach.call(headings, function (h) {
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
        toc.appendChild(title);

        var ul = document.createElement('ul');
        Array.prototype.forEach.call(headings, function (heading, i) {
            var id = 'toc-heading-' + i;
            heading.id = id;

            var level = Math.min(4, used.indexOf(parseInt(heading.tagName.charAt(1), 10)));
            var li = document.createElement('li');
            li.className = 'toc-level-' + level;
            var a = document.createElement('a');
            a.href = '#' + id;
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
        toggleBtn.innerHTML = ICON_LIST;
        var stack = fabStack();
        stack.insertBefore(toggleBtn, stack.firstChild);

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
            var target = document.getElementById(link.getAttribute('href').slice(1));
            if (target) {
                target.scrollIntoView({behavior: 'smooth', block: 'start'});
                if (history.replaceState) { history.replaceState(null, '', link.getAttribute('href')); }
            }
        });

        // Scroll spy
        var ticking = false;
        var lastActive = null;
        function spy() {
            ticking = false;
            var current = headings[0].id;
            Array.prototype.forEach.call(headings, function (heading) {
                if (heading.getBoundingClientRect().top <= 110) { current = heading.id; }
            });
            if (current === lastActive) { return; }
            lastActive = current;
            Array.prototype.forEach.call(tocLinks, function (link) {
                var on = link.getAttribute('href') === '#' + current;
                link.classList.toggle('active', on);
                if (on && !toc.classList.contains('is-sheet') && toc.scrollHeight > toc.clientHeight) {
                    // keep the active entry visible inside a long sidebar TOC
                    var top = link.offsetTop - toc.clientHeight / 2;
                    toc.scrollTop = Math.max(0, top);
                }
            });
        }
        window.addEventListener('scroll', function () {
            if (!ticking) {
                ticking = true;
                window.requestAnimationFrame(spy);
            }
        }, {passive: true});
        spy();
    }

    // Called by the Markdown bootstrap once the article HTML exists.
    window.enhancePost = function () {
        addCodeCopyButtons();
        addImageLazyLoading();
        wrapTables();
        markExternalLinks();
        var viewer = document.getElementById('post-viewer');
        if (viewer) {
            var headings = viewer.querySelectorAll('h1, h2, h3, h4, h5');
            if (headings.length >= 2) { buildTOC(headings); }
        }
    };

    function init() {
        initThemeToggle();
        initHeader();
        initBackToTop();
        initReadingProgress();
        initTitleGimmick();
        initPrefetch();
        if (!document.getElementById('post-viewer') && !document.getElementById('page-viewer')) {
            addCodeCopyButtons();
            addImageLazyLoading();
        }
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
