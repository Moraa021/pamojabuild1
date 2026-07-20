import { taskBrowserActions, taskBrowserStore } from '../state/volunteerStore.js';
import { TaskCard }        from '../components/TaskCard.js';
import { TaskFilter }      from '../components/TaskFilter.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader }          from '../components/Loader.js';
import { navigate }        from '../utils/utils.js';

export async function renderDonorDonationsPage(container) {
  container.innerHTML = `
    <section class="donor-page container">
      <div class="page-intro-row">
        <div>
          <h1 class="page-intro__title">Fund a Campaign</h1>
          <p class="page-intro__desc">
            Browse open campaigns and donate directly via Bitcoin Lightning.
            Every payment is recorded on-chain and released through a transparent
            3-of-5 multi-sig trustee payout.
          </p>
        </div>
      </div>
      <div id="donor-error"></div>
      <div id="filter-wrap"></div>
      <div id="donor-loading">
        <div class="task-grid" aria-hidden="true">
          ${Array.from({length:6}).map(() => `<div class="skeleton-card"></div>`).join('')}
        </div>
      </div>
      <div id="donor-grid" class="task-grid" role="list" aria-label="Campaigns accepting donations"></div>
    </section>
  `;

  const errDisplay = new APIErrorDisplay(container.querySelector('#donor-error'), {
    onRetry: () => renderDonorDonationsPage(container),
  });

  // Only show tasks that are ACTIVE or LIQUIDATING — i.e. accepting donations
  new TaskFilter(container.querySelector('#filter-wrap'), {
    onChange: (filters) => {
      taskBrowserActions.setFilter(filters);
      renderCards();
    },
  });

  const spinner = Loader.inline(container.querySelector('#donor-loading'), 'Loading campaigns…');

  try {
    await taskBrowserActions.fetchTasks();
    spinner.remove();
  } catch (err) {
    spinner.remove();
    errDisplay.show(err);
    return;
  }

  // Pre-filter: only tasks that accept donations
  const donatableStates = new Set(['ACTIVE', 'LIQUIDATING']);

  function renderCards() {
    const grid  = container.querySelector('#donor-grid');
    const tasks = taskBrowserActions.getFilteredTasks()
      .filter(t => donatableStates.has(t.financial_state));

    grid.innerHTML = '';

    if (!tasks.length) {
      grid.innerHTML = `
        <div class="empty-box" style="grid-column:1/-1">
          <div class="empty-box__icon">⚡</div>
          <div class="empty-box__title">No campaigns accepting donations</div>
          <p class="empty-box__desc">No campaigns are currently accepting donations. Check back soon.</p>
        </div>
      `;
      return;
    }

    tasks.forEach(task => {
      const card = new TaskCard(task, {
        onClick: () => navigate(`/tasks/${task.slug}/donate`),
      });
      card.element.setAttribute('role', 'listitem');
      grid.appendChild(card.element);
    });
  }

  taskBrowserStore.subscribe(() => renderCards());
  renderCards();
}
