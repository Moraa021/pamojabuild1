export class SubmissionUploader {
  #container;
  #onSubmit;
  #evidenceUrls = [];
  element;

  constructor(container, { onSubmit } = {}) {
    this.#container = container;
    this.#onSubmit  = onSubmit || (() => {});
    this.element    = this.#render();
    this.#container.appendChild(this.element);
  }

  #render() {
    const el = document.createElement('div');
    el.className = 'submission-uploader';

    el.innerHTML = `
      <div id="submit-error"></div>
      <form id="submission-form" novalidate aria-label="Work submission form">
        <div class="form-field">
          <label for="sub-description">Work description <span aria-hidden="true">*</span></label>
          <textarea id="sub-description" name="description" rows="5" required
                    placeholder="Describe what you did, what was achieved, and any challenges."></textarea>
          <div class="form-field__error" role="alert" aria-live="polite"></div>
        </div>

        <div class="form-field">
          <label>Evidence URLs</label>
          <div class="url-list" id="url-list" aria-live="polite"></div>
          <div class="url-add-row">
            <input id="url-input" type="url" class="url-add-row__input"
                   placeholder="https://…" aria-label="Evidence URL" />
            <button type="button" class="btn btn--ghost btn--sm" id="add-url-btn">Add URL</button>
          </div>
          <p class="form-field__hint">Add links to photos, videos, reports, or other evidence of completed work.</p>
        </div>

        <div class="form-actions">
          <button type="submit" class="btn btn--primary" id="sub-submit-btn">
            <span class="btn__label">Submit work</span>
            <span class="btn__loading" hidden>Submitting…</span>
          </button>
        </div>
      </form>
    `;

    // URL list management
    const urlList  = el.querySelector('#url-list');
    const urlInput = el.querySelector('#url-input');
    const addUrlBtn = el.querySelector('#add-url-btn');

    const renderUrls = () => {
      urlList.innerHTML = this.#evidenceUrls.map((u, i) => `
        <div class="url-item" data-index="${i}">
          <a href="${esc(u)}" target="_blank" rel="noopener" class="url-item__link mono">${esc(u)}</a>
          <button type="button" class="url-item__remove" aria-label="Remove URL" data-index="${i}">✕</button>
        </div>
      `).join('');

      urlList.querySelectorAll('.url-item__remove').forEach(btn => {
        btn.addEventListener('click', () => {
          const idx = Number(btn.getAttribute('data-index'));
          this.#evidenceUrls.splice(idx, 1);
          renderUrls();
        });
      });
    };

    addUrlBtn.addEventListener('click', () => {
      const url = urlInput.value.trim();
      if (!url) return;
      try {
        new URL(url); // validate
        this.#evidenceUrls.push(url);
        urlInput.value = '';
        renderUrls();
      } catch {
        urlInput.setAttribute('aria-invalid', 'true');
      }
    });

    urlInput.addEventListener('input', () => urlInput.removeAttribute('aria-invalid'));

    // Form submit
    const form      = el.querySelector('#submission-form');
    const submitBtn = el.querySelector('#sub-submit-btn');

    form.addEventListener('submit', async (e) => {
      e.preventDefault();

      const descEl = form.querySelector('[name="description"]');
      const errEl  = descEl.closest('.form-field').querySelector('.form-field__error');

      if (!descEl.value.trim()) {
        errEl.textContent = 'Description is required.';
        descEl.setAttribute('aria-invalid', 'true');
        return;
      }

      errEl.textContent = '';
      descEl.removeAttribute('aria-invalid');

      submitBtn.disabled = true;
      submitBtn.querySelector('.btn__label').hidden  = true;
      submitBtn.querySelector('.btn__loading').hidden = false;

      try {
        await this.#onSubmit({
          description:   descEl.value.trim(),
          evidence_urls: [...this.#evidenceUrls],
        });
        form.reset();
        this.#evidenceUrls = [];
        renderUrls();
      } finally {
        submitBtn.disabled = false;
        submitBtn.querySelector('.btn__label').hidden  = false;
        submitBtn.querySelector('.btn__loading').hidden = true;
      }
    });

    return el;
  }
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
