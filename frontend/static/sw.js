// DockerVerse Service Worker — self-unregistering cleanup version
// The previous SW used Cache First for JS/CSS which caused stale chunks
// after frontend rebuilds, resulting in a permanent "Loading..." screen.
// This version clears all caches and unregisters itself so the browser
// fetches everything fresh from the network on every visit.

self.addEventListener('install', () => {
  self.skipWaiting();
});

self.addEventListener('activate', (event) => {
  event.waitUntil(
    caches.keys()
      .then((keys) => Promise.all(keys.map((key) => caches.delete(key))))
      .then(() => self.registration.unregister())
      .then(() => self.clients.matchAll())
      .then((clients) => clients.forEach((client) => client.navigate(client.url)))
  );
});
