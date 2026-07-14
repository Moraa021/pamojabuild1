class ToastManager {
  #container;

  constructor() {
    this.#container = this.#createContainer();
    document.body.appendChild(this.#container);
  }

  #createContainer() {
    const el = document.createElement('div');
    el.setAttribute('role', 'status');
    el.setAttribute('aria-live', 'polite');
    el.setAttribute('aria-atomic', 'false');
    el.className = 'toast-container';
    return el;
  }

  show({ message, type = 'info', duration = 4000 }) {
    const toast = document.createElement('div');
    toast.className  = `toast toast--${type}`;
    toast.setAttribute('role', type === 'error' ? 'alert' : 'status');
    toast.innerHTML = `
      <span class="toast__icon" aria-hidden="true">${this.#icon(type)}</span>
      <span class="toast__message">${this.#escape(message)}</span>
      <button class="toast__close" aria-label="Dismiss notification">✕</button>
    `;

    const dismiss = () => this.#dismiss(toast);
    toast.querySelector('.toast__close').addEventListener('click', dismiss);

    this.#container.appendChild(toast);

    // Trigger enter animation
    requestAnimationFrame(() => toast.classList.add('toast--visible'));

    const timer = setTimeout(dismiss, duration);
    toast._timer = timer;
    return toast;
  }

  #dismiss(toast) {
    clearTimeout(toast._timer);
    toast.classList.remove('toast--visible');
    toast.classList.add('toast--leaving');
    toast.addEventListener('transitionend', () => toast.remove(), { once: true });
  }

  #icon(type) {
    const icons = { success: '✓', error: '✕', warning: '⚠', info: 'ℹ' };
    return icons[type] || icons.info;
  }

  #escape(str) {
    const div = document.createElement('div');
    div.textContent = str;
    return div.innerHTML;
  }
}

export const Toast = new ToastManager();
