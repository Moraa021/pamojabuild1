const STATUS_CONFIG = {
  pending:  { label: 'Pending',  cls: 'app-badge--pending'  },
  approved: { label: 'Approved', cls: 'app-badge--approved' },
  rejected: { label: 'Rejected', cls: 'app-badge--rejected' },
};

export class ApplicationStatusBadge {
  #status;
  element;

  constructor(status) {
    this.#status = status;
    this.element = this.#render();
  }

  #render() {
    const cfg = STATUS_CONFIG[this.#status] || { label: this.#status, cls: 'app-badge--unknown' };
    const el  = document.createElement('span');
    el.className = `app-badge ${cfg.cls}`;
    el.textContent = cfg.label;
    el.setAttribute('aria-label', `Application status: ${cfg.label}`);
    return el;
  }
}
