import { profileActions, applicationActions, paymentActions, profileStore, applicationStore, paymentStore } from '../state/volunteerStore.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader } from '../components/Loader.js';
import { ReputationBadge } from '../components/ReputationBadge.js';
import { ApplicationStatusBadge } from '../components/ApplicationStatusBadge.js';
import { formatSats, formatDate } from '../utils/utils.js';

export async function renderVolunteerDashboardPage(container) {
  container.innerHTML = `
    <section class="vol-dashboard container">
      <div id="dash-error"></div>
      <div id="dash-loading"></div>
    </section>
  `;

  const section    = container.querySelector('.vol-dashboard');
  const errDisplay = new APIErrorDisplay(container.querySelector('#dash-error'), {
    onRetry: () => renderVolunteerDashboardPage(container),
  });
  const spinner = Loader.inline(container.querySelector('#dash-loading'), 'Loading your dashboard…');

  try {
    await Promise.all([
      profileActions.fetchProfile(),
      applicationActions.fetchApplications(),
      paymentActions.fetchPayments(),
    ]);
    spinner.remove();
  } catch (err) {
    spinner.remove();
    errDisplay.show(err);
    return;
  }

  const { profile }      = profileStore.state;
  const { applications } = applicationStore.state;
  const { payments }     = paymentStore.state;

  const totalEarned  = payments.filter(p => p.status === 'completed').reduce((s, p) => s + (p.amount_sats || 0), 0);
  const pendingSats  = payments.filter(p => p.status === 'pending' || p.status === 'approved').reduce((s, p) => s + (p.amount_sats || 0), 0);
  const completedTasks = payments.filter(p => p.status === 'completed').length;
  const approvedApps   = applications.filter(a => a.status === 'approved').length;

  // Build active tasks from approved applications (tasks user is working on)
  const activeTasks = applications
    .filter(a => a.status === 'approved')
    .slice(0, 5);

  section.innerHTML = `
    <div class="page-intro-row">
      <div>
        <h1 class="page-intro__title">Welcome back, ${esc(profile.display_name)}</h1>
        <div style="display:flex;align-items:center;gap:var(--space-3);margin-top:var(--space-2)">
          <div id="rep-badge-container"></div>
          <span style="font-size:var(--text-sm);color:var(--color-text-muted)">Volunteer Dashboard</span>
        </div>
      </div>
      <a href="/volunteer/profile" class="btn btn--ghost btn--sm">Edit Profile</a>
    </div>

    <!-- Stat cards -->
    <div class="stat-grid stat-grid--4">
      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Earned Sats</span>
          <span class="stat-card__icon stat-card__icon--orange">₿</span>
        </div>
        <div class="stat-card__value">${formatSats(totalEarned)}</div>
        ${pendingSats > 0 ? `<div class="stat-card__sub">+${formatSats(pendingSats)} pending</div>` : ''}
      </div>

      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Completed Tasks</span>
          <span class="stat-card__icon stat-card__icon--green">✓</span>
        </div>
        <div class="stat-card__value">${completedTasks}</div>
      </div>

      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Applications</span>
          <span class="stat-card__icon stat-card__icon--blue">📄</span>
        </div>
        <div class="stat-card__value">${applications.length}</div>
        <div class="stat-card__sub">${approvedApps} approved</div>
      </div>

      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Reputation Score</span>
          <span class="stat-card__icon stat-card__icon--amber">★</span>
        </div>
        <div class="stat-card__value">${profile.reputation_score ?? 0}</div>
      </div>
    </div>

    <!-- Two-column widget row -->
    <div class="two-col-grid" style="margin-bottom:var(--space-8)">

      <!-- Active Tasks card -->
      <div class="card">
        <div class="card__header card__header--row">
          <div class="card__header-left">
            <span class="card__title">Active Tasks</span>
            <span class="card__description">Tasks you are currently working on</span>
          </div>
          <a href="/volunteer/tasks" class="btn btn--ghost btn--sm">Browse more</a>
        </div>
        <div class="card__content card__content--flush" id="active-tasks-content">
          ${activeTasks.length === 0 ? `
            <div class="empty-box--sm empty-box" style="margin:var(--space-4)">
              <div class="empty-box__icon">⚡</div>
              <div class="empty-box__title">No active tasks</div>
              <p class="empty-box__desc">Apply for tasks to get started.</p>
              <a href="/volunteer/tasks" class="btn btn--primary btn--sm">Find Tasks</a>
            </div>
          ` : activeTasks.map(a => `
            <div class="item-row" id="active-task-${esc(a.id)}">
              <div>
                <a class="item-row__title" href="/volunteer/tasks/${esc(a.task_slug)}">${esc(a.task_slug)}</a>
                <div class="item-row__sub">
                  <span class="item-row__meta-pill">${esc(a.status)}</span>
                </div>
              </div>
              <a href="/volunteer/tasks/${esc(a.task_slug)}" class="btn btn--ghost btn--sm">View</a>
            </div>
          `).join('')}
        </div>
      </div>

      <!-- Recent Applications card -->
      <div class="card">
        <div class="card__header card__header--row">
          <div class="card__header-left">
            <span class="card__title">Recent Applications</span>
            <span class="card__description">Status of your recent applications</span>
          </div>
          ${applications.length > 5 ? `<a href="/volunteer/applications" class="btn btn--ghost btn--sm">View all</a>` : ''}
        </div>
        <div class="card__content card__content--flush">
          ${applications.length === 0 ? `
            <div class="empty-box--sm empty-box" style="margin:var(--space-4)">
              <div class="empty-box__icon">📄</div>
              <div class="empty-box__title">No applications</div>
              <p class="empty-box__desc">You haven't applied to any tasks yet.</p>
            </div>
          ` : applications.slice(0, 5).map(a => `
            <div class="item-row">
              <div>
                <a class="item-row__title" href="/volunteer/tasks/${esc(a.task_slug)}">${esc(a.task_slug)}</a>
                <div class="item-row__sub">
                  <time>${formatDate(a.applied_at)}</time>
                </div>
              </div>
              <div id="app-badge-${a.id}"></div>
            </div>
          `).join('')}
        </div>
      </div>

    </div>

    <!-- Quick actions -->
    <div class="card" style="margin-bottom:var(--space-8)">
      <div class="card__header">
        <span class="card__title">Quick Actions</span>
      </div>
      <div class="card__content" style="display:flex;gap:var(--space-3);flex-wrap:wrap">
        <a class="btn btn--primary" href="/volunteer/tasks">Browse tasks</a>
        <a class="btn btn--ghost"   href="/volunteer/profile">Edit profile</a>
        <a class="btn btn--ghost"   href="/volunteer/payments">Payment history</a>
        <a class="btn btn--ghost"   href="/volunteer/reputation">My reputation</a>
      </div>
    </div>
  `;

  // Mount rep badge
  const repBadge = new ReputationBadge({ score: profile.reputation_score });
  section.querySelector('#rep-badge-container').appendChild(repBadge.element);

  // Mount application status badges
  applications.slice(0, 5).forEach(a => {
    const badgeEl = section.querySelector(`#app-badge-${a.id}`);
    if (badgeEl) {
      const badge = new ApplicationStatusBadge(a.status);
      badgeEl.appendChild(badge.element);
    }
  });
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
