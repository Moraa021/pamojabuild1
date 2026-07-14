import { truncateHex } from '../utils/utils.js';

export class SignatureStatusTracker {
  #container;
  #threshold;
  #total;
  #el;

  constructor(container, { threshold = 3, total = 5 } = {}) {
    this.#container = container;
    this.#threshold = threshold;
    this.#total     = total;
    this.#el        = this.#build();
    this.#container.appendChild(this.#el);
  }

  #build() {
    const el = document.createElement('div');
    el.className = 'sig-tracker';
    el.setAttribute('aria-label', `Multi-signature status: 0 of ${this.#threshold} required signatures collected`);

    el.innerHTML = `
      <div class="sig-tracker__header">
        <h4 class="sig-tracker__title">Co-signature Progress</h4>
        <span class="sig-tracker__count">
          <strong class="sig-tracker__collected">0</strong> / <strong>${this.#threshold}</strong> required
        </span>
      </div>
      <ol class="sig-tracker__slots" aria-label="Trustee signature slots">
        ${Array.from({ length: this.#total }, (_, i) => `
          <li class="sig-tracker__slot" data-index="${i}" aria-label="Trustee slot ${i + 1}: unsigned">
            <span class="sig-tracker__slot-index">${i + 1}</span>
            <span class="sig-tracker__slot-key mono">Awaiting signature</span>
          </li>
        `).join('')}
      </ol>
      <div class="sig-tracker__bar" role="progressbar"
           aria-valuemin="0" aria-valuemax="${this.#threshold}" aria-valuenow="0">
        <div class="sig-tracker__bar-fill"></div>
      </div>
    `;

    return el;
  }

  update(signedKeys = []) {
    const count    = signedKeys.length;
    const slots    = this.#el.querySelectorAll('.sig-tracker__slot');
    const keyEls   = this.#el.querySelectorAll('.sig-tracker__slot-key');
    const countEl  = this.#el.querySelector('.sig-tracker__collected');
    const barFill  = this.#el.querySelector('.sig-tracker__bar-fill');
    const bar      = this.#el.querySelector('.sig-tracker__bar');

    signedKeys.forEach((key, i) => {
      if (i >= slots.length) return;
      slots[i].classList.add('sig-tracker__slot--signed');
      slots[i].setAttribute('aria-label', `Trustee slot ${i + 1}: signed`);
      keyEls[i].textContent = truncateHex(key, 6);
    });

    countEl.textContent = count;
    const pct = Math.min((count / this.#threshold) * 100, 100);
    barFill.style.width = `${pct}%`;
    bar.setAttribute('aria-valuenow', count);
    this.#el.setAttribute('aria-label', `Multi-signature status: ${count} of ${this.#threshold} required signatures collected`);
  }

  /** Mark the threshold as reached */
  setThresholdReached() {
    this.#el.classList.add('sig-tracker--complete');
    const countEl = this.#el.querySelector('.sig-tracker__collected');
    if (countEl) countEl.closest('.sig-tracker__count').insertAdjacentHTML(
      'afterend',
      '<p class="sig-tracker__threshold-badge">✓ Threshold reached — payout authorised</p>'
    );
  }
}
