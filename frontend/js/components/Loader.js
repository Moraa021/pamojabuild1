class LoaderManager {
  #overlay;

  constructor() {
    this.#overlay = this.#buildOverlay();
  }

  #buildOverlay() {
    const el = document.createElement('div');
    el.className = 'loader-overlay';
    el.setAttribute('role', 'status');
    el.setAttribute('aria-live', 'polite');
    el.innerHTML = `
      <div class="loader-overlay__inner">
        <div class="loader__spinner" aria-hidden="true"></div>
        <p class="loader__label"></p>
      </div>
    `;
    return el;
  }

  /** Show a full-screen loading overlay */
  show(label = 'Loading…') {
    this.#overlay.querySelector('.loader__label').textContent = label;
    document.body.appendChild(this.#overlay);
    requestAnimationFrame(() => this.#overlay.classList.add('loader-overlay--visible'));
  }

  /** Hide and remove the overlay */
  hide() {
    this.#overlay.classList.remove('loader-overlay--visible');
    this.#overlay.addEventListener('transitionend', () => this.#overlay.remove(), { once: true });
  }

  inline(container, label = 'Loading…') {
    const el = document.createElement('div');
    el.className = 'loader-inline';
    el.setAttribute('role', 'status');
    el.innerHTML = `
      <div class="loader__spinner loader__spinner--sm" aria-hidden="true"></div>
      <span class="loader__label--inline">${label}</span>
    `;
    container.appendChild(el);
    return el;
  }
}

export const Loader = new LoaderManager();
