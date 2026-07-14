import { QRCodeViewer }   from './QRCodeViewer.js';
import { formatSats }     from '../utils/utils.js';
import { truncateHex }    from '../utils/utils.js';

export class DonationCard {
  #container;
  #invoice;
  #amountSats;
  #qrViewer;
  element;

  /**
   * @param {HTMLElement}                container
   * @param {import('../api/lightningApi.js').DonationInvoiceResponse} invoice
   * @param {number}                     amountSats  - echoed from request for display
   */
  constructor(container, invoice, amountSats = 0) {
    this.#container  = container;
    this.#invoice    = invoice;
    this.#amountSats = amountSats;
    this.element     = this.#render();
    this.#container.appendChild(this.element);
    this.#mountQR();
  }

  #render() {
    const inv = this.#invoice;
    const el  = document.createElement('div');
    el.className = 'donation-card';

    el.innerHTML = `
      <div class="donation-card__header">
        <span class="donation-card__label">Lightning Invoice</span>
        <span class="donation-card__amount value-sats-sm">${formatSats(this.#amountSats)}</span>
      </div>

      <div class="donation-card__qr" id="dc-qr"></div>

      <div class="donation-card__meta">
        <div class="donation-card__hash">
          <span class="donation-card__meta-label">Payment hash</span>
          <code class="mono donation-card__hash-value" title="${esc(inv.payment_hash)}">
            ${truncateHex(inv.payment_hash, 8)}
          </code>
        </div>
        <button class="btn btn--ghost btn--sm donation-card__copy"
                aria-label="Copy BOLT11 invoice to clipboard">
          Copy invoice
        </button>
      </div>
    `;

    el.querySelector('.donation-card__copy').addEventListener('click', async () => {
      const btn = el.querySelector('.donation-card__copy');
      try {
        await navigator.clipboard.writeText(inv.payment_request);
        btn.textContent = 'Copied ✓';
        setTimeout(() => { btn.textContent = 'Copy invoice'; }, 2500);
      } catch {
        btn.textContent = 'Copy failed';
      }
    });

    return el;
  }

  async #mountQR() {
    const qrContainer = this.element.querySelector('#dc-qr');
    this.#qrViewer    = new QRCodeViewer(qrContainer);
    await this.#qrViewer.render(this.#invoice);
  }

  destroy() {
    this.#qrViewer?.destroy();
    this.element.remove();
  }
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
