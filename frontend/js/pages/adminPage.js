import { taskBrowserActions, taskBrowserStore } from '../state/volunteerStore.js';
import { APIErrorDisplay } from '../components/APIErrorDisplay.js';
import { Loader }          from '../components/Loader.js';
import { formatDate }      from '../utils/utils.js';

const STATUS_ORDER = ['open', 'in_progress', 'pending_verification', 'completed'];
const FIN_ORDER    = ['ACTIVE', 'LIQUIDATING', 'READY_FOR_PAYOUT', 'SYSTEM_LOCKDOWN', 'ARCHIVED'];

export async function renderAdminPage(container) {
  container.innerHTML = `
    <section class="admin-page container">
      <div class="page-intro">
        <h1 class="page-intro__title">Admin Overview</h1>
        <p class="page-intro__desc">Platform-wide statistics and system health.</p>
      </div>
      <div id="admin-error"></div>
      <div id="admin-loading"></div>
    </section>
  `;

  const section    = container.querySelector('.admin-page');
  const errDisplay = new APIErrorDisplay(container.querySelector('#admin-error'), {
    onRetry: () => renderAdminPage(container),
  });

  // Skeleton loading
  container.querySelector('#admin-loading').innerHTML = `
    <div class="admin-stat-grid" aria-hidden="true">
      ${Array.from({length: 4}).map(() => `
        <div class="skeleton" style="height:96px;border-radius:var(--radius-lg)"></div>
      `).join('')}
    </div>
  `;

  const spinner = Loader.inline(container.querySelector('#admin-error').previousElementSibling, '');

  try {
    await taskBrowserActions.fetchTasks();
    container.querySelector('#admin-loading').remove();
  } catch (err) {
    container.querySelector('#admin-loading').remove();
    errDisplay.show(err);
    return;
  }

  const tasks = taskBrowserStore.state.tasks;

  // Aggregate counts
  const byStatus = STATUS_ORDER.reduce((acc, s) => {
    acc[s] = tasks.filter(t => t.status === s).length;
    return acc;
  }, {});

  const byFinancial = FIN_ORDER.reduce((acc, s) => {
    acc[s] = tasks.filter(t => t.financial_state === s).length;
    return acc;
  }, {});

  const completedCount = byStatus['completed'] || 0;
  const totalFunded    = tasks.reduce((s, t) => s + (t.funded_sats || 0), 0);
  const totalPaid      = tasks.filter(t => t.financial_state === 'ARCHIVED').reduce((s, t) => s + (t.funded_sats || 0), 0);

  const flaggedTasks = tasks.filter(t =>
    t.financial_state === 'SYSTEM_LOCKDOWN' || t.status === 'pending_verification'
  );

  section.innerHTML += `
    <!-- Stat cards row -->
    <div class="admin-stat-grid">
      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Total Tasks</span>
          <span class="stat-card__icon stat-card__icon--blue">⬛</span>
        </div>
        <div class="stat-card__value">${tasks.length}</div>
      </div>
      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">System TVL (Funded)</span>
          <span class="stat-card__icon stat-card__icon--green">💰</span>
        </div>
        <div class="stat-card__value" style="font-family:var(--font-mono)">${totalFunded.toLocaleString()} sats</div>
      </div>
      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Total Paid Out</span>
          <span class="stat-card__icon stat-card__icon--orange">₿</span>
        </div>
        <div class="stat-card__value" style="font-family:var(--font-mono)">${totalPaid.toLocaleString()} sats</div>
      </div>
      <div class="stat-card">
        <div class="stat-card__header">
          <span class="stat-card__label">Completed Tasks</span>
          <span class="stat-card__icon stat-card__icon--purple">✓</span>
        </div>
        <div class="stat-card__value">${completedCount}</div>
      </div>
    </div>

    <!-- Two-column breakdowns -->
    <div class="two-col-grid" style="margin-bottom:var(--space-8)">

      <!-- Tasks by Financial State -->
      <div class="card">
        <div class="card__header">
          <span class="card__title">Tasks by Financial State</span>
        </div>
        <div class="card__content">
          ${FIN_ORDER.map(s => `
            <div class="state-entry">
              <span class="state-entry__name">
                <span class="badge badge--financial badge--financial-${s}">${s.replace(/_/g,' ')}</span>
              </span>
              <span class="state-entry__count">${byFinancial[s]}</span>
            </div>
          `).join('')}
        </div>
      </div>

      <!-- Flagged Tasks -->
      <div class="card">
        <div class="card__header card__header--row">
          <div class="card__header-left">
            <span class="card__title">⚠ Flagged Tasks</span>
            <span class="card__description">Tasks requiring admin attention</span>
          </div>
        </div>
        <div class="card__content">
          ${flaggedTasks.length === 0
            ? `<p class="admin-clear" style="text-align:center;color:var(--color-text-muted);padding:var(--space-8)">No flagged tasks.</p>`
            : flaggedTasks.map(t => `
                <div class="flagged-row">
                  <div>
                    <div class="flagged-row__title">${esc(t.title || t.slug)}</div>
                    <div class="flagged-row__state">${t.financial_state.replace(/_/g,' ')}</div>
                  </div>
                  <a href="/tasks/${esc(t.slug)}" class="flagged-row__link">View</a>
                </div>
              `).join('')
          }
        </div>
      </div>

    </div>

    <!-- Status breakdown + quick actions -->
    <div class="two-col-grid" style="margin-bottom:var(--space-8)">
      <div class="card">
        <div class="card__header">
          <span class="card__title">Tasks by Status</span>
        </div>
        <div class="card__content">
          ${STATUS_ORDER.map(s => `
            <div class="state-entry">
              <span class="badge badge--status badge--status-${s}">${s.replace(/_/g,' ')}</span>
              <span class="state-entry__count">${byStatus[s]}</span>
            </div>
          `).join('')}
        </div>
      </div>

      <div class="card">
        <div class="card__header">
          <span class="card__title">Quick Actions</span>
        </div>
        <div class="card__content" style="display:flex;flex-direction:column;gap:var(--space-3)">
          <a href="/volunteer/tasks"  class="btn btn--ghost">All tasks</a>
          <a href="/trustee/payout"   class="btn btn--ghost">Payout review</a>
          <a href="/campaigns/new"    class="btn btn--ghost">Create campaign</a>
          <a href="/trustee/register" class="btn btn--ghost">Register trustee</a>
        </div>
      </div>
    </div>
  `;
}

function esc(str) {
  const d = document.createElement('div');
  d.textContent = str || '';
  return d.innerHTML;
}
