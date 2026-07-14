export class Modal {
  #opts;
  #overlay;
  #dialog;
  #previouslyFocused;

  constructor(opts = {}) {
    this.#opts = {
      title:        opts.title        || '',
      content:      opts.content      || null,
      onConfirm:    opts.onConfirm    || null,
      onClose:      opts.onClose      || null,
      confirmLabel: opts.confirmLabel || 'Confirm',
      cancelLabel:  opts.cancelLabel  || 'Cancel',
      destructive:  opts.destructive  || false,
      hideCancel:   opts.hideCancel   || false,
    };

    this.#build();
  }

  #build() {
    this.#overlay = document.createElement('div');
    this.#overlay.className = 'modal-overlay';
    this.#overlay.setAttribute('aria-hidden', 'true');

    this.#dialog = document.createElement('div');
    this.#dialog.className = 'modal';
    this.#dialog.setAttribute('role', 'dialog');
    this.#dialog.setAttribute('aria-modal', 'true');
    this.#dialog.setAttribute('aria-labelledby', 'modal-title');

    this.#dialog.innerHTML = `
      <header class="modal__header">
        <h2 class="modal__title" id="modal-title"></h2>
        <button class="modal__close-btn" aria-label="Close dialog">✕</button>
      </header>
      <div class="modal__body"></div>
      <footer class="modal__footer"></footer>
    `;

    // Populate
    this.#dialog.querySelector('.modal__title').textContent = this.#opts.title;

    const body = this.#dialog.querySelector('.modal__body');
    if (this.#opts.content instanceof HTMLElement) {
      body.appendChild(this.#opts.content);
    } else if (typeof this.#opts.content === 'string') {
      body.innerHTML = this.#opts.content;
    }

    // Footer buttons
    const footer = this.#dialog.querySelector('.modal__footer');
    if (!this.#opts.hideCancel) {
      const cancelBtn = document.createElement('button');
      cancelBtn.className = 'btn btn--ghost';
      cancelBtn.textContent = this.#opts.cancelLabel;
      cancelBtn.addEventListener('click', () => this.close());
      footer.appendChild(cancelBtn);
    }
    if (this.#opts.onConfirm) {
      const confirmBtn = document.createElement('button');
      confirmBtn.className = `btn ${this.#opts.destructive ? 'btn--danger' : 'btn--primary'}`;
      confirmBtn.textContent = this.#opts.confirmLabel;
      confirmBtn.addEventListener('click', async () => {
        confirmBtn.disabled = true;
        try {
          await this.#opts.onConfirm();
          this.close();
        } catch {
          confirmBtn.disabled = false;
        }
      });
      footer.appendChild(confirmBtn);
    }

    // Close handlers
    this.#dialog.querySelector('.modal__close-btn').addEventListener('click', () => this.close());
    this.#overlay.addEventListener('click', () => this.close());

    // Trap focus
    this.#dialog.addEventListener('keydown', e => {
      if (e.key === 'Escape') this.close();
      if (e.key === 'Tab')    this.#trapFocus(e);
    });

    this.#overlay.appendChild(this.#dialog);
  }

  open() {
    this.#previouslyFocused = document.activeElement;
    document.body.appendChild(this.#overlay);
    document.body.style.overflow = 'hidden';

    requestAnimationFrame(() => {
      this.#overlay.classList.add('modal-overlay--visible');
      // Focus first focusable element
      const focusable = this.#dialog.querySelector('button, [href], input, select, textarea, [tabindex]:not([tabindex="-1"])');
      focusable?.focus();
    });
  }

  close() {
    this.#overlay.classList.remove('modal-overlay--visible');
    document.body.style.overflow = '';
    this.#previouslyFocused?.focus();
    this.#opts.onClose?.();
    this.#overlay.addEventListener('transitionend', () => this.#overlay.remove(), { once: true });
  }

  destroy() {
    this.#overlay.remove();
  }

  #trapFocus(e) {
    const focusable = Array.from(
      this.#dialog.querySelectorAll('button:not([disabled]), [href], input, select, textarea, [tabindex]:not([tabindex="-1"])')
    );
    if (!focusable.length) return;
    const first = focusable[0];
    const last  = focusable[focusable.length - 1];
    if (e.shiftKey) {
      if (document.activeElement === first) { e.preventDefault(); last.focus(); }
    } else {
      if (document.activeElement === last) { e.preventDefault(); first.focus(); }
    }
  }
}
