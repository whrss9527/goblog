(function () {
    // Dark mode
    var theme = localStorage.getItem('theme') || 'light';
    document.documentElement.setAttribute('data-theme', theme);

    document.addEventListener('DOMContentLoaded', function () {
        var toggle = document.getElementById('dark-toggle');
        if (toggle) {
            toggle.addEventListener('click', function () {
                theme = theme === 'dark' ? 'light' : 'dark';
                document.documentElement.setAttribute('data-theme', theme);
                localStorage.setItem('theme', theme);
                var giscusFrame = document.querySelector('iframe.giscus-frame');
                if (giscusFrame) {
                    giscusFrame.contentWindow.postMessage({
                        giscus: {setConfig: {theme: theme === 'dark' ? 'dark' : 'light'}}
                    }, 'https://giscus.app');
                }
            });
        }

        // Reading progress bar
        var progressBar = document.getElementById('reading-progress');
        if (progressBar) {
            window.addEventListener('scroll', function () {
                var scrollTop = window.scrollY;
                var docHeight = document.documentElement.scrollHeight - window.innerHeight;
                if (docHeight > 0) {
                    progressBar.style.width = (scrollTop / docHeight * 100) + '%';
                }
            });
        }

        // For non-post pages, enhance immediately
        var postViewer = document.getElementById('post-viewer');
        if (!postViewer) {
            addCodeCopyButtons();
            addImageLazyLoading();
        }
    });

    function addCodeCopyButtons() {
        var codeBlocks = document.querySelectorAll('pre');
        codeBlocks.forEach(function (pre) {
            var wrapper = document.createElement('div');
            wrapper.className = 'code-block-wrapper';
            pre.parentNode.insertBefore(wrapper, pre);
            wrapper.appendChild(pre);

            var btn = document.createElement('button');
            btn.className = 'copy-btn';
            btn.textContent = '复制';
            btn.addEventListener('click', function () {
                var code = pre.querySelector('code');
                var text = code ? code.textContent : pre.textContent;
                navigator.clipboard.writeText(text).then(function () {
                    btn.textContent = '已复制';
                    btn.classList.add('copied');
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
        images.forEach(function (img) {
            img.setAttribute('loading', 'lazy');
        });
    }

    window.enhancePost = function () {
        addCodeCopyButtons();
        addImageLazyLoading();
        var viewer = document.getElementById('post-viewer');
        console.log('[TOC] post-viewer found:', !!viewer);
        if (viewer) {
            var headings = viewer.querySelectorAll('h1, h2, h3, h4, h5');
            console.log('[TOC] headings found:', headings.length);
            if (headings.length >= 2) {
                buildTOC(viewer, headings);
            }
        }
    };

    function buildTOC(article, headings) {

        var minLevel = 6;
        headings.forEach(function (h) {
            var level = parseInt(h.tagName.charAt(1));
            if (level < minLevel) minLevel = level;
        });

        var toc = document.createElement('div');
        toc.className = 'toc-container';
        var title = document.createElement('div');
        title.className = 'toc-title';
        title.textContent = '目录';
        toc.appendChild(title);

        var ul = document.createElement('ul');
        headings.forEach(function (heading, i) {
            var id = 'toc-heading-' + i;
            heading.id = id;

            var level = parseInt(heading.tagName.charAt(1)) - minLevel;
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
        toggleBtn.className = 'toc-toggle';
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
        toc.querySelectorAll('a').forEach(function (link) {
            link.addEventListener('click', closeMobileToc);
        });

        // Scroll spy
        var tocLinks = toc.querySelectorAll('a');
        window.addEventListener('scroll', function () {
            var current = '';
            headings.forEach(function (heading) {
                if (heading.getBoundingClientRect().top <= 100) {
                    current = heading.id;
                }
            });
            tocLinks.forEach(function (link) {
                link.classList.remove('active');
                if (link.getAttribute('href') === '#' + current) {
                    link.classList.add('active');
                }
            });
        });
    }
})();
