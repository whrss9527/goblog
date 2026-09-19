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

    function applyTheme(theme, persist) {
        root.setAttribute('data-theme', theme);
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
            var flip = function () {
                applyTheme(currentTheme() === 'dark' ? 'light' : 'dark', true);
            };
            toggle.addEventListener('click', flip);
            toggle.addEventListener('keydown', function (e) {
                if (e.key === 'Enter' || e.key === ' ') {
                    e.preventDefault();
                    flip();
                }
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

    /* ---------- Table of contents ---------- */
    function buildTOC(headings) {
        var minLevel = 6;
        Array.prototype.forEach.call(headings, function (h) {
            var level = parseInt(h.tagName.charAt(1), 10);
            if (level < minLevel) { minLevel = level; }
        });

        var toc = document.createElement('nav');
        toc.className = 'toc-container';
        toc.setAttribute('aria-label', '文章目录');
        var title = document.createElement('div');
        title.className = 'toc-title';
        title.textContent = '目录';
        toc.appendChild(title);

        var ul = document.createElement('ul');
        Array.prototype.forEach.call(headings, function (heading, i) {
            var id = 'toc-heading-' + i;
            heading.id = id;

            var level = parseInt(heading.tagName.charAt(1), 10) - minLevel;
            var li = document.createElement('li');
            li.className = 'toc-level-' + level;
            var a = document.createElement('a');
            a.href = '#' + id;
            a.textContent = heading.textContent;
            a.addEventListener('click', function (e) {
                e.preventDefault();
                heading.scrollIntoView({behavior: 'smooth', block: 'start'});
            });
            li.appendChild(a);
            ul.appendChild(li);
        });
        toc.appendChild(ul);
        document.body.appendChild(toc);

        // Mobile: toggle button + overlay
        var toggleBtn = document.createElement('button');
        toggleBtn.type = 'button';
        toggleBtn.className = 'toc-toggle';
        toggleBtn.setAttribute('aria-label', '打开目录');
        toggleBtn.textContent = '☰';
        var overlay = document.createElement('div');
        overlay.className = 'toc-overlay';
        document.body.appendChild(overlay);
        document.body.appendChild(toggleBtn);

        function closeMobileToc() {
            toc.classList.remove('mobile-visible');
            overlay.classList.remove('visible');
        }

        toggleBtn.addEventListener('click', function () {
            var open = toc.classList.toggle('mobile-visible');
            overlay.classList.toggle('visible', open);
        });
        overlay.addEventListener('click', closeMobileToc);
        var tocLinks = toc.querySelectorAll('a');
        Array.prototype.forEach.call(tocLinks, function (link) {
            link.addEventListener('click', closeMobileToc);
        });

        // Scroll spy
        var ticking = false;
        function spy() {
            ticking = false;
            var current = '';
            Array.prototype.forEach.call(headings, function (heading) {
                if (heading.getBoundingClientRect().top <= 100) { current = heading.id; }
            });
            Array.prototype.forEach.call(tocLinks, function (link) {
                link.classList.toggle('active', link.getAttribute('href') === '#' + current);
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
        var viewer = document.getElementById('post-viewer');
        if (viewer) {
            var headings = viewer.querySelectorAll('h1, h2, h3, h4, h5');
            if (headings.length >= 2) { buildTOC(headings); }
        }
    };

    function init() {
        initThemeToggle();
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
