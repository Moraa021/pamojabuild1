import { truncateHex } from '../utils/utils.js';

export class TrusteeCard {
  #container;
  #trustee;
  #opts;
  element;

  constructor(container, trustee, opts = {}) {
    this.#container = container;
    this.#trustee   = trustee;
    this.#opts      = {
      showKeyDetails: opts.showKeyDetails ?? true,
      isSigned:       opts.isSigned       ?? false,
      onSelect:       opts.onSelect       || null,
    };
    this.element = this.#render();
    this.#container.appendChild(this.element);
  }

  #render() {
    const t  = this.#trustee;
    const el = document.createElement('div');
    el.className = `trustee-card ${this.#opts.isSigned ? 'trustee-card--signed' : ''}`;

    const slotLabel = `Trustee ${t.trustee_index + 1}`;

    el.innerHTML = `
      <div class="trustee-card__header">
        <div class="trustee-card__slot-badge" aria-label="${slotLabel}">
          ${t.trustee_index + 1}
        </div>
        <div class="trustee-card__identity">
          <span class="trustee-card__slot-label">${slotLabel}</span>
          ${this.#opts.isSigned
            ? '<span class="trustee-card__signed-badge">✓ Signed</span>'
            : '<span class="trustee-card__unsigned-badge">Awaiting signature</span>'
          }
        </div>
      </div>

      ${this.#opts.showKeyDetails ? `
        <dl class="trustee-card__keys">
          <div>
            <dt>xpub</dt>
            <dd class="mono trustee-card__key-value" title="${esc(t.xpub)}">
              ${truncateHex(t.xpub, 10)}
            </dd>
          </div>
          <div>
            <dt>WebCrypto pubkey</dt>
            <dd class="mono trustee-card__key-value" title="${esc(t.web_crypto_pubkey_hex)}">
              ${truncateHex(t.web_crypto_pubkey_hex, 10)}
            </dd>
          </div>
        </dl>
      ` : ''}

      ${this.#opts.onSelect && !this.#opts.isSigned ? `
        <button class="btn btn--ghost btn--sm trustee-card__select-btn">
          Select this slot
        </button>
      ` : ''}
    `;

    if (this.#opts.onSelect && !this.#opts.isSigned) {
      el.querySelector('.trustee-card__select-btn').addEventListener('click', () => {
        this.#opts.onSelect(t);
      });
    }

    return el;
  }

  markSigned() {
    this.#opts.isSigned = true;
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
