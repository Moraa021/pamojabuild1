import { ROLE_NAV, ROLE_LABELS, ROLES } from '../config/roles.js';
import authStore from '../state/authStore.js';

export class Sidebar {
  #container;
  #opts;
  #unsub;
  element;

  constructor(container, opts = {}) {
    this.#container = container;
    this.#opts = {
      activeHref: opts.activeHref || window.location.pathname,
    };
    this.element = this.#render();
    this.#container.appendChild(this.element);

    // Re-render when auth/role changes
    this.#unsub = authStore.subscribe(() => {
      const updated = this.#render();
      this.element.replaceWith(updated);
      this.element = updated;
    });
  }

  #render() {
    const role  = authStore.state.role || ROLES.GUEST;
    const links = ROLE_NAV[role]        || ROLE_NAV[ROLES.GUEST];
    const label = ROLE_LABELS[role]     || 'Menu';

    const el = document.createElement('aside');
    el.className = 'sidebar';
    el.setAttribute('aria-label', `${label} navigation`);

    el.innerHTML = `
      <div class="sidebar__header">
        <span class="sidebar__role-label">${label}</span>
        <button class="sidebar__toggle" aria-label="Collapse sidebar" aria-expanded="true">
          ‹
        </button>
      </div>
      <nav class="sidebar__nav">
        <ul class="sidebar__links" role="list">
          ${links.map(({ label, href, icon }) => `
            <li class="sidebar__item">
              <a href="${href}"
                 class="sidebar__link ${this.#opts.activeHref === href ? 'sidebar__link--active' : ''}"
                 ${this.#opts.activeHref === href ? 'aria-current="page"' : ''}>
                ${icon ? `<span class="sidebar__icon" aria-hidden="true">${icon}</span>` : ''}
                <span class="sidebar__link-label">${label}</span>
              </a>
            </li>
          `).join('')}
        </ul>
      </nav>
    `;

    // Collapse/expand toggle
    const toggle = el.querySelector('.sidebar__toggle');
    toggle.addEventListener('click', () => {
      const expanded = toggle.getAttribute('aria-expanded') === 'true';
      toggle.setAttribute('aria-expanded', String(!expanded));
      toggle.textContent = expanded ? '›' : '‹';
      el.classList.toggle('sidebar--collapsed', expanded);
    });

    return el;
  }

  /** Update the active link without full re-render */
  setActive(href) {
    this.#opts.activeHref = href;
    this.element.querySelectorAll('.sidebar__link').forEach(a => {
      const isActive = a.getAttribute('href') === href;
      a.classList.toggle('sidebar__link--active', isActive);
      if (isActive) a.setAttribute('aria-current', 'page');
      else a.removeAttribute('aria-current');
    });
  }

  destroy() {
    this.#unsub?.();
    this.element.remove();
  }
}
