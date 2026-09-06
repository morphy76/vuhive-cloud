// PLACEHOLDER SERVICE WORKER FOR ISSUE #58
// To be replaced by Vite PWA / Workbox build in Issue #63
self.addEventListener('install', (event) => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener('fetch', (event) => {
  // Pass-through during placeholder testing
});
