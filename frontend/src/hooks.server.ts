import type { Handle } from '@sveltejs/kit';

export const handle: Handle = async ({ event, resolve }) => {
  const response = await resolve(event, {
    // Preload critical assets
    preload: ({ type }) => {
      return type === 'font' || type === 'css' || type === 'js';
    }
  });

  const url = event.url.pathname;
  
  // Static assets - aggressive caching (1 year)
  if (url.match(/\.(js|css|woff2?|ttf|eot|svg|png|jpg|webp|avif|ico)$/)) {
    response.headers.set('Cache-Control', 'public, max-age=31536000, immutable');
  }
  // HTML/main routes - short cache with revalidation
  else if (url === '/' || url.match(/^\/[^.]*$/)) {
    response.headers.set('Cache-Control', 'public, max-age=60, stale-while-revalidate=300');
  }
  
  // Security & performance headers
  response.headers.set('X-Content-Type-Options', 'nosniff');
  response.headers.set('X-Frame-Options', 'SAMEORIGIN');
  response.headers.set('Referrer-Policy', 'strict-origin-when-cross-origin');
  
  // Enable compression negotiation hint
  response.headers.set('Vary', 'Accept-Encoding');
  
  return response;
};
