console.log("Service-worker loaded.");

const cacheName = 'site-cache-v1'
const assetsToCache = [
    '/pwa-examples/',
    '/pwa-examples/index.html',
    '/pwa-examples/css/styles.css',
    '/pwa-examples/js/app.js',
]
self.addEventListener('install', ( event ) => {
  self.skipWaiting(); // skip waiting
  event.waitUntil(  
      caches.open(cacheName).then((cache) => {
            return cache.addAll(assetsToCache);
      })
    );
});