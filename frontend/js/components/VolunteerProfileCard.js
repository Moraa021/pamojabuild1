import { ReputationBadge } from './ReputationBadge.js';
import { formatSats }      from '../utils/utils.js';

export class VolunteerProfileCard {
  #container;
  #opts;
  element;

  constructor(container, opts = {}) {
    this.#container = container;
    this.#opts = {
      profile:        opts.profile        || null,
      paymentProfile: opts.paymentProfile || null,
      stats:          opts.stats          || {},
      editable:       opts.editable       || false,
    };
    this.element = this.#render();
    this.#container.appendChild(this.element);
  }

  #render() {
    const { profile, paymentProfile, stats, editable } = this.#opts;
    if (!profile) {
      const empty = document.createElement('div');
      empty.className  = 'profile-card profile-card--empty';
      empty.textContent = 'No profile loaded.';
      return empty;
    }

    const el = document.createElement('div');
    el.className = 'profile-card';

    el.innerHTML = `
      <div class="profile-card__hero">
        <div class="profile-card__avatar" aria-hidden="true">
          ${esc(profile.display_name?.[0]?.toUpperCase() || '?')}
        </div>
        <div class="profile-card__identity">
          <h2 class="profile-card__name">${esc(profile.display_name)}</h2>
          <div class="profile-card__rep" id="pc-rep-${profile.id}"></div>
          ${editable ? `<a href="/volunteer/profile" class="btn btn--ghost btn--sm profile-card__edit">Edit profile</a>` : ''}
        </div>
      </div>

      ${profile.bio ? `<p class="profile-card__bio">${esc(profile.bio)}</p>` : ''}

      <dl class="profile-card__stats">
        <div class="profile-card__stat">
          <dt>Completed tasks</dt>
          <dd>${stats.completedTasks ?? '—'}</dd>
        </div>
        <div class="profile-card__stat">
          <dt>Total earned</dt>
          <dd class="value-sats-sm">${stats.totalEarnedSats != null ? formatSats(stats.totalEarnedSats) : '—'}</dd>
        </div>
        <div class="profile-card__stat">
          <dt>Approval rate</dt>
          <dd>
            ${stats.approvalRate != null
              ? `<div class="rep-bar" role="progressbar" aria-valuenow="${stats.approvalRate}" aria-valuemin="0" aria-valuemax="100">
                   <div class="rep-bar__fill" style="width:${stats.approvalRate}%"></div>
                 </div>
                 <span>${stats.approvalRate}%</span>`
              : '—'
            }
          </dd>
        </div>
      </dl>

      ${paymentProfile ? `
        <div class="profile-card__payment">
          <h3 class="profile-card__section-title">Payment profile</h3>
          <dl class="profile-card__payment-dl">
            ${paymentProfile.lightning_address
              ? `<div><dt>Lightning address</dt><dd class="mono">${esc(paymentProfile.lightning_address)}</dd></div>`
              : ''}
            ${paymentProfile.default_invoice
              ? `<div><dt>Default invoice</dt>
                   <dd class="mono profile-card__invoice">${esc(paymentProfile.default_invoice)}</dd>
                 </div>`
              : ''}
          </dl>
        </div>
      ` : ''}
    `;

    // Mount reputation badge
    const repContainer = el.querySelector(`#pc-rep-${profile.id}`);
    if (repContainer) {
      repContainer.appendChild(new ReputationBadge({ score: profile.reputation_score }).element);
    }

    return el;
  }

  /** Re-render with updated data */
  update(opts = {}) {
    this.#opts = { ...this.#opts, ...opts };
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
