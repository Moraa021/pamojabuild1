import { applicationActions, applicationStore } from '../state/volunteerStore.js';
import { ApplicationStatusBadge } from '../components/ApplicationStatusBadge.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader } from '../components/Loader.js';
import { formatDate, navigate } from '../utils/utils.js';

export async function renderApplicationHistoryPage(container) {
  container.innerHTML = `
    <section class="app-history container">
      <div class="page-intro">
        <h1 class="page-intro__title">My Applications</h1>
        <p class="page-intro__desc">Track the status of every task you have applied for.</p>
      </div>
      <div id="app-error"></div>
      <div id="app-loading">
        <div class="card" style="padding:var(--space-4)">
          ${Array.from({length:4}).map(() => `
            <div style="display:flex;justify-content:space-between;padding:var(--space-4) 0;border-bottom:1px solid var(--color-border)">
              <div style="display:flex;flex-direction:column;gap:var(--space-2);flex:1">
                <div class="skeleton skeleton--title" style="width:40%"></div>
                <div class="skeleton skeleton--text" style="width:60%"></div>
              </div>
              <div class="skeleton skeleton--badge" style="align-self:center"></div>
            </div>
          `).join('')}
        </div>
      </div>
      <div class="card card__content--flush" id="app-list"></div>
    </section>
  `;

  const errDisplay = new APIErrorDisplay(container.querySelector('#app-error'), {
    onRetry: () => renderApplicationHistoryPage(container),
  });
  const spinner = Loader.inline(container.querySelector('#app-loading'), 'Loading applications…');

  try {
    await applicationActions.fetchApplications();
    spinner.remove();
  } catch (err) {
    spinner.remove();
    errDisplay.show(err);
    return;
  }

  const { applications } = applicationStore.state;
  const listEl = container.querySelector('#app-list');

  if (!applications.length) {
    listEl.innerHTML = `
      <div class="empty-box">
        <div class="empty-box__icon">📄</div>
        <div class="empty-box__title">No applications yet</div>
        <p class="empty-box__desc">You have not applied for any tasks yet. Browse open tasks to get started.</p>
        <a href="/volunteer/tasks" class="btn btn--primary btn--sm">Browse Tasks</a>
      </div>
    `;
    return;
  }

  applications.forEach(app => {
    const row = document.createElement('div');
    row.className = 'item-row';
    row.innerHTML = `
      <div style="flex:1">
        <button class="item-row__title app-row__slug" data-slug="${esc(app.task_slug)}">
          ${esc(app.task_slug)}
        </button>
        <div class="item-row__sub">
          <span class="item-row__meta-pill">${esc(app.message ? app.message.slice(0, 50) : '—')}</span>
          <time>${formatDate(app.applied_at)}</time>
        </div>
      </div>
      <div id="badge-${app.id}"></div>
    `;

    row.querySelector(`#badge-${app.id}`).appendChild(new ApplicationStatusBadge(app.status).element);
    row.querySelector('.app-row__slug').addEventListener('click', () => navigate(`/volunteer/tasks/${app.task_slug}`));
    listEl.appendChild(row);
  });
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
