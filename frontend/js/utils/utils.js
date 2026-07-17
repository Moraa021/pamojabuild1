export function formatSats(sats) {
  if (sats == null || isNaN(sats)) return '—';
  return `${Number(sats).toLocaleString()} sats`;
}

export function satsToBtc(sats) {
  if (sats == null || isNaN(sats)) return '—';
  return (sats / 1e8).toFixed(8) + ' BTC';
}

export function formatDate(value) {
  if (!value) return '—';
  const d = typeof value === 'number' ? new Date(value * 1000) : new Date(value);
  return d.toLocaleString(undefined, { dateStyle: 'medium', timeStyle: 'short' });
}

export function secondsUntil(unixTs) {
  return Math.max(0, unixTs - Math.floor(Date.now() / 1000));
}

export function truncateHex(hex, chars = 8) {
  if (!hex || hex.length <= chars * 2) return hex;
  return `${hex.slice(0, chars)}…${hex.slice(-chars)}`;
}

export function slugToTitle(slug) {
  return slug ? slug.replace(/-/g, ' ').replace(/\b\w/g, c => c.toUpperCase()) : '';
}

export function isValidEmail(email) {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(email);
}

export function isValidPhone(phone) {
  return /^\+?[0-9]{7,15}$/.test(phone.replace(/[\s\-().]/g, ''));
}

export function isValidXpub(xpub) {
  return typeof xpub === 'string' && /^xpub[A-Za-z0-9]{107}$/.test(xpub);
}

export function isHex(str) {
  return typeof str === 'string' && /^[0-9a-fA-F]+$/.test(str);
}

export function $one(selector, root = document) {
  const el = root.querySelector(selector);
  if (!el) throw new Error(`Element not found: ${selector}`);
  return el;
}

export function $all(selector, root = document) {
  return Array.from(root.querySelectorAll(selector));
}

export function setText(el, text) {
  if (el) el.textContent = text;
}

export function getPathParam(index) {
  return window.location.pathname.split('/').filter(Boolean)[index] || null;
}

export function navigate(path) {
  window.history.pushState({}, '', path);
  window.dispatchEvent(new PopStateEvent('popstate'));
}