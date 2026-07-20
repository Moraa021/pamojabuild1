import { taskBrowserActions, taskBrowserStore } from '../state/volunteerStore.js';
import { TaskCard } from '../components/TaskCard.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { navigate } from '../utils/utils.js';

export async function renderVolunteerTaskBrowserPage(container) {
  container.innerHTML = `
    <section class="task-browser container">
      <div class="page-intro">
        <h1 class="page-intro__title">Browse Tasks</h1>
        <p class="page-intro__desc">Find volunteer opportunities that match your skills and location.</p>
      </div>

      <div id="browser-error"></div>

      <!-- Filter bar -->
      <div class="filter-bar" id="filter-bar">
        <div class="search-group" style="flex:1;min-width:200px">
          <span class="search-group__icon" aria-hidden="true">🔍</span>
          <input
            id="task-search"
            class="search-group__input"
            type="search"
            placeholder="Search campaigns…"
            aria-label="Search tasks"
          />
        </div>
        <select id="task-status-filter" class="filter-bar__select" aria-label="Filter by status">
          <option value="">All Statuses</option>
          <option value="open">Open</option>
          <option value="in_progress">In Progress</option>
          <option value="pending_verification">Pending Verification</option>
          <option value="completed">Completed</option>
        </select>
        <select id="task-layer-filter" class="filter-bar__select" aria-label="Filter by layer">
          <option value="">All Layers</option>
          <option value="L1">L1 (On-chain)</option>
          <option value="L2">L2 (Lightning)</option>
        </select>
      </div>

      <!-- Loading skeletons -->
      <div id="browser-loading">
        <div class="task-grid" aria-hidden="true">
          ${Array.from({ length: 6 }).map(() => `
            <div class="skeleton-card">
              <div style="display:flex;justify-content:space-between;margin-bottom:var(--space-2)">
                <div class="skeleton skeleton--badge"></div>
                <div class="skeleton skeleton--badge"></div>
              </div>
              <div class="skeleton skeleton--title" style="margin-bottom:var(--space-2)"></div>
              <div class="skeleton skeleton--text" style="width:50%;margin-bottom:var(--space-6)"></div>
              <div class="skeleton" style="height:48px;width:100%;margin-top:auto"></div>
            </div>
          `).join('')}
        </div>
      </div>

      <div id="task-grid" class="task-grid" role="list" aria-label="Available tasks"></div>
    </section>
  `;

  const errDisplay = new APIErrorDisplay(container.querySelector('#browser-error'), {
    onRetry: () => renderVolunteerTaskBrowserPage(container),
  });

  const loadingEl = container.querySelector('#browser-loading');
  const searchInput   = container.querySelector('#task-search');
  const statusFilter  = container.querySelector('#task-status-filter');
  const layerFilter   = container.querySelector('#task-layer-filter');

  function getFilters() {
    return {
      status: statusFilter.value || null,
      layer:  layerFilter.value  || null,
      search: searchInput.value.trim().toLowerCase() || null,
    };
  }

  // Wire filter controls
  const onChange = () => {
    taskBrowserActions.setFilter(getFilters());
    renderCards();
  };

  searchInput.addEventListener('input', onChange);
  statusFilter.addEventListener('change', onChange);
  layerFilter.addEventListener('change', onChange);

  try {
    await taskBrowserActions.fetchTasks();
    loadingEl.remove();
  } catch (err) {
    loadingEl.remove();
    errDisplay.show(err);
    return;
  }

  function renderCards() {
    const grid  = container.querySelector('#task-grid');
    const tasks = taskBrowserActions.getFilteredTasks();
    grid.innerHTML = '';

    if (!tasks.length) {
      grid.innerHTML = `
        <div class="empty-box" style="grid-column:1/-1">
          <div class="empty-box__icon">🔍</div>
          <div class="empty-box__title">No tasks found</div>
          <p class="empty-box__desc">
            We couldn't find any tasks matching your filters. Try adjusting your search criteria.
          </p>
        </div>
      `;
      return;
    }

    tasks.forEach(task => {
      const card = new TaskCard(task, {
        onClick: () => navigate(`/volunteer/tasks/${task.slug}`),
      });
      grid.appendChild(card.element);
    });
  }

  taskBrowserStore.subscribe(() => renderCards());
  renderCards();
}
