import { formatSats } from '../utils/utils.js';

export class PaymentSummaryCard {
  #container;
  #payments;
  element;

  constructor(container, payments = []) {
    this.#container = container;
    this.#payments  = payments;
    this.element    = this.#render();
    this.#container.appendChild(this.element);
  }

  #compute() {
    const p = this.#payments;
    return {
      total:     p.reduce((sum, x) => sum + (x.amount_sats || 0), 0),
      completed: p.filter(x => x.status === 'completed').length,
      pending:   p.filter(x => x.status === 'pending').length,
      count:     p.length,
    };
  }

  #render() {
    const { total, completed, pending, count } = this.#compute();
    const el = document.createElement('div');
    el.className = 'payment-summary';

    el.innerHTML = `
      <dl class="payment-summary__grid">
        <div class="payment-summary__item">
          <dt>Total earned</dt>
          <dd class="value-sats">${formatSats(total)}</dd>
        </div>
        <div class="payment-summary__item">
          <dt>Completed</dt>
          <dd>${completed}</dd>
        </div>
        <div class="payment-summary__item">
          <dt>Pending</dt>
          <dd>${pending}</dd>
        </div>
        <div class="payment-summary__item">
          <dt>Total payouts</dt>
          <dd>${count}</dd>
        </div>
      </dl>
    `;

    return el;
  }

  update(payments) {
    this.#payments = payments;
    const updated = this.#render();
    this.element.replaceWith(updated);
    this.element = updated;
  }
}
