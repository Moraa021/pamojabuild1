export class Router {
  #routes;
  #container;
  #notFoundHandler;

  constructor(routes, { container, notFoundHandler } = {}) {
    this.#routes          = routes;
    this.#container       = container || document.querySelector('#app');
    this.#notFoundHandler = notFoundHandler || this.#defaultNotFound;
  }

  init() {
    window.addEventListener('popstate', () => this.#resolve());
    document.addEventListener('click', (e) => {
      const link = e.target.closest('a[href]');
      if (!link) return;
      const href = link.getAttribute('href');
      if (!href || href.startsWith('http') || href.startsWith('//') || href.startsWith('mailto:')) return;
      e.preventDefault();
      this.navigate(href);
    });
    this.#resolve();
  }

  navigate(path) {
    window.history.pushState({}, '', path);
    this.#resolve();
  }

  #resolve() {
    const path = window.location.pathname;
    for (const route of this.#routes) {
      if (this.#match(route.path, path)) {
        this.#container.innerHTML = '';
        route.handler(this.#container);
        return;
      }
    }
    this.#notFoundHandler(this.#container);
  }

  #match(pattern, path) {
    const regex = new RegExp(
      '^' + pattern.replace(/:[^/]+/g, '[^/]+') + '(?:/)?$'
    );
    return regex.test(path);
  }

  #defaultNotFound(container) {
    container.innerHTML = `
      <section class="container not-found">
        <h1>Page not found</h1>
        <p>The page you are looking for does not exist.</p>
        <a class="btn btn--primary" href="/">Return home</a>
      </section>
    `;
  }
}
