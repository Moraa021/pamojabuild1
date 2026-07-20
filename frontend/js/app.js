import { Router }                           from './router.js';
import { Navbar }                           from './components/Navbar.js';
import { Toast }                            from './components/Toast.js';
import { ApiError, NetworkError }           from './api/client.js';
import authStore                            from './state/authStore.js';
import { ROLE_NAV, ROLES }                  from './config/roles.js';

import { renderSignInPage }                 from './pages/signInPage.js';
import { renderRegisterPage }               from './pages/registerPage.js';
import { renderTaskDetailPage }             from './pages/taskDetailPage.js';
import { renderCreateCampaignPage }         from './pages/createCampaignPage.js';
import { renderCampaignsPage }              from './pages/campaignsPage.js';
import { renderDonationPage }               from './pages/donationPage.js';
import { renderTrusteeDashboardPage }       from './pages/trusteeDashboardPage.js';
import { renderPayoutReviewPage }           from './pages/payoutReviewPage.js';
import { renderAdminPage }                  from './pages/adminPage.js';
import { renderDonorDonationsPage }         from './pages/donorDonationPage.js';

import { renderVolunteerDashboardPage }     from './pages/volunteerDashboardPage.js';
import { renderVolunteerTaskBrowserPage }   from './pages/volunteerTaskBrowserPage.js';
import { renderVolunteerTaskDetailsPage }   from './pages/volunteerTaskDetailsPage.js';
import { renderVolunteerProfilePage }       from './pages/volunteerProfilePage.js';
import { renderApplicationHistoryPage }     from './pages/applicationHistoryPage.js';
import { renderSubmissionPage }             from './pages/submissionPage.js';
import { renderVolunteerPaymentsPage }      from './pages/volunteerPaymentPage.js';
import { renderReputationPage }             from './pages/reputationPage.js';

function renderHomePage(container) {
  const year = new Date().getFullYear();
  container.innerHTML = `
    <div class="page-layout">
      <main class="page-layout__content" style="padding:0">

        <!-- Hero -->
        <section class="home__hero-panel">
          <div class="home__hero-grid-bg" aria-hidden="true"></div>
          <div class="home__hero-gradient" aria-hidden="true"></div>
          <div class="home__hero-inner">
            <div class="home__hero-badge">
              ⚡ Powered by Bitcoin Lightning Network
            </div>
            <h1 class="home__headline">
              Where Community Work Meets
              <span class="home__headline-accent"> Real Money</span>
            </h1>
            <p class="home__sub">
              A trustless platform for funding volunteers. Campaigns raise sats via Lightning invoices,
              releasing funds through transparent 3-of-5 multi-sig payouts.
            </p>
            <div class="home__actions">
              <a href="/volunteer/tasks" class="btn btn--primary btn--lg">
                Browse Tasks →
              </a>
              <a href="/campaigns/new" class="btn btn--ghost btn--lg">
                Create Campaign
              </a>
            </div>
          </div>
        </section>

        <!-- How it works -->
        <section class="home__how">
          <div class="container">
            <div class="home__how-header">
              <h2 class="home__how-title">A precise, trustless workflow</h2>
              <p class="home__how-sub">No middlemen. Code is law. Volunteer work verified by cryptography.</p>
            </div>
            <div class="home__how-grid">
              <div class="home__how-card">
                <div class="home__how-icon home__how-icon--blue" aria-hidden="true">₿</div>
                <h3 class="home__how-card-title">1. Fund via Lightning</h3>
                <p class="home__how-card-desc">
                  Anyone can donate sats to an open campaign instantly and cheaply using the Lightning Network.
                </p>
              </div>
              <div class="home__how-card">
                <div class="home__how-icon home__how-icon--green" aria-hidden="true">👥</div>
                <h3 class="home__how-card-title">2. Work &amp; Verify</h3>
                <p class="home__how-card-desc">
                  Volunteers apply, work, and submit evidence. A decentralized group of trustees reviews the work.
                </p>
              </div>
              <div class="home__how-card">
                <div class="home__how-icon home__how-icon--purple" aria-hidden="true">🛡</div>
                <h3 class="home__how-card-title">3. Multi-Sig Payout</h3>
                <p class="home__how-card-desc">
                  Trustees sign a PSBT. Once 3 of 5 sign, the funds are trustlessly released to the volunteer.
                </p>
              </div>
            </div>
          </div>
        </section>

      </main>

      <footer class="site-footer">
        <div class="container site-footer__inner">
          &copy; ${year} PamojaBuild. Powered by Bitcoin.
        </div>
      </footer>
    </div>
  `;
}

function mountNavbar() {
  const { role, name } = authStore.state;
  const activeRole  = role  || ROLES.GUEST;
  const links       = ROLE_NAV[activeRole] || ROLE_NAV[ROLES.GUEST];
  const navbar = new Navbar({ logoText: 'PamojaBuild', links, role: activeRole, name });
  const existing = document.querySelector('.navbar');
  if (existing) existing.replaceWith(navbar.element);
  else document.body.prepend(navbar.element);
}

mountNavbar();
authStore.subscribe(() => mountNavbar());

const router = new Router([
  // Public
  { path: '/',                                handler: renderHomePage                  },
  { path: '/signin',                          handler: renderSignInPage                },
  { path: '/register',                        handler: renderRegisterPage              },

  // Donor
  { path: '/donor/donations',                 handler: renderDonorDonationsPage        },

  // Campaign creator
  { path: '/campaigns',                       handler: renderCampaignsPage             },
  { path: '/campaigns/new',                   handler: renderCreateCampaignPage        },
  { path: '/tasks/:slug',                     handler: renderTaskDetailPage            },
  { path: '/tasks/:slug/donate',              handler: renderDonationPage              },

  // Trustee
  { path: '/trustee/register',                handler: renderTrusteeDashboardPage      },
  { path: '/trustee/payout',                  handler: renderPayoutReviewPage          },

  // Volunteer
  { path: '/volunteer/dashboard',             handler: renderVolunteerDashboardPage    },
  { path: '/volunteer/tasks',                 handler: renderVolunteerTaskBrowserPage  },
  { path: '/volunteer/tasks/:slug',           handler: renderVolunteerTaskDetailsPage  },
  { path: '/volunteer/tasks/:slug/submit',    handler: renderSubmissionPage            },
  { path: '/volunteer/profile',               handler: renderVolunteerProfilePage      },
  { path: '/volunteer/applications',          handler: renderApplicationHistoryPage    },
  { path: '/volunteer/payments',              handler: renderVolunteerPaymentsPage     },
  { path: '/volunteer/reputation',            handler: renderReputationPage            },

  // Admin
  { path: '/admin',                           handler: renderAdminPage                 },
]);

router.init();

window.addEventListener('unhandledrejection', (e) => {
  const err = e.reason;
  if (err instanceof NetworkError) {
    Toast.show({ message: 'Connection lost. Check your internet.', type: 'error' });
  } else if (err instanceof ApiError && err.status === 401) {
    Toast.show({ message: 'Session expired — please sign in again.', type: 'warning' });
  }
});
