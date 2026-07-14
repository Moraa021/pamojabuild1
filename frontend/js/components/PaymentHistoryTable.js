import { formatSats, formatDate, truncateHex } from '../utils/utils.js';

const PAYMENT_STEPS = ['pending', 'approved', 'signed', 'broadcast', 'completed'];

function stepIndex(status) {
  const idx = PAYMENT_STEPS.indexOf(status);
  return idx === -1 ? 0 : idx;
}

export class PaymentHistoryTable {
  #container;
  #payments;
  element;

  constructor(container, payments = []) {
    this.#container = container;
    this.#payments  = payments;
    this.element    = this.#render();
    this.#container.appendChild(this.element);
  }

  #render() {
    const el = document.createElement('div');
    el.className = 'payment-table-wrap';

    if (!this.#payments.length) {
      el.innerHTML = '<p class="payment-table__empty">No payment history yet.</p>';
      return el;
    }

    el.innerHTML = `
      <table class="payment-table" role="table" aria-label="Payment history">
        <thead>
          <tr>
            <th scope="col">Campaign</th>
            <th scope="col">Amount</th>
            <th scope="col">Status</th>
            <th scope="col">Progress</th>
            <th scope="col">Date</th>
            <th scope="col">Hash</th>
          </tr>
        </thead>
        <tbody>
          ${this.#payments.map(p => this.#row(p)).join('')}
        </tbody>
      </table>
    `;

    return el;
  }

  #row(p) {
    const current = stepIndex(p.status);
    const steps   = PAYMENT_STEPS.map((s, i) => `
      <li class="pstep ${i <= current ? 'pstep--done' : ''}" aria-label="${s}${i === current ? ' (current)' : ''}">
        <span class="pstep__dot"></span>
        <span class="pstep__label">${s}</span>
      </li>
    `).join('');

    return `
      <tr>
        <td><code class="mono">${esc(p.task_slug)}</code></td>
        <td class="value-sats">${formatSats(p.amount_sats)}</td>
        <td><span class="badge badge--payment-${p.status}">${esc(p.status)}</span></td>
        <td>
          <ol class="pstep-track" aria-label="Payment progress">
            ${steps}
          </ol>
        </td>
        <td><time datetime="${esc(p.paid_at)}">${p.paid_at ? formatDate(p.paid_at) : '—'}</time></td>
        <td><code class="mono" title="${esc(p.payment_hash)}">${p.payment_hash ? truncateHex(p.payment_hash, 6) : '—'}</code></td>
      </tr>
    `;
  }

  update(payments) {
    this.#payments = payments;
    const updated = this.#render();
    this.element.replaceWith(updated);
    this.element = updated;
  }
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
