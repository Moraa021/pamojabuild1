import authStore, { authActions } from '../state/authStore.js';
import { navigate } from '../utils/utils.js';

export class Navbar {
  #opts;
  element;
  #unsub;

  constructor(opts = {}) {
    this.#opts = {
      logoText:    opts.logoText    || 'VolunteerTasks',
      links:       opts.links       || [],
      onLogoClick: opts.onLogoClick || (() => navigate('/')),
    };
    this.element = this.#render();
    this.#unsub  = authStore.subscribe(() => this.#updateAuthControls());
  }

  #render() {
    const nav = document.createElement('nav');
    nav.className = 'navbar';
    nav.setAttribute('aria-label', 'Main navigation');

    nav.innerHTML = `
      <div class="navbar__inner container">
        <a class="navbar__logo" href="/" aria-label="Go to home page">
          <span class="navbar__logo-icon" aria-hidden="true">⚡</span>
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

    // Render links
    const linksEl = nav.querySelector('.navbar__links');
    this.#opts.links.forEach(({ label, href }) => {
      const li = document.createElement('li');
      li.innerHTML = `<a class="navbar__link" href="${href}">${label}</a>`;
      linksEl.appendChild(li);
    });

    // Hamburger toggle
    const hamburger = nav.querySelector('.navbar__hamburger');
    hamburger.addEventListener('click', () => {
      const expanded = hamburger.getAttribute('aria-expanded') === 'true';
      hamburger.setAttribute('aria-expanded', String(!expanded));
      nav.classList.toggle('navbar--open');
    });

    this.#updateAuthControls(nav);
    return nav;
  }

  #updateAuthControls(nav) {
    const root    = nav || this.element;
    const authEl  = root.querySelector('.navbar__auth');
    if (!authEl) return;

    const { userId } = authStore.state;
    authEl.innerHTML = '';

    if (userId) {
      const signOut = document.createElement('button');
      signOut.className   = 'btn btn--ghost btn--sm';
      signOut.textContent = 'Sign out';
      signOut.addEventListener('click', () => {
        authActions.clearSession();
        navigate('/');
      });
      authEl.appendChild(signOut);
    } else {
      const signIn = document.createElement('a');
      signIn.className   = 'btn btn--primary btn--sm';
      signIn.href        = '/signin';
      signIn.textContent = 'Sign in';
      authEl.appendChild(signIn);
    }
  }

  destroy() {
    this.#unsub?.();
    this.element.remove();
  }
}
