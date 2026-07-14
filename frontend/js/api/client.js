import { ENV } from '../config/env.js';

export class ApiError extends Error {
  constructor(status, code, message, details = null) {
    super(message);
    this.name    = 'ApiError';
    this.status  = status;
    this.code    = code;
    this.details = details;
  }
}

export class NetworkError extends Error {
  constructor(message = 'Network request failed') {
    super(message);
    this.name = 'NetworkError';
  }
}

export class TimeoutError extends Error {
  constructor() {
    super(`Request timed out after ${ENV.REQUEST_TIMEOUT}ms`);
    this.name = 'TimeoutError';
  }
}

const requestInterceptors  = [];
const responseInterceptors = [];

export function addRequestInterceptor(fn)  { requestInterceptors.push(fn); }
export function addResponseInterceptor(fn) { responseInterceptors.push(fn); }

let _getToken = () => null;
export function setTokenAccessor(fn) { _getToken = fn; }

async function request(url, options = {}, attempt = 1) {
  let config = {
    headers: {
      'Content-Type': 'application/json',
      'Accept':       'application/json',
      ...(options.headers || {}),
    },
    ...options,
  };

  const token = _getToken();
  if (token) {
    config.headers['Authorization'] = `Bearer ${token}`;
  }

  for (const interceptor of requestInterceptors) {
    config = await interceptor(config) || config;
  }

  const controller = new AbortController();
  const timeoutId  = setTimeout(() => controller.abort(), ENV.REQUEST_TIMEOUT);
  config.signal    = controller.signal;

  let response;
  try {
    response = await fetch(url, config);
    clearTimeout(timeoutId);
  } catch (err) {
    clearTimeout(timeoutId);
    if (err.name === 'AbortError') throw new TimeoutError();
    throw new NetworkError(err.message);
  }

  for (const interceptor of responseInterceptors) {
    await interceptor(response.clone());
  }

  const contentType = response.headers.get('Content-Type') || '';
  let body = null;
  if (contentType.includes('application/json')) {
    body = await response.json();
  } else {
    body = await response.text();
  }

  if (!response.ok) {
    const isRetryable = response.status === 429 || response.status >= 500;
    if (isRetryable && attempt < ENV.RETRY_ATTEMPTS) {
      const delay = ENV.RETRY_DELAY_MS * Math.pow(2, attempt - 1);
      await new Promise(r => setTimeout(r, delay));
      return request(url, options, attempt + 1);
    }

    const message = (body && body.message) || response.statusText || 'An error occurred';
    const code    = (body && body.code)    || `HTTP_${response.status}`;
    throw new ApiError(response.status, code, message, body);
  }

  return body;
}

export const client = {
  get:    (url, opts = {})         => request(url, { method: 'GET',    ...opts }),
  post:   (url, data, opts = {})   => request(url, { method: 'POST',   body: JSON.stringify(data), ...opts }),
  put:    (url, data, opts = {})   => request(url, { method: 'PUT',    body: JSON.stringify(data), ...opts }),
  patch:  (url, data, opts = {})   => request(url, { method: 'PATCH',  body: JSON.stringify(data), ...opts }),
  delete: (url, opts = {})         => request(url, { method: 'DELETE', ...opts }),
};

export default client;
