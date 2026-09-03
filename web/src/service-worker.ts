/// <reference types="@sveltejs/kit" />
/// <reference lib="webworker" />

import { build, files, version } from '$service-worker';

const worker = self as unknown as ServiceWorkerGlobalScope;
const CACHE = `fajr-${version}`;
const SHELL = [...build, ...files];

// Pages a learner is likely to reopen on a bad connection. Anything else is
// fetched fresh and only falls back to the cache when the network is gone.
const OFFLINE = '/offline';

worker.addEventListener('install', (event) => {
	event.waitUntil(
		caches
			.open(CACHE)
			.then((cache) => cache.addAll([...SHELL, OFFLINE]))
			.then(() => worker.skipWaiting())
	);
});

worker.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			for (const key of await caches.keys()) {
				if (key !== CACHE) await caches.delete(key);
			}
			await worker.clients.claim();
		})()
	);
});

worker.addEventListener('fetch', (event) => {
	const url = new URL(event.request.url);
	if (event.request.method !== 'GET' || url.origin !== location.origin) return;

	// The session lives in a cookie, so a cached page could show one person's
	// work to the next. Only the shell and the offline page are kept.
	const isShell = SHELL.includes(url.pathname);

	event.respondWith(
		(async () => {
			const cache = await caches.open(CACHE);
			if (isShell) {
				const hit = await cache.match(url.pathname);
				if (hit) return hit;
			}
			try {
				const response = await fetch(event.request);
				if (isShell && response.ok) cache.put(url.pathname, response.clone());
				return response;
			} catch {
				const hit = await cache.match(url.pathname);
				if (hit) return hit;
				if (event.request.mode === 'navigate') {
					const offline = await cache.match(OFFLINE);
					if (offline) return offline;
				}
				throw new Error('offline');
			}
		})()
	);
});
