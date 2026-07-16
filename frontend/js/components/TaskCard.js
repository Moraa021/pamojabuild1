import { formatSats, formatDate } from '../utils/utils.js';

const STATUS_LABELS = {
  open:                 'Open',
  in_progress:          'In Progress',
  pending_verification: 'Pending Verification',
  completed:            'Completed',
};

const FINANCIAL_LABELS = {
  ACTIVE:           'Active',
  LIQUIDATING:      'Liquidating',
  READY_FOR_PAYOUT: 'Ready for Payout',
  SYSTEM_LOCKDOWN:  'System Lockdown',
  ARCHIVED:         'Archived',
};

export class TaskCard {
  #task;
  #onClick;

  constructor(task, { onClick } = {}) {
    this.#task    = task;
    this.#onClick = onClick || null;
    this.element  = this.#render();
  }

  #render() {
    const el = document.createElement('article');
    el.className = 'task-card';
    el.setAttribute('tabindex', '0');
    el.setAttribute('role', 'button');
    el.setAttribute('aria-label', `View campaign: ${this.#task.title}`);

    el.innerHTML = this.#html();

    if (this.#onClick) {
      el.addEventListener('click', () => this.#onClick(this.#task));
      el.addEventListener('keydown', e => {
        if (e.key === 'Enter' || e.key === ' ') this.#onClick(this.#task);
      });
    }

    return el;
  }

  #html() {
    const t = this.#task;
    const imagePart = t.image_path
      ? `<div class="task-card__image"><img src="${t.image_path}" alt="" loading="lazy" /></div>`
      : `<div class="task-card__image task-card__image--placeholder" aria-hidden="true"></div>`;

    return `
      ${imagePart}
      <div class="task-card__body">
        <header class="task-card__header">
          <span class="badge badge--category">${this.#escape(t.category)}</span>
          <span class="badge badge--status badge--status-${t.status}">${STATUS_LABELS[t.status] || t.status}</span>
        </header>
        <h3 class="task-card__title">${this.#escape(t.title)}</h3>
        <p class="task-card__description">${this.#escape(t.description)}</p>
        <footer class="task-card__footer">
          <dl class="task-card__meta">
            <div>
              <dt>Region</dt><dd>${this.#escape(t.region)}</dd>
            </div>
            <div>
              <dt>Goal</dt><dd>${t.goal_sats ? formatSats(t.goal_sats) : '—'}</dd>
            </div>
            <div>
              <dt>Volunteers</dt><dd>${t.max_volunteers > 0 ? t.max_volunteers : 'Unlimited'}</dd>
            </div>
            <div>
              <dt>Financial State</dt>
              <dd class="badge badge--financial badge--financial-${t.financial_state}">
                ${FINANCIAL_LABELS[t.financial_state] || t.financial_state}
              </dd>
            </div>
          </dl>
          <time class="task-card__date" datetime="${t.created_at}">${formatDate(t.created_at)}</time>
        </footer>
      </div>
    `;
  }

  /** Update the card with a fresh TaskResponse */
  update(task) {
    this.#task = task;
    this.element.innerHTML = this.#html();
  }

  #escape(str) {
    const d = document.createElement('div');
    d.textContent = str || '';
    return d.innerHTML;
  }
}
