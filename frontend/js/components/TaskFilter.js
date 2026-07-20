const CATEGORIES = ['environment', 'education', 'health', 'infrastructure', 'community', 'other'];
const STATUSES   = ['open', 'in_progress', 'pending_verification', 'completed'];

export class TaskFilter {
  #container;
  #onChange;
  element;

  constructor(container, { onChange } = {}) {
    this.#container = container;
    this.#onChange  = onChange || (() => {});
    this.element    = this.#render();
    this.#container.appendChild(this.element);
  }

  #render() {
    const el = document.createElement('form');
    el.className = 'task-filter';
    el.setAttribute('role', 'search');
    el.setAttribute('aria-label', 'Filter tasks');

    el.innerHTML = `
      <div class="task-filter__group">
        <label for="filter-category" class="task-filter__label">Category</label>
        <select id="filter-category" name="category" class="task-filter__select">
          <option value="">All categories</option>
          ${CATEGORIES.map(c => `<option value="${c}">${cap(c)}</option>`).join('')}
        </select>
      </div>

      <div class="task-filter__group">
        <label for="filter-region" class="task-filter__label">Region</label>
        <input id="filter-region" name="region" type="search"
               class="task-filter__input" placeholder="Search by region…" />
      </div>

      <div class="task-filter__group">
        <label for="filter-status" class="task-filter__label">Status</label>
        <select id="filter-status" name="status" class="task-filter__select">
          <option value="open">Open</option>
          ${STATUSES.map(s => `<option value="${s}">${cap(s.replace('_', ' '))}</option>`).join('')}
          <option value="">All statuses</option>
        </select>
      </div>

      <button type="button" class="btn btn--ghost btn--sm task-filter__reset">Reset</button>
    `;

    const emit = () => {
      const data = new FormData(el);
      this.#onChange({
        category: data.get('category') || '',
        region:   data.get('region')   || '',
        status:   data.get('status')   || '',
      });
    };

    el.querySelectorAll('select, input').forEach(f => f.addEventListener('input', emit));

    el.querySelector('.task-filter__reset').addEventListener('click', () => {
      el.reset();
      emit();
    });

    return el;
  }
}

function cap(str) {
  return str.charAt(0).toUpperCase() + str.slice(1);
}
