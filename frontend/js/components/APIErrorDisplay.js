import { ApiError, NetworkError, TimeoutError } from '../api/client.js';

export class APIErrorDisplay {
  #container;
  #opts;
  #el = null;

  constructor(container, opts = {}) {
    this.#container = container;
    this.#opts = {
      onRetry: opts.onRetry || null,
    };
  }

  show(err) {
    this.clear();

    const { heading, body } = this.#classify(err);

    this.#el = document.createElement('div');
    this.#el.className  = 'api-error';
    this.#el.setAttribute('role', 'alert');
    this.#el.innerHTML = `
      <div class="api-error__icon" aria-hidden="true">⚠</div>
      <div class="api-error__content">
        <strong class="api-error__heading"></strong>
        <p class="api-error__body"></p>
        ${this.#opts.onRetry ? '<button class="btn btn--ghost api-error__retry">Try again</button>' : ''}
      </div>
    `;

    this.#el.querySelector('.api-error__heading').textContent = heading;
    this.#el.querySelector('.api-error__body').textContent    = body;

    if (this.#opts.onRetry) {
      this.#el.querySelector('.api-error__retry').addEventListener('click', () => {
        this.clear();
        this.#opts.onRetry();
      });
    }

    this.#container.prepend(this.#el);
  }

  /** Remove the error element */
  clear() {
    this.#el?.remove();
    this.#el = null;
  }

  #classify(err) {
    if (err instanceof TimeoutError) {
      return { heading: 'Request timed out', body: 'The server took too long to respond. Check your connection and try again.' };
    }
    if (err instanceof NetworkError) {
      return { heading: 'Connection error', body: 'Could not reach the server. Check your internet connection.' };
    }
    if (err instanceof ApiError) {
      if (err.status === 401) return { heading: 'Session expired', body: 'Please sign in again to continue.' };
      if (err.status === 403) return { heading: 'Access denied', body: 'You do not have permission to perform this action.' };
      if (err.status === 404) return { heading: 'Not found', body: 'The requested resource does not exist.' };
      if (err.status === 422) return { heading: 'Validation error', body: err.message || 'Check the fields and try again.' };
      if (err.status >= 500)  return { heading: 'Server error', body: 'Something went wrong on the server. Try again in a moment.' };
      return { heading: 'Request failed', body: err.message || 'An unexpected error occurred.' };
    }
    return { heading: 'Something went wrong', body: err.message || 'An unexpected error occurred.' };
  }
}
