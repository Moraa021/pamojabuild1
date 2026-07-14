import { ReputationBadge } from './ReputationBadge.js';

export class VolunteerCard {
  #volunteer;
  #onClick;

  constructor(volunteer, { onClick } = {}) {
    this.#volunteer = volunteer;
    this.#onClick   = onClick || null;
    this.element    = this.#render();
  }

  #render() {
    const v  = this.#volunteer;
    const el = document.createElement('article');
    el.className = 'volunteer-card';

    el.innerHTML = `
      <div class="volunteer-card__avatar" aria-hidden="true">
        ${esc(v.display_name?.[0]?.toUpperCase() || '?')}
      </div>
      <div class="volunteer-card__body">
        <h3 class="volunteer-card__name">${esc(v.display_name)}</h3>
        <p class="volunteer-card__bio">${esc(v.bio || 'No bio provided.')}</p>
        <div class="volunteer-card__footer" id="rep-${v.id}"></div>
      </div>
    `;

    const repBadge = new ReputationBadge({ score: v.reputation_score });
    el.querySelector(`#rep-${v.id}`).appendChild(repBadge.element);

    if (this.#onClick) {
      el.setAttribute('tabindex', '0');
      el.setAttribute('role', 'button');
      el.addEventListener('click', () => this.#onClick(v));
      el.addEventListener('keydown', e => {
        if (e.key === 'Enter' || e.key === ' ') this.#onClick(v);
      });
    }

    return el;
  }
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
