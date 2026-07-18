import { profileActions, profileStore } from '../state/volunteerStore.js';
import { ReputationBadge } from '../components/ReputationBadge.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader } from '../components/Loader.js';
import { Toast } from '../components/Toast.js';

export async function renderVolunteerProfilePage(container) {
  container.innerHTML = `
    <section class="vol-profile container">
      <div id="profile-error"></div>
      <div id="profile-loading"></div>
    </section>
  `;

  const section    = container.querySelector('.vol-profile');
  const errDisplay = new APIErrorDisplay(container.querySelector('#profile-error'), {
    onRetry: () => renderVolunteerProfilePage(container),
  });
  const spinner = Loader.inline(container.querySelector('#profile-loading'), 'Loading profile…');

  try {
    await Promise.all([
      profileActions.fetchProfile(),
      profileActions.fetchPaymentProfile(),
    ]);
    spinner.remove();
  } catch (err) {
    spinner.remove();
    errDisplay.show(err);
    return;
  }

  const { profile, paymentProfile } = profileStore.state;
  const initial = profile.display_name?.[0]?.toUpperCase() || '?';

  section.innerHTML = `
    <div class="page-intro" style="margin-bottom:var(--space-8)">
      <h1 class="page-intro__title">Volunteer Profile</h1>
      <p class="page-intro__desc">Manage your identity and skills on PamojaBuild.</p>
    </div>

    <div class="profile-layout">

      <!-- Sidebar -->
      <div>
        <!-- Avatar card -->
        <div class="card profile-avatar-card" style="margin-bottom:var(--space-6)">
          <div class="profile-avatar-card__banner"></div>
          <div class="profile-avatar-card__body">
            <div class="profile-avatar-wrap" aria-hidden="true">${esc(initial)}</div>
            <div class="profile-avatar-name">${esc(profile.display_name)}</div>
            <div id="rep-container"></div>
            <div class="profile-avatar-stats">
              <span>★</span>
              <span>${profile.completed_tasks || 0} tasks completed</span>
            </div>
          </div>
        </div>

        <!-- Reputation info -->
        <div class="card card--muted">
          <div class="card__header">
            <span class="card__title" style="font-size:var(--text-sm);display:flex;align-items:center;gap:var(--space-2)">
              🛡 Reputation System
            </span>
          </div>
          <div class="card__content rep-info-card">
            <p>Your reputation score is tied directly to your verified work.</p>
            <ul>
              <li>Completing tasks increases score</li>
              <li>Higher scores grant priority access to premium tasks</li>
              <li>Your score is cryptographically provable</li>
            </ul>
          </div>
        </div>
      </div>

      <!-- Main content -->
      <div>
        <div id="edit-error"></div>

        <!-- Public profile card -->
        <div class="card" style="margin-bottom:var(--space-6)">
          <div class="card__header">
            <span class="card__title">Edit Profile</span>
            <span class="card__description">Update how task creators see you.</span>
          </div>
          <div class="card__content">
            <form id="profile-form" novalidate aria-label="Edit volunteer profile">
              <div class="form-field">
                <label for="field-display-name">Public Name <span aria-hidden="true">*</span></label>
                <input id="field-display-name" name="display_name" type="text" required
                       value="${esc(profile.display_name)}" />
                <div class="form-field__error" role="alert" aria-live="polite"></div>
              </div>
              <div class="form-field">
                <label for="field-bio">Bio</label>
                <textarea id="field-bio" name="bio" rows="4"
                  placeholder="Tell creators about your experience and background..."
                >${esc(profile.bio || '')}</textarea>
                <p class="form-field__hint">Shown on your applications.</p>
              </div>
              <div class="form-actions" style="border-top:1px solid var(--color-border);padding-top:var(--space-5);margin-top:var(--space-5)">
                <button type="submit" class="btn btn--primary" id="profile-save-btn">
                  <span class="btn__label">Save Profile</span>
                  <span class="btn__loading" hidden>Saving…</span>
                </button>
              </div>
            </form>
          </div>
        </div>

        <!-- Payment profile card -->
        <div class="card">
          <div class="card__header">
            <span class="card__title">Payment Profile</span>
            <span class="card__description">Where to receive your Lightning payouts.</span>
          </div>
          <div class="card__content">
            <form id="payment-form" novalidate aria-label="Payment profile form">
              <div class="form-field">
                <label for="field-ln-address">Lightning Address</label>
                <input id="field-ln-address" name="lightning_address" type="text"
                       placeholder="you@wallet.example"
                       value="${esc(paymentProfile?.lightning_address || '')}" />
                <p class="form-field__hint">Used to receive Lightning payments for completed tasks.</p>
              </div>
              <div class="form-field">
                <label for="field-default-invoice">Default BOLT11 Invoice (fallback)</label>
                <textarea id="field-default-invoice" name="default_invoice" rows="3"
                          class="mono" placeholder="lnbc…">${esc(paymentProfile?.default_invoice || '')}</textarea>
              </div>
              <div class="form-actions" style="border-top:1px solid var(--color-border);padding-top:var(--space-5);margin-top:var(--space-5)">
                <button type="submit" class="btn btn--primary" id="payment-save-btn">
                  <span class="btn__label">Save Payment Profile</span>
                  <span class="btn__loading" hidden>Saving…</span>
                </button>
              </div>
            </form>
          </div>
        </div>

      </div>
    </div>
  `;

  // Rep badge
  const repBadge = new ReputationBadge({ score: profile.reputation_score });
  section.querySelector('#rep-container').appendChild(repBadge.element);

  const editErrDisplay = new APIErrorDisplay(section.querySelector('#edit-error'));

  // Profile form
  section.querySelector('#profile-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    editErrDisplay.clear();
    const data    = new FormData(e.target);
    const nameVal = data.get('display_name').trim();
    if (!nameVal) {
      e.target.querySelector('.form-field__error').textContent = 'Display name is required.';
      return;
    }
    const btn = section.querySelector('#profile-save-btn');
    setLoading(btn, true);
    try {
      await profileActions.updateProfile({ display_name: nameVal, bio: data.get('bio').trim() });
      Toast.show({ message: 'Profile updated.', type: 'success' });
    } catch (err) {
      editErrDisplay.show(err);
    } finally {
      setLoading(btn, false);
    }
  });

  // Payment form
  section.querySelector('#payment-form').addEventListener('submit', async (e) => {
    e.preventDefault();
    editErrDisplay.clear();
    const data = new FormData(e.target);
    const btn  = section.querySelector('#payment-save-btn');
    setLoading(btn, true);
    try {
      await profileActions.savePaymentProfile({
        lightning_address: data.get('lightning_address').trim(),
        default_invoice:   data.get('default_invoice').trim(),
      });
      Toast.show({ message: 'Payment profile saved.', type: 'success' });
    } catch (err) {
      editErrDisplay.show(err);
    } finally {
      setLoading(btn, false);
    }
  });
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
