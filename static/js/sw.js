/* goblog service worker: offline reading and instant static assets.

   Served as /sw.js by internal/handler/front/pwa.go, which prepends
   `self.__GOBLOG = {version, precache, offline}`.

   - pages (navigations): network first. Every successfully loaded page is kept,
     so articles that were read once stay readable without a connection. When
     the network fails, or takes too long, the kept copy is used; an address
     that was never visited gets the offline page.
   - /static and /covers: fingerprinted URLs (?v=) never change and come from
     the cache; other files are served from the cache and refreshed in the
     background.
   - /admin (and its assets), /api, /random, feeds, non-GET and cross-origin
     requests are none of this worker's business. */
(function () {
    'use strict';

    var CONFIG = self.__GOBLOG || {version: 'dev', precache: [], offline: '/offline'};
    var PREFIX = 'goblog-';
    var PAGES = PREFIX + 'pages';
    var STATIC = PREFIX + 'static';
    var PAGE_LIMIT = 60;
    var STATIC_LIMIT = 150;
    var NETWORK_TIMEOUT = 5000;
    var BYPASS = /^\/(admin|api|random|feed|intro|sw\.js|manifest\.webmanifest|robots\.txt|ping)(\/|$)|\.xml$/;

    // what the offline page needs must never be evicted
    var PINNED = {};
    (CONFIG.precache || []).concat([CONFIG.offline]).forEach(function (path) {
        PINNED[new URL(path, self.location.href).href] = true;
    });

    function trim(cacheName, limit) {
        return caches.open(cacheName).then(function (cache) {
            return cache.keys().then(function (keys) {
                // keys come back oldest first
                var evictable = keys.filter(function (key) { return !PINNED[key.url]; });
                var extra = evictable.slice(0, Math.max(0, keys.length - limit));
                return Promise.all(extra.map(function (key) { return cache.delete(key); }));
            });
        });
    }

    function store(cacheName, request, response, limit) {
        return caches.open(cacheName).then(function (cache) {
            // delete first so a refreshed entry moves to the young end of the cache
            return cache.delete(request).then(function () {
                return cache.put(request, response);
            });
        }).then(function () {
            return trim(cacheName, limit);
        }).catch(function () { /* quota exceeded or similar: caching is best effort */ });
    }

    function fromCache(request, url) {
        return caches.match(request, {ignoreVary: true}).then(function (hit) {
            if (hit || !/^\/(posts|pages)\//.test(url.pathname)) { return hit; }
            // /posts/x?utm_source=… is still /posts/x
            return caches.match(request, {ignoreVary: true, ignoreSearch: true});
        });
    }

    function keepable(url, response) {
        return response && response.ok && response.type === 'basic' && !url.searchParams.has('keyword') &&
            (response.headers.get('Content-Type') || '').indexOf('text/html') === 0;
    }

    function page(event, url) {
        var request = event.request;
        return new Promise(function (resolve) {
            var settled = false;
            function settle(response) {
                if (!settled && response) { settled = true; resolve(response); }
                return settled;
            }
            // slow connection: show the kept copy, the request below still refreshes it
            var timer = setTimeout(function () {
                fromCache(request, url).then(settle);
            }, NETWORK_TIMEOUT);

            var network = fetch(request).then(function (response) {
                clearTimeout(timer);
                if (keepable(url, response)) {
                    var copy = response.clone();
                    event.waitUntil(store(PAGES, request, copy, PAGE_LIMIT));
                }
                settle(response);
            }).catch(function () {
                clearTimeout(timer);
                if (settled) { return; }
                return fromCache(request, url).then(function (hit) {
                    return hit || caches.match(CONFIG.offline, {ignoreVary: true});
                }).then(function (fallback) {
                    settle(fallback || Response.error());
                });
            });
            event.waitUntil(network);
        });
    }

    function asset(event, url) {
        var request = event.request;
        return caches.match(request, {ignoreVary: true}).then(function (hit) {
            var refresh = function () {
                return fetch(request).then(function (response) {
                    if (response && response.ok && response.type === 'basic') {
                        event.waitUntil(store(STATIC, request, response.clone(), STATIC_LIMIT));
                    }
                    return response;
                });
            };
            if (!hit) { return refresh(); }
            if (!url.searchParams.has('v')) {
                event.waitUntil(refresh().catch(function () { /* offline: the cached copy is all we have */ }));
            }
            return hit;
        });
    }

    self.addEventListener('install', function (event) {
        event.waitUntil(caches.open(STATIC).then(function (cache) {
            var urls = (CONFIG.precache || []).concat([CONFIG.offline]);
            return Promise.all(urls.map(function (url) {
                // one missing file must not keep the worker from installing
                return cache.add(new Request(url, {cache: 'reload'})).catch(function () { /* skip */ });
            }));
        }).then(function () {
            return self.skipWaiting();
        }));
    });

    self.addEventListener('activate', function (event) {
        event.waitUntil(caches.keys().then(function (names) {
            return Promise.all(names.filter(function (name) {
                return name.indexOf(PREFIX) === 0 && name !== PAGES && name !== STATIC;
            }).map(function (name) { return caches.delete(name); }));
        }).then(function () {
            return self.clients.claim();
        }));
    });

    self.addEventListener('fetch', function (event) {
        var request = event.request;
        if (request.method !== 'GET') { return; }
        var url = new URL(request.url);
        if (url.origin !== self.location.origin || BYPASS.test(url.pathname)) { return; }

        if (request.mode === 'navigate') {
            event.respondWith(page(event, url));
        } else if ((/^\/(static|covers)\//.test(url.pathname) && !/^\/static\/admin\//.test(url.pathname)) || url.pathname === '/favicon.ico') {
            event.respondWith(asset(event, url));
        }
    });
})();
