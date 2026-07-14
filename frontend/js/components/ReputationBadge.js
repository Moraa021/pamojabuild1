const TIERS = [
  { min: 0,    label: 'New',      cls: 'rep--new'      },
  { min: 100,  label: 'Active',   cls: 'rep--active'   },
  { min: 500,  label: 'Trusted',  cls: 'rep--trusted'  },
  { min: 2000, label: 'Expert',   cls: 'rep--expert'   },
  { min: 5000, label: 'Elite',    cls: 'rep--elite'    },
];

function getTier(score) {
  return [...TIERS].reverse().find(t => score >= t.min) || TIERS[0];
}

export class ReputationBadge {
  #score;
  element;

  constructor({ score = 0 } = {}) {
    this.#score  = score;
    this.element = this.#render();
  }

  #render() {
    const tier = getTier(this.#score);
    const el   = document.createElement('div');
    el.className = `rep-badge ${tier.cls}`;
    el.setAttribute('aria-label', `Reputation score: ${this.#score} — ${tier.label}`);
    el.innerHTML = `
      <span class="rep-badge__score">${this.#score.toLocaleString()}</span>
      <span class="rep-badge__label">${tier.label}</span>
    `;
    return el;
  }

  update(score) {
    this.#score = score;
    const updated = this.#render();
    this.element.replaceWith(updated);
    this.element = updated;
  }
}
