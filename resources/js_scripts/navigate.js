// Navigate to any URL (parameterized)
// Params: { "url": "https://..." }
console.log('[SNIPPET] navigate.js - Starting navigation');

const targetUrl = params?.url || 'https://www.google.com';
console.log('[SNIPPET] Target URL:', targetUrl);
console.log('[SNIPPET] Current URL:', window.location.href);

// Validate URL format
try {
  new URL(targetUrl);
} catch (error) {
  console.error('[SNIPPET] Invalid URL format:', targetUrl);
  return `Invalid URL: ${targetUrl}`;
}

window.location.href = targetUrl;

console.log('[SNIPPET] Navigation command issued');
return `Navigating to: ${targetUrl}`;
