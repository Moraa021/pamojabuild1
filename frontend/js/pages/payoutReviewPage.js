import payoutStore, { payoutActions } from '../state/payoutStore.js';
import authStore from '../state/authStore.js';
import trusteeStore from '../state/trusteeStore.js';
import { SignatureStatusTracker } from '../components/SignatureStatusTracker.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader } from '../components/Loader.js';
import { Toast } from '../components/Toast.js';
import { Modal } from '../components/Modal.js';
import { signWithPrivateKey, toBytes } from '../utils/crypto.js';
import { formatSats, truncateHex, navigate } from '../utils/utils.js';

// In-memory reference to the generated private key (session only)
let _sessionPrivateKey = null;

export function setSessionPrivateKey(key) { _sessionPrivateKey = key; }

export async function renderPayoutReviewPage(container) {
  const params = new URLSearchParams(window.location.search);
  const slug   = params.get('slug');

  if (!slug) {
    container.innerHTML = '<section class="container"><p>No campaign specified.</p></section>';
    return;
  }

  container.innerHTML = `
    <section class="payout-review container">
      <div class="page-intro">
        <h1 class="page-intro__title">Trustee Payout Review</h1>
        <p class="page-intro__desc">Review completed tasks and co-sign PSBTs to release funds.</p>
      </div>
      <div id="payout-error"></div>
      <div id="payout-loading"></div>
    </section>
  `;

  const section    = container.querySelector('.payout-review');
  const errDisplay = new APIErrorDisplay(container.querySelector('#payout-error'), {
    onRetry: () => renderPayoutReviewPage(container),
  });
  const spinner = Loader.inline(container.querySelector('#payout-loading'), 'Fetching payout manifest…');

  let manifest;
  try {
    manifest = await payoutActions.fetchManifest(slug);
    spinner.remove();
  } catch (err) {
    spinner.remove();
    errDisplay.show(err);
    return;
  }

  section.innerHTML = `
    <div class="payout-layout">

      <!-- Left: PSBT + signing form -->
      <div style="display:flex;flex-direction:column;gap:var(--space-6)">

        <div class="card">
          <div class="card__header">
            <span class="card__title">🔑 Partial Signed Bitcoin Transaction (PSBT)</span>
          </div>
          <div class="card__content">
            <div class="mono-block" id="psbt-value">${esc(manifest.unsigned_psbt_hex)}</div>
            <button class="btn btn--ghost btn--sm" style="margin-top:var(--space-3)" data-copy-target="psbt-value">
              Copy PSBT
            </button>
          </div>
        </div>

        <div class="card">
          <div class="card__header">
            <span class="card__title">Layer 2 Volunteer Invoice</span>
          </div>
          <div class="card__content">
            <div class="mono-block" id="invoice-value">${esc(manifest.volunteer_invoice)}</div>
            <button class="btn btn--ghost btn--sm" style="margin-top:var(--space-3)" data-copy-target="invoice-value">
              Copy Invoice
            </button>
          </div>
        </div>

        <div class="card">
          <div class="card__header">
            <span class="card__title">Submit Co-Signature</span>
          </div>
          <div class="card__content">
            <div id="sig-error"></div>
            <form id="sig-form" novalidate aria-label="Co-signature submission form">

              <div class="form-field">
                <label for="field-pubkey-hex">Your Trustee Public Key (hex) <span aria-hidden="true">*</span></label>
                <input id="field-pubkey-hex" name="trustee_public_key_hex" type="text"
                       required class="mono" placeholder="Paste your public key hex" />
                <div class="form-field__error" role="alert" aria-live="polite"></div>
              </div>

              <div class="form-field">
                <label for="field-psbt-sig">Layer 1 PSBT Signature Fragment <span aria-hidden="true">*</span></label>
                <textarea id="field-psbt-sig" name="layer1_psbt_signature_fragment" rows="3"
                          required class="mono" placeholder="Paste the PSBT signature fragment from your hardware wallet"></textarea>
                <div class="form-field__error" role="alert" aria-live="polite"></div>
              </div>

              <div class="form-field">
                <label>Layer 2 WebCrypto Signature</label>
                <div class="key-gen-panel">
                  <code id="l2-sig-display" class="mono key-gen-panel__key" aria-live="polite">Not yet signed</code>
                  <button type="button" class="btn btn--ghost" id="sign-l2-btn">Sign with browser key</button>
                </div>
                <input type="hidden" id="field-l2-sig" name="layer2_web_crypto_signature" />
                <p class="form-field__hint">
                  Signs the volunteer invoice using the browser key pair registered during trustee setup.
                </p>
              </div>

              <div class="form-actions">
                <button type="submit" class="btn btn--primary" id="sign-submit-btn">
                  <span class="btn__label">🛡 Submit Signatures</span>
                  <span class="btn__loading" hidden>Submitting…</span>
                </button>
              </div>
            </form>
          </div>
        </div>

      </div>

      <!-- Right sidebar: summary + trustee keys -->
      <div style="display:flex;flex-direction:column;gap:var(--space-6)">

        <!-- Signature tracker slot -->
        <div id="sig-tracker-container"></div>

        <!-- Payout Summary card -->
        <div class="card card--muted">
          <div class="card__header">
            <span class="card__title">Payout Summary</span>
          </div>
          <div class="card__content">
            <dl class="payout-summary-dl">
              <div>
                <dt>Campaign</dt>
                <dd><code class="mono" style="font-size:var(--text-xs)">${esc(slug)}</code></dd>
              </div>
              <div>
                <dt>Layer 1 (on-chain)</dt>
                <dd class="payout-amount">${formatSats(manifest.l1_amount_sats)}</dd>
              </div>
              <div>
                <dt>Layer 2 (Lightning)</dt>
                <dd class="payout-amount">${formatSats(manifest.l2_amount_sats)}</dd>
              </div>
            </dl>
          </div>
        </div>

        <!-- Trustee Keys card -->
        <div class="card">
          <div class="card__header">
            <span class="card__title">Trustee Keys</span>
          </div>
          <div class="card__content" id="trustee-keys-list">
            ${(manifest.trustees || []).map((t, i) => `
              <div class="trustee-key-row">
                <div style="display:flex;align-items:center;gap:var(--space-2)">
                  <div class="trustee-key-dot${t.signed ? ' trustee-key-dot--signed' : ''}"></div>
                  <code class="mono" style="font-size:var(--text-xs);color:var(--color-text-muted)">
                    ${esc((t.xpub || '').substring(0, 8))}…
                  </code>
                </div>
                <span style="font-size:var(--text-xs);color:${t.signed ? 'var(--color-success)' : 'var(--color-text-muted)'}">
                  ${t.signed ? 'Signed' : 'Waiting'}
                </span>
              </div>
            `).join('')}
          </div>
        </div>

      </div>
    </div>
  `;

  // Signature tracker
  const tracker = new SignatureStatusTracker(
    section.querySelector('#sig-tracker-container'),
    { threshold: 3, total: 5 }
  );

  // Subscribe to payout store for live updates
  const unsub = payoutStore.subscribe(state => {
    tracker.update(state.submittedSigs);
    if (state.thresholdReached) {
      tracker.setThresholdReached();
      Toast.show({ message: 'Threshold reached. Payout is authorised.', type: 'success', duration: 8000 });
      unsub();
    }
  });

  // Copy buttons
  section.querySelectorAll('[data-copy-target]').forEach(btn => {
    btn.addEventListener('click', async () => {
      const targetId = btn.getAttribute('data-copy-target');
      const text = section.querySelector(`#${targetId}`)?.textContent;
      if (!text) return;
      try {
        await navigator.clipboard.writeText(text);
        btn.textContent = 'Copied ✓';
        setTimeout(() => btn.textContent = 'Copy', 2500);
      } catch {
        btn.textContent = 'Failed';
      }
    });
  });

  // Layer 2 signing
  const signL2Btn    = section.querySelector('#sign-l2-btn');
  const l2SigDisplay = section.querySelector('#l2-sig-display');
  const l2SigHidden  = section.querySelector('#field-l2-sig');

  signL2Btn.addEventListener('click', async () => {
    if (!_sessionPrivateKey) {
      Toast.show({ message: 'No browser key found for this session. Re-register your trustee keys.', type: 'error' });
      return;
    }
    signL2Btn.disabled   = true;
    signL2Btn.textContent = 'Signing…';
    try {
      const sigHex = await signWithPrivateKey(_sessionPrivateKey, toBytes(manifest.volunteer_invoice));
      l2SigHidden.value = sigHex;
      l2SigDisplay.textContent = truncateHex(sigHex, 12);
      Toast.show({ message: 'Layer 2 signature generated.', type: 'success' });
    } catch {
      Toast.show({ message: 'Signing failed. Check your browser key.', type: 'error' });
    } finally {
      signL2Btn.disabled   = false;
      signL2Btn.textContent = 'Sign with browser key';
    }
  });

  // Form submission
  const form = section.querySelector('#sig-form');
  const sigErrDisplay = new APIErrorDisplay(section.querySelector('#sig-error'));

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    sigErrDisplay.clear();

    if (!validateSigForm(form)) return;

    const data    = new FormData(form);
    const payload = {
      trustee_public_key_hex:          data.get('trustee_public_key_hex').trim(),
      layer1_psbt_signature_fragment:  data.get('layer1_psbt_signature_fragment').trim(),
      layer2_web_crypto_signature:     data.get('layer2_web_crypto_signature').trim(),
    };

    // Confirm before submitting
    const confirmContent = document.createElement('p');
    confirmContent.textContent = `You are about to submit your co-signature for payout on campaign "${slug}". This action cannot be undone.`;

    const confirmModal = new Modal({
      title:        'Confirm signature submission',
      content:      confirmContent,
      confirmLabel: 'Submit signatures',
      onConfirm:    async () => {
        const submitBtn = section.querySelector('#sign-submit-btn');
        setLoading(submitBtn, true);
        try {
          await payoutActions.submitSignature(slug, payload);
          Toast.show({ message: 'Signatures submitted.', type: 'success' });
        } catch (err) {
          sigErrDisplay.show(err);
          setLoading(submitBtn, false);
        }
      },
    });
    confirmModal.open();
  });
}

function validateSigForm(form) {
  let valid = true;
  form.querySelectorAll('[required]').forEach(field => {
    const errEl = field.closest('.form-field')?.querySelector('.form-field__error');
    if (!field.value.trim()) {
      if (errEl) errEl.textContent = 'This field is required.';
      field.setAttribute('aria-invalid', 'true');
      valid = false;
    } else {
      if (errEl) errEl.textContent = '';
      field.removeAttribute('aria-invalid');
    }
  });
  return valid;
}

function setLoading(btn, loading) {
  btn.disabled = loading;
  btn.querySelector('.btn__label').hidden  = loading;
  btn.querySelector('.btn__loading').hidden = !loading;
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
