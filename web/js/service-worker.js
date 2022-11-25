/*
Copyright 2015, 2019 Google Inc. All Rights Reserved.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 http://www.apache.org/licenses/LICENSE-2.0
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

// Service worker loaded
console.log('Loaded service worker!');

const vapidPublicKey = "NONE";

// Incrementing OFFLINE_VERSION will kick off the install event and force
// previously cached resources to be updated from the network.
const OFFLINE_VERSION = 1;
const CACHE_NAME = 'poenskelisten-cache';
// Customize this with a different URL if needed.
const OFFLINE_URL = 'offline.html';
const urlsToCache = [
  '/',
	'manifest.json',
	'offline.html',
  'index.html',
  'index.js',
  'functions.js',
	'css/custom.css',
  'assets/logo/version3/logo_red.svg',
	'assets/logo/version4/logo_round_red.png',
	'assets/logo/version4/logo_round_red.svg',
	'assets/logo/version4/logo_square_red.png',
	'assets/minecraft.svg',
  'assets/pattern.png',
	'assets/tab.png',
	'assets/icons/icon-72x72.png',
	'assets/icons/icon-96x96.png',
	'assets/icons/icon-128x128.png',
	'assets/icons/icon-192x192.png',
	'assets/icons/icon-384x384.png',
	'assets/icons/icon-512x512.png',
	'assets/launch-screens/launch-screen-2048x2732.png',
	'assets/launch-screens/launch-screen-2732x2048.png',
	'assets/launch-screens/launch-screen-1668x2388.png',
	'assets/launch-screens/launch-screen-2388x1668.png',
	'assets/launch-screens/launch-screen-1668x2224.png',
	'assets/launch-screens/launch-screen-2224x1668.png',
	'assets/launch-screens/launch-screen-2048x1536.png',
	'assets/launch-screens/launch-screen-1536x2048.png',
	'assets/launch-screens/launch-screen-1242x2688.png',
	'assets/launch-screens/launch-screen-2688x1242.png',
	'assets/launch-screens/launch-screen-828x1792.png',
	'assets/launch-screens/launch-screen-1792x828.png',
	'assets/launch-screens/launch-screen-1125x2436.png',
	'assets/launch-screens/launch-screen-2436x1125.png',
	'assets/launch-screens/launch-screen-1242x2208.png',
	'assets/launch-screens/launch-screen-2208x1242.png',
	'assets/launch-screens/launch-screen-750x1334.png',
	'assets/launch-screens/launch-screen-1334x750.png',
	'assets/launch-screens/launch-screen-640x1136.png',
	'assets/launch-screens/launch-screen-1136x640.png',
	'assets/favicons/apple-touch-icon-57x57.png',
	'assets/favicons/apple-touch-icon-60x60.png',
	'assets/favicons/apple-touch-icon-72x72.png',
	'assets/favicons/apple-touch-icon-76x76.png',
	'assets/favicons/apple-touch-icon-114x114.png',
	'assets/favicons/apple-touch-icon-120x120.png',
	'assets/favicons/apple-touch-icon-144x144.png',
	'assets/favicons/apple-touch-icon-152x152.png',
	'assets/favicons/favicon-16x16.png',
	'assets/favicons/favicon-32x32.png',
	'assets/favicons/favicon-96x96.png',
	'assets/favicons/favicon-128x128.png',
	'assets/favicons/favicon-196x196.png',
	'assets/favicons/ms-tile-70x70.png',
	'assets/favicons/ms-tile-144x144.png',
	'assets/favicons/ms-tile-150x150.png',
	'assets/favicons/ms-tile-310x150.png',
	'assets/favicons/ms-tile-310x310.png',
	'assets/favicons/favicon.ico'
];

self.addEventListener('install', (event) => {
  event.waitUntil((async () => {
    const cache = await caches.open(CACHE_NAME);
    // Setting {cache: 'reload'} in the new request will ensure that the response
    // isn't fulfilled from the HTTP cache; i.e., it will be from the network.
    for(var i = 0; i < urlsToCache.length; i++) {
        await cache.add(new Request(urlsToCache[i], {cache: 'reload'}));
    }
  })());
});

self.addEventListener('activate', (event) => {
  event.waitUntil((async () => {
    // Enable navigation preload if it's supported.
    // See https://developers.google.com/web/updates/2017/02/navigation-preload
    if ('navigationPreload' in self.registration) {
      await self.registration.navigationPreload.enable();
    }
  })());

  // Tell the active service worker to take control of the page immediately.
  self.clients.claim();
});

self.addEventListener('fetch', (event) => {
    // We only want to call event.respondWith() if this is a navigation request
    // for an HTML page.
    if (event.request.mode === 'navigate') {
        event.respondWith((async () => {
            try {
                // First, try to use the navigation preload response if it's supported.
                const preloadResponse = await event.preloadResponse;
                if (preloadResponse) {
                  return preloadResponse;
                }

                const networkResponse = await fetch(event.request);
                return networkResponse;
            } catch (error) {
                // catch is only triggered if an exception is thrown, which is likely
                // due to a network error.
                // If fetch() returns a valid HTTP response with a response code in
                // the 4xx or 5xx range, the catch() will NOT be called.
                console.log('Fetch failed; returning offline page instead.', error);

                const cache = await caches.open(CACHE_NAME);
                const cachedResponse = await cache.match(OFFLINE_URL);
                return cachedResponse;
            }
          })());
    }

    // If our if() condition is false, then this fetch handler won't intercept the
    // request. If there are any other fetch handlers registered, they will get a
    // chance to call event.respondWith(). If no fetch handlers call
    // event.respondWith(), the request will be handled by the browser as if there
    // were no service worker involvement.
});

self.addEventListener('notificationclose', event => {
    const notification = event.notification;
    const primaryKey = notification.data.primaryKey;

    console.log('Closed notification: ' + primaryKey);
});

self.addEventListener('notificationclick', event => {
    const notification = event.notification;
    const primaryKey = notification.data.primaryKey;
    const action = event.action;

    if (action === 'close') {
        notification.close();
    } else {
        clients.openWindow('chat.html');
        notification.close();
    }

    // TODO 5.3 - close all notifications when one is clicked

});

self.addEventListener('push', event => {
    let body;

    console.log(event.data);

    if (event.data) {
      body = event.data.text();
    } else {
      body = 'Krenkelsesarmeen';
    }

    const options = {
        body: body,
        icon: 'assets/logo/version4/logo_round_red.png',
        badge: 'assets/logo/version4/logo_round_red_trans.png',
        vibrate: [100, 50, 100],
        data: {
          dateOfArrival: Date.now(),
          primaryKey: 1
        },
        actions: [
            {action: 'explore', title: 'Read chat',
              icon: 'images/checkmark.png'},
            {action: 'close', title: 'Close the notification',
              icon: 'images/xmark.png'},
        ],
        tag: 'Message'
    };

    event.waitUntil(
        self.registration.showNotification('Krenkelsesarmeen Chat', options)
    );
});