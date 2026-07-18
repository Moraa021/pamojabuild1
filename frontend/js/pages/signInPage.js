import { ENV }           from '../config/env.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { authActions }   from '../state/authStore.js';
import { Toast }         from '../components/Toast.js';
import { navigate, isValidPhone } from '../utils/utils.js';

const AUTH_SIGNIN_URL = `${ENV.API_BASE_URL}/api/${ENV.API_VERSION}/auth/signin`;

export function renderSignInPage(container) {
  container.innerHTML = `
    <section class="auth-page">
      <div class="auth-card" role="main">

        <div class="auth-card__header">
          <div class="auth-card__logo-icon" aria-hidden="true">⚡</div>
          <h1 class="auth-card__title">Sign In</h1>
          <p class="auth-card__sub">Enter your credentials to access your account.</p>
        </div>

        <div class="auth-card__body">
          <div id="signin-error"></div>

          <form id="signin-form" novalidate aria-label="Sign in form">
            <div class="form-field">
              <label for="field-phone">Phone number <span aria-hidden="true">*</span></label>
              <input id="field-phone" name="phone_number" type="tel" required
                     autocomplete="tel" placeholder="+254 700 000 000" />
              <div class="form-field__error" role="alert" aria-live="polite"></div>
            </div>

            <div class="form-field">
              <label for="field-password">Password <span aria-hidden="true">*</span></label>
              <input id="field-password" name="password" type="password" required
                     autocomplete="current-password" placeholder="••••••••" />
              <div class="form-field__error" role="alert" aria-live="polite"></div>
            </div>

            <button type="submit" class="btn btn--primary btn--full" id="signin-btn" style="margin-top:var(--space-2)">
              <span class="btn__label">Sign In</span>
              <span class="btn__loading" hidden>Signing in…</span>
            </button>
          </form>
        </div>

        <div class="auth-card__footer">
          Don't have an account? <a href="/register">Register</a>
        </div>

      </div>
    </section>
  `;

  const form       = container.querySelector('#signin-form');
  const submitBtn  = container.querySelector('#signin-btn');
  const errDisplay = new APIErrorDisplay(container.querySelector('#signin-error'));

  form.addEventListener('submit', async (e) => {
    e.preventDefault();
    errDisplay.clear();

    const data         = new FormData(form);
    const phone_number = data.get('phone_number').trim();
    const password     = data.get('password');

    let valid = true;
    if (!phone_number || !isValidPhone(phone_number)) {
      setFieldError(form, 'phone_number', 'A valid phone number is required.');
      valid = false;
    }
    if (!password) {
      setFieldError(form, 'password', 'Password is required.');
      valid = false;
    }
    if (!valid) return;

    setLoading(submitBtn, true);

    try {
      const res = await fetch(AUTH_SIGNIN_URL, {
        method:  'POST',
        headers: { 'Content-Type': 'application/json' },
        body:    JSON.stringify({ phone_number, password }),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.message || `Sign in failed (${res.status})`);
      }

      const { token, user_id, role, display_name } = await res.json();
      authActions.setSession({ token, userId: user_id, role, name: display_name });

      Toast.show({ message: 'Signed in successfully.', type: 'success' });

      const redirectMap = {
        volunteer: '/volunteer/dashboard',
        creator:   '/campaigns',
        trustee:   '/trustee/payout',
        admin:     '/admin',
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
