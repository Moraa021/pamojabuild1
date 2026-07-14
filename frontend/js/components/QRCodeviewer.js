import { formatDate, secondsUntil } from '../utils/utils.js';

const QRCODE_CDN = 'https://cdnjs.cloudflare.com/ajax/libs/qrcodejs/1.0.0/qrcode.min.js';

async function loadQRLib() {
  if (window.QRCode) return;
  return new Promise((resolve, reject) => {
    const script = document.createElement('script');
    script.src = QRCODE_CDN;
    script.onload  = resolve;
    script.onerror = () => reject(new Error('Failed to load QR code library'));
    document.head.appendChild(script);
  });
}

export class QRCodeViewer {
  #container;
  #qrEl;
  #countdownEl;
  #copyBtn;
  #countdownTimer;

  constructor(container) {
    this.#container = container;
  }

  async render(invoice) {
    await loadQRLib();

    this.destroy();

    const wrapper = document.createElement('div');
    wrapper.className = 'qr-viewer';

    wrapper.innerHTML = `
      <div class="qr-viewer__canvas" aria-label="QR code for Lightning payment"></div>
      <div class="qr-viewer__meta">
        <p class="qr-viewer__expires">
          Expires: <time class="qr-viewer__countdown" datetime="${new Date(invoice.expires_at * 1000).toISOString()}"></time>
        </p>
        <div class="qr-viewer__hash">
          <span class="label">Payment hash</span>
          <code class="mono">${invoice.payment_hash}</code>
        </div>
        <button class="btn btn--ghost qr-viewer__copy" aria-label="Copy BOLT11 invoice to clipboard">
          Copy invoice
        </button>
      </div>
    `;

    this.#container.appendChild(wrapper);

    // Render QR code
    this.#qrEl = new window.QRCode(wrapper.querySelector('.qr-viewer__canvas'), {
      text:          invoice.payment_request,
      width:         240,
      height:        240,
      colorDark:     '#0a0e1a',
      colorLight:    '#ffffff',
      correctLevel:  window.QRCode.CorrectLevel.M,
    });

    this.#countdownEl = wrapper.querySelector('.qr-viewer__countdown');
    this.#copyBtn     = wrapper.querySelector('.qr-viewer__copy');

    this.#copyBtn.addEventListener('click', () => this.#copyInvoice(invoice.payment_request));

    this.startCountdown(invoice.expires_at);
  }

  /** Start a live countdown to the invoice expiry */
  startCountdown(expiresAt) {
    const update = () => {
      if (!this.#countdownEl) return;
      const secs = secondsUntil(expiresAt);
      if (secs <= 0) {
        this.#countdownEl.textContent = 'Expired';
        this.#countdownEl.closest('.qr-viewer')?.classList.add('qr-viewer--expired');
        clearInterval(this.#countdownTimer);
        return;
      }
      const m = Math.floor(secs / 60);
      const s = String(secs % 60).padStart(2, '0');
      this.#countdownEl.textContent = `${m}:${s}`;
    };
    update();
    this.#countdownTimer = setInterval(update, 1000);
  }

  async #copyInvoice(paymentRequest) {
    try {
      await navigator.clipboard.writeText(paymentRequest);
      this.#copyBtn.textContent = 'Copied ✓';
      setTimeout(() => { if (this.#copyBtn) this.#copyBtn.textContent = 'Copy invoice'; }, 2500);
    } catch {
      this.#copyBtn.textContent = 'Copy failed';
    }
  }

  /** Clean up timers and DOM */
  destroy() {
    clearInterval(this.#countdownTimer);
    this.#container.innerHTML = '';
    this.#qrEl          = null;
    this.#countdownEl   = null;
    this.#copyBtn       = null;
  }
}
