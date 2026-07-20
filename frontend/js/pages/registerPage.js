import { ENV }             from '../config/env.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { authActions }     from '../state/authStore.js';
import { Toast }           from '../components/Toast.js';
import { navigate, isValidPhone } from '../utils/utils.js';
import { ROLES }           from '../config/roles.js';

const AUTH_REGISTER_URL = `${ENV.API_BASE_URL}/api/${ENV.API_VERSION}/auth/register`;

const ROLE_OPTIONS = [
  {
    value: ROLES.VOLUNTEER,
    label: 'Volunteer',
    desc:  'Complete tasks to earn sats',
  },
  {
    value: ROLES.DONOR,
    label: 'Donor',
    desc:  'Fund campaigns via Lightning',
  },
  {
    value: ROLES.CREATOR,
    label: 'Campaign Creator',
    desc:  'Organize and create volunteer tasks',
  },
  {
    value: ROLES.TRUSTEE,
    label: 'Trustee',
    desc:  'Verify work & co-sign payouts',
  },
];

export function renderRegisterPage(container) {
  const roleOptionsHTML = ROLE_OPTIONS.map(opt => `
    <label class="role-option" for="role-${opt.value}">
      <input
        type="radio"
        id="role-${opt.value}"
        name="role"
        value="${opt.value}"
        ${opt.value === ROLES.VOLUNTEER ? 'checked' : ''}
      />
      <span class="role-option__text">
        <strong>${opt.label}</strong>
        ${opt.desc}
      </span>
    </label>
  `).join('');

  container.innerHTML = `
    <section class="auth-page">
      <div class="auth-card" role="main">

        <div class="auth-card__header">
          <div class="auth-card__logo-icon" aria-hidden="true">⚡</div>
          <h1 class="auth-card__title">Create an Account</h1>
          <p class="auth-card__sub">Join PamojaBuild to participate in campaigns.</p>
        </div>

        <div class="auth-card__body">
          <div id="register-error"></div>

          <form id="register-form" novalidate aria-label="Registration form">

            <div class="form-field">
              <label for="field-display-name">Display name <span aria-hidden="true">*</span></label>
              <input id="field-display-name" name="display_name" type="text" required
                     autocomplete="name" placeholder="Your public name" />
              <div class="form-field__error" role="alert" aria-live="polite"></div>
            </div>

            <div class="form-field">
              <label for="field-phone">Phone number <span aria-hidden="true">*</span></label>
              <input id="field-phone" name="phone_number" type="tel" required
                     autocomplete="tel" placeholder="+254 700 000 000" />
              <div class="form-field__error" role="alert" aria-live="polite"></div>
            </div>

            <div class="form-field">
              <label for="field-password">Password <span aria-hidden="true">*</span></label>
              <input id="field-password" name="password" type="password" required
                     autocomplete="new-password" placeholder="At least 8 characters" minlength="8" />
              <div class="form-field__error" role="alert" aria-live="polite"></div>
            </div>

            <div class="form-field">
              <label for="field-confirm-password">Confirm password <span aria-hidden="true">*</span></label>
              <input id="field-confirm-password" name="confirm_password" type="password" required
                     autocomplete="new-password" placeholder="Repeat your password" />
              <div class="form-field__error" role="alert" aria-live="polite"></div>
            </div>

            <div class="form-field" style="margin-bottom:var(--space-2)">
              <label>I want to join as a… <span aria-hidden="true">*</span></label>
              <div class="role-picker" id="role-picker" role="radiogroup" aria-label="Choose your role">
                ${roleOptionsHTML}
              </div>
              <div class="form-field__error" role="alert" aria-live="polite" id="role-error"></div>
            </div>

            <button type="submit" class="btn btn--primary btn--full" id="register-btn" style="margin-top:var(--space-4)">
              <span class="btn__label">Create Account</span>
              <span class="btn__loading" hidden>Creating account…</span>
            </button>
          </form>
        </div>

        <div class="auth-card__footer">
          Already have an account? <a href="/signin">Sign In</a>
        </div>

      </div>
    </section>
  `;

  const form       = container.querySelector('#register-form');
  const submitBtn  = container.querySelector('#register-btn');
  const errDisplay = new APIErrorDisplay(container.querySelector('#register-error'));

  // Highlight selected role option on change
  form.querySelectorAll('input[name="role"]').forEach(radio => {
    radio.addEventListener('change', () => {
      form.querySelectorAll('.role-option').forEach(el => el.classList.remove('role-option--selected'));
      radio.closest('.role-option')?.classList.add('role-option--selected');
    });
  });
  // Set initial highlight
  const defaultRadio = form.querySelector('input[name="role"]:checked');
  defaultRadio?.closest('.role-option')?.classList.add('role-option--selected');

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errDisplay.clear();

    const data            = new FormData(form);
    const display_name    = data.get('display_name').trim();
    const phone_number    = data.get('phone_number').trim();
    const password        = data.get('password');
    const confirmPassword = data.get('confirm_password');
    const role            = data.get('role');

    let valid = true;

    if (!display_name) {
      setFieldError(form, 'display_name', 'Display name is required.');
      valid = false;
    }
    if (!phone_number || !isValidPhone(phone_number)) {
      setFieldError(form, 'phone_number', 'A valid phone number is required (e.g. +254 700 000 000).');
      valid = false;
    }
    if (!password || password.length < 8) {
      setFieldError(form, 'password', 'Password must be at least 8 characters.');
      valid = false;
    }
    if (password !== confirmPassword) {
      setFieldError(form, 'confirm_password', 'Passwords do not match.');
      valid = false;
    }
    if (!role) {
      container.querySelector('#role-error').textContent = 'Please select a role.';
      valid = false;
    }
    if (!valid) return;

    setLoading(submitBtn, true);

    try {
      const res = await fetch(AUTH_REGISTER_URL, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ display_name, phone_number, password, role }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.message || `Registration failed (${res.status})`);
      }

      const { token, user_id } = await res.json();
      authActions.setSession({ token, userId: user_id, role, name: display_name });

      Toast.show({ message: 'Account created! Welcome aboard.', type: 'success' });

      const redirectMap = {
        volunteer: '/volunteer/dashboard',
        creator:   '/campaigns/new',
        trustee:   '/trustee/register',
        donor:     '/',
      };
      navigate(redirectMap[role] || '/');

    } catch (err) {
      errDisplay.show(err);
      setLoading(submitBtn, false);
    }
  });
}

function setFieldError(form, name, msg) {
  const field = form.querySelector(`[name="${name}"]`);
  const errEl = field?.closest('.form-field')?.querySelector('.form-field__error');
  if (errEl) errEl.textContent = msg;
  field?.setAttribute('aria-invalid', 'true');
}

function setLoading(btn, loading) {
  btn.disabled = loading;
  btn.querySelector('.btn__label').hidden  = loading;
  btn.querySelector('.btn__loading').hidden = !loading;
}