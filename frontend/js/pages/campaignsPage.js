import taskApi             from '../api/taskApi.js';
import { TaskCard }        from '../components/TaskCard.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader }          from '../components/Loader.js';
import { navigate }        from '../utils/utils.js';
import authStore           from '../state/authStore.js';

export async function renderCampaignsPage(container) {
  container.innerHTML = `
    <section class="campaigns-page container">
      <div class="page-intro-row">
        <div>
          <h1 class="page-intro__title">My Campaigns</h1>
          <p class="page-intro__desc">Manage your volunteer task campaigns.</p>
        </div>
        <a href="/campaigns/new" class="btn btn--primary">+ New Campaign</a>
      </div>
      <div id="campaigns-error"></div>
      <div id="campaigns-loading">
        <div class="task-grid" aria-hidden="true">
          ${Array.from({length:3}).map(() => `<div class="skeleton-card"></div>`).join('')}
        </div>
      </div>
      <div id="campaigns-grid" class="task-grid" role="list" aria-label="Your campaigns"></div>
    </section>
  `;

  const errDisplay = new APIErrorDisplay(container.querySelector('#campaigns-error'), {
    onRetry: () => renderCampaignsPage(container),
  });
  const spinner = Loader.inline(container.querySelector('#campaigns-loading'), 'Loading your campaigns…');

  let tasks = [];
  try {
    const data = await taskApi.list();
    // Backend returns { tasks: [...] } — unwrap the array
    tasks = Array.isArray(data) ? data : (data?.tasks ?? []);
    spinner.remove();
  } catch (err) {
    spinner.remove();
    errDisplay.show(err);
    return;
  }

  const userId = authStore.state.userId;
  const mine   = userId ? tasks.filter(t => t.creator_id === Number(userId)) : tasks;
  const grid   = container.querySelector('#campaigns-grid');

  if (!mine.length) {
    grid.innerHTML = `
      <div class="empty-box" style="grid-column:1/-1">
        <div class="empty-box__icon">📋</div>
        <div class="empty-box__title">No campaigns yet</div>
        <p class="empty-box__desc">You have not created any campaigns yet.</p>
        <a href="/campaigns/new" class="btn btn--primary btn--sm">Create your first campaign</a>
      </div>
    `;
    return;
  }

  mine.forEach(task => {
    const card = new TaskCard(task, {
      onClick: () => navigate(`/tasks/${task.slug}`),
    });
    card.element.setAttribute('role', 'listitem');
    grid.appendChild(card.element);
  });
}
