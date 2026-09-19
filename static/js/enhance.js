/* goblog front-end core (vanilla JS, no dependencies): theme, header, floating
   buttons, prefetch and a few helpers shared through window.goblog.
   Article-only features live in article.js. */
(function () {
    'use strict';

    var root = document.documentElement;

    var ICONS = {
        up: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><line x1="12" y1="19" x2="12" y2="5"/><polyline points="5 12 12 5 19 12"/></svg>',
        list: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/><line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/></svg>',
        link: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"/><path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"/></svg>',
        close: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>',
        left: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><polyline points="15 18 9 12 15 6"/></svg>',
        right: '<svg class="icon" viewBox="0 0 24 24" aria-hidden="true"><polyline points="9 18 15 12 9 6"/></svg>'
    };

    /* ---------- Small helpers ---------- */
    function each(list, fn) {
        Array.prototype.forEach.call(list, fn);
    }

    // rAF-throttled passive scroll listener.
    function onScroll(fn) {
        var ticking = false;
        window.addEventListener('scroll', function () {
            if (ticking) { return; }
            ticking = true;
            window.requestAnimationFrame(function () {
                ticking = false;
                fn();
            });
        }, {passive: true});
    }

    function storageGet(key) {
        try { return localStorage.getItem(key); } catch (e) { return null; }
    }

    function storageSet(key, value) {
        try { localStorage.setItem(key, value); } catch (e) { /* private mode / quota */ }
    }

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

    /* ---------- Toast ---------- */
    // toast('text') or toast('text', {action: '继续', onAction: fn, duration: 8000})
    var toastTimer = null;
    function toast(message, opts) {
        opts = opts || {};
        var el = document.getElementById('toast');
        if (!el) {
            el = document.createElement('div');
            el.id = 'toast';
            el.className = 'toast';
            el.setAttribute('role', 'status');
            el.setAttribute('aria-live', 'polite');
            document.body.appendChild(el);
        }
        el.innerHTML = '';
        var text = document.createElement('span');
        text.textContent = message;
        el.appendChild(text);

        function hide() {
            clearTimeout(toastTimer);
            el.classList.remove('is-visible');
        }

        if (opts.action) {
            var btn = document.createElement('button');
            btn.type = 'button';
            btn.className = 'toast-action';
            btn.textContent = opts.action;
            btn.addEventListener('click', function () {
                hide();
                if (opts.onAction) { opts.onAction(); }
            });
            el.appendChild(btn);
            var close = document.createElement('button');
            close.type = 'button';
            close.className = 'toast-close';
            close.setAttribute('aria-label', '关闭');
            close.innerHTML = ICONS.close;
            close.addEventListener('click', hide);
            el.appendChild(close);
        }

        // restart the transition even when a toast is already showing
        el.classList.remove('is-visible');
        void el.offsetWidth;
        el.classList.add('is-visible');
        clearTimeout(toastTimer);
        toastTimer = setTimeout(hide, opts.duration || 2400);
        return hide;
    }

    /* ---------- Theme ---------- */
    // The initial theme is applied by an inline script in <head> (no flash).
    function currentTheme() {
        return root.getAttribute('data-theme') === 'dark' ? 'dark' : 'light';
    }

    function savedTheme() {
        var t = storageGet('theme');
        return (t === 'dark' || t === 'light') ? t : null;
    }

    function syncToggleState() {
        var toggle = document.getElementById('dark-toggle');
        if (toggle) { toggle.setAttribute('aria-checked', currentTheme() === 'dark' ? 'true' : 'false'); }
    }

    function applyTheme(theme, persist) {
        root.setAttribute('data-theme', theme);
        syncToggleState();
        if (persist) { storageSet('theme', theme); }
        var giscusFrame = document.querySelector('iframe.giscus-frame');
        if (giscusFrame && giscusFrame.contentWindow) {
            giscusFrame.contentWindow.postMessage({
                giscus: {setConfig: {theme: theme === 'dark' ? 'dark' : 'light'}}
            }, 'https://giscus.app');
        }
    }

    function toggleTheme() {
        applyTheme(currentTheme() === 'dark' ? 'light' : 'dark', true);
    }

    function initThemeToggle() {
        var toggle = document.getElementById('dark-toggle');
        if (toggle) {
            syncToggleState();
            toggle.addEventListener('click', toggleTheme);
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
        function update() {
            var y = window.scrollY;
            header.classList.toggle('is-scrolled', y > 4);
            if (Math.abs(y - lastY) > 6) {
                header.classList.toggle('is-hidden', y > lastY && y > 160);
                lastY = y;
            }
        }
        onScroll(update);
        update();
    }

    /* ---------- Floating buttons ---------- */
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
        btn.innerHTML = ICONS.up;
        btn.addEventListener('click', function () {
            window.scrollTo({top: 0, behavior: 'smooth'});
        });
        fabStack().appendChild(btn);
        onScroll(function () {
            btn.classList.toggle('is-off', window.scrollY < 600);
        });
    }

    /* ---------- Reading progress bar ---------- */
    function readingRatio() {
        var docHeight = root.scrollHeight - window.innerHeight;
        return docHeight > 0 ? Math.min(1, Math.max(0, window.scrollY / docHeight)) : 0;
    }

    function initReadingProgress() {
        var bar = document.getElementById('reading-progress');
        if (!bar) { return; }
        function update() { bar.style.width = (readingRatio() * 100) + '%'; }
        onScroll(update);
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
            if (/^\/(admin|api|feed|static|covers|random)/.test(url.pathname) || /\.xml$/.test(url.pathname)) { return false; }
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

    // Shared with article.js and page-level scripts.
    window.goblog = {
        icons: ICONS,
        each: each,
        onScroll: onScroll,
        storageGet: storageGet,
        storageSet: storageSet,
        copyText: copyText,
        toast: toast,
        fabStack: fabStack,
        readingRatio: readingRatio,
        toggleTheme: toggleTheme
    };

    function init() {
        initThemeToggle();
        initHeader();
        initBackToTop();
        initReadingProgress();
        initTitleGimmick();
        initPrefetch();
    }

    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
