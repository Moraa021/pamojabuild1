const _env = window.__ENV__ || {};

export const ENV = Object.freeze({
  API_BASE_URL:    _env.API_BASE_URL    || 'http://localhost:8080',
  API_VERSION:     _env.API_VERSION     || 'v1',
  REQUEST_TIMEOUT: _env.REQUEST_TIMEOUT || 15000,
  RETRY_ATTEMPTS:  _env.RETRY_ATTEMPTS  || 3,
  RETRY_DELAY_MS:  _env.RETRY_DELAY_MS  || 800,
  APP_ENV:         _env.APP_ENV         || 'development',
});
