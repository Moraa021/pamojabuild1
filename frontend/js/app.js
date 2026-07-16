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
  container.innerHTML = `
    <section class="home container">
      <div class="home__hero">
        <h1 class="home__headline">Fund volunteers with Bitcoin</h1>
        <p class="home__sub">
          Create a campaign, raise sats via Lightning, and release funds through a
          transparent 3-of-5 multi-sig payout reviewed by independent trustees.
        </p>
        <div class="home__actions">
          <a href="/volunteer/tasks" class="btn btn--primary btn--lg">Browse tasks</a>
          <a href="/campaigns/new"   class="btn btn--ghost btn--lg">Create a campaign</a>
        </div>
      </div>
    </section>
  `;
}

function mountNavbar() {
  const role  = authStore.state.role || ROLES.GUEST;
  const links = ROLE_NAV[role]        || ROLE_NAV[ROLES.GUEST];
  const navbar = new Navbar({ logoText: 'VolunteerTasks', links });
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
