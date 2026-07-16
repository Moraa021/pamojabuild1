const IMAGE_EXTS = /\.(jpg|jpeg|png|gif|webp|svg)(\?.*)?$/i;

export class EvidenceGallery {
  #container;
  #submissions;
  element;

  constructor(container, submissions = []) {
    this.#container  = container;
    this.#submissions = submissions;
    this.element     = this.#render();
    this.#container.appendChild(this.element);
  }

  #render() {
    const el = document.createElement('div');
    el.className = 'evidence-gallery';

    if (!this.#submissions.length) {
      el.innerHTML = '<p class="evidence-gallery__empty">No submissions yet.</p>';
      return el;
    }

    this.#submissions.forEach(sub => {
      const card = document.createElement('div');
      card.className = 'evidence-card';

      const urlLinks = (sub.evidence_urls || []).map(url => {
        const isImage = IMAGE_EXTS.test(url);
        return isImage
          ? `<figure class="evidence-card__image-wrap">
               <img src="${esc(url)}" alt="Evidence image" loading="lazy" class="evidence-card__img" />
               <figcaption><a href="${esc(url)}" target="_blank" rel="noopener" class="evidence-card__link mono">${esc(url)}</a></figcaption>
             </figure>`
          : `<a href="${esc(url)}" target="_blank" rel="noopener" class="evidence-card__link mono">${esc(url)}</a>`;
      }).join('');

      card.innerHTML = `
        <div class="evidence-card__meta">
          <span class="badge badge--status-${sub.status}">${esc(sub.status)}</span>
          <time class="evidence-card__date" datetime="${esc(sub.submitted_at)}">${new Date(sub.submitted_at).toLocaleString()}</time>
        </div>
        <p class="evidence-card__description">${esc(sub.description)}</p>
        <div class="evidence-card__urls">
          ${urlLinks || '<p class="text-muted">No evidence URLs provided.</p>'}
        </div>
      `;

      el.appendChild(card);
    });

    return el;
  }

  update(submissions) {
    this.#submissions = submissions;
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
