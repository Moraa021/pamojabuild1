import authStore, { authActions } from '../state/authStore.js';
import { navigate }               from '../utils/utils.js';
import { ROLES }                  from '../config/roles.js';

const ICON_MAP = {
  Browse:       '🔍',
  Dashboard:    '⬛',
  Applications: '📄',
  Payments:     '💳',
  Profile:      '👤',
  Campaigns:    '📋',
  Create:       '＋',
  Payout:       '✅',
  Register:     '🔑',
  Admin:        '⚙',
  Donations:    '💰',
  Tasks:        '🗂',
};

function getIcon(label) {
  const word = label.split(' ')[0];
  return ICON_MAP[word] || '';
}

export class Navbar {
  #opts;
  element;
  #unsub;

  constructor(opts = {}) {
    this.#opts = {
      logoText:    opts.logoText    || 'PamojaBuild',
      links:       opts.links       || [],
      role:        opts.role        || ROLES.GUEST,
      name:        opts.name        || null,
      onLogoClick: opts.onLogoClick || (() => navigate('/')),
    };
    this.element = this.#render();
    this.#unsub  = authStore.subscribe(() => this.#refresh());
  }

  #currentPath() {
    return window.location.pathname;
  }

  #render() {
    const nav = document.createElement('nav');
    nav.className = 'navbar';
    nav.setAttribute('aria-label', 'Main navigation');

    nav.innerHTML = `
      <div class="navbar__inner container">
        <a class="navbar__logo" href="/" aria-label="Go to home page">
          <span class="navbar__logo-badge" aria-hidden="true">⚡</span>
          <span class="navbar__logo-text"></span>
        </a>
        <button class="navbar__hamburger" aria-label="Open navigation menu" aria-expanded="false" aria-controls="navbar-links">
          <span></span><span></span><span></span>
        </button>
        <ul class="navbar__links" id="navbar-links" role="list"></ul>
        <div class="navbar__auth"></div>
      </div>
    `;

    nav.querySelector('.navbar__logo-text').textContent = this.#opts.logoText;
    this.#renderLinks(nav);
    this.#bindHamburger(nav);
    this.#updateAuthControls(nav);
    return nav;
  }

  #renderLinks(nav) {
    const path     = this.#currentPath();
    const linksEl  = nav.querySelector('.navbar__links');
    linksEl.innerHTML = '';

    this.#opts.links.forEach(({ label, href }) => {
      const isActive = path === href || path.startsWith(href + '/');
      const li = document.createElement('li');
      li.innerHTML = `
        <a class="navbar__link${isActive ? ' navbar__link--active' : ''}" href="${href}">
          <span class="navbar__link-icon" aria-hidden="true">${getIcon(label)}</span>
          ${label}
        </a>
      `;
      linksEl.appendChild(li);
    });
  }

  #bindHamburger(nav) {
    const hamburger = nav.querySelector('.navbar__hamburger');
    hamburger.addEventListener('click', () => {
      const expanded = hamburger.getAttribute('aria-expanded') === 'true';
      hamburger.setAttribute('aria-expanded', String(!expanded));
      nav.classList.toggle('navbar--open');
    });

    // Close mobile menu when a link is clicked
    nav.querySelector('.navbar__links').addEventListener('click', () => {
      hamburger.setAttribute('aria-expanded', 'false');
      nav.classList.remove('navbar--open');
    });
  }

  #updateAuthControls(nav) {
    const root   = nav || this.element;
    const authEl = root.querySelector('.navbar__auth');
    if (!authEl) return;

    const { userId, role, name } = authStore.state;
    authEl.innerHTML = '';

    if (userId) {
      const initial = name ? name.charAt(0).toUpperCase() : role?.charAt(0).toUpperCase() || 'U';

      const userBlock = document.createElement('div');
      userBlock.className = 'navbar__user';
      userBlock.innerHTML = `
        <div class="navbar__user-info">
          <div class="navbar__avatar" aria-hidden="true">${initial}</div>
          <div>
            <div class="navbar__user-name">${name || 'User'}</div>
            <div class="navbar__user-role">${role || ''}</div>
          </div>
        </div>
        ${role === ROLES.VOLUNTEER
          ? `<a class="btn btn--ghost btn--sm" href="/volunteer/profile">Profile</a>`
          : ''}
      `;

      const signOut = document.createElement('button');
      signOut.className   = 'btn btn--ghost btn--sm';
      signOut.textContent = 'Sign out';
      signOut.addEventListener('click', () => {
        authActions.clearSession();
        navigate('/');
      });

      authEl.appendChild(userBlock);
      authEl.appendChild(signOut);
    } else {
      const signIn = document.createElement('a');
      signIn.className   = 'btn btn--ghost btn--sm';
      signIn.href        = '/signin';
      signIn.textContent = 'Sign in';

      const register = document.createElement('a');
      register.className   = 'btn btn--primary btn--sm';
      register.href        = '/register';
      register.textContent = 'Register';

      authEl.appendChild(signIn);
      authEl.appendChild(register);
    }
  }

  #refresh() {
    // Re-render links with updated active state
    this.#renderLinks(this.element);
    this.#updateAuthControls(this.element);
  }

  destroy() {
    this.#unsub?.();
    this.element.remove();
  }
}
