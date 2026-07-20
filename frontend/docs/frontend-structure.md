```text
frontend/
├── .gitignore
├── README.md
├── index.html
├── package.json
├── vite.config.js
├── playwright.config.js
│
├── assets/
│   ├── fonts/           # Drop custom font files here
│   ├── icons/           # SVG icons / favicon
│   └── images/          # Static images / OG images
│
├── docs/
│   └── api-integration.md     # Full API table, error/retry/auth/caching strategy
│
├── styles/
│   ├── variables.css          # All design tokens (colours, spacing, type, radius…)
│   ├── reset.css              # Box-model reset, base typography
│   ├── layout.css             # Container, navbar, hero, two-column grids, mobile nav
│   ├── components/
│   │   ├── components.css     # Buttons, badges, cards, forms, Toast, Modal, Loader,
│   │   │                      #   QRCodeViewer, SignatureStatusTracker, TaskCard,
│   │   │                      #   APIErrorDisplay, KeyGenPanel, CopyableBlock
│   │   └── volunteer.css      # VolunteerCard, ReputationBadge, ApplicationStatusBadge,
│   │                          #   TaskFilter, TaskGrid, SubmissionUploader, EvidenceGallery,
│   │                          #   PaymentTable, PaymentSummary, PaymentTabs,
│   │                          #   VerificationTracker, VolunteerDashboard, AppHistory,
│   │                          #   VolunteerProfile, ReputationPage
│   └── pages/
│       ├── auth.css           # Sign In & Register card layout
│       ├── admin.css          # Admin grid, stat rows, flagged task list
│       └── shared.css         # Sidebar, TrusteeCard, DonationCard, VolunteerProfileCard,
│                              #   page-header row variant, value-sats-sm utility
│
├── templates/                 # HTML partial templates (reserved for SSR/build use)
│
├── js/
│   ├── app.js                 # Bootstrap: mounts Navbar, initialises Router, global error handler
│   ├── router.js              # History-API SPA router, link interception
│   │
│   ├── config/
│   │   ├── env.js             # window.__ENV__ reader — all runtime config
│   │   ├── routes.js          # Every backend endpoint as a typed function
│   │   └── roles.js           # ROLES enum, ROLE_LABELS, ROLE_NAV per-role link sets
│   │
│   ├── api/                   # One module per backend package — raw HTTP only
│   │   ├── client.js          # fetch wrapper: auth injection, interceptors, retry, timeout,
│   │   │                      #   ApiError / NetworkError / TimeoutError types
│   │   ├── authApi.js         # POST /auth/register, /auth/signin, /auth/signout
│   │   ├── taskApi.js         # GET/POST /tasks, GET /tasks/:slug
│   │   ├── trusteeApi.js      # POST /tasks/:slug/trustees
│   │   ├── lightningApi.js    # POST /tasks/:slug/donate
│   │   ├── ledgerApi.js       # Placeholder + shared ledger types
│   │   ├── escrowApi.js       # GET/POST /trustees/payouts/:slug
│   │   └── volunteerApi.js    # All volunteer domain endpoints (profile, apply,
│   │                          #   submissions, complete, payments, payment-profile)
│   │
│   ├── services/              # Business logic: multi-step flows, derived computations
│   │   ├── taskService.js     # canAcceptDonations, isReadyForPayout, fetchTaskAndInvoice
│   │   ├── volunteerService.js# computeReputationStats, findApplication, filterPayments
│   │   └── escrowService.js   # buildCoSignPayload, submitSignature, signaturesRemaining
│   │
│   ├── state/                 # Reactive Store instances — one per domain
│   │   ├── store.js           # Base Store class (pub/sub, immutable state snapshots)
│   │   ├── authStore.js       # token, userId, role — wires tokenAccessor into client
│   │   ├── taskStore.js       # currentTask, loading, error
│   │   ├── trusteeStore.js    # registeredKeys, loading, error
│   │   ├── donationStore.js   # invoice, loading, error
│   │   ├── payoutStore.js     # manifest, thresholdReached, submittedSigs
│   │   └── volunteerStore.js  # profileStore, applicationStore, submissionStore,
│   │                          #   paymentStore, taskBrowserStore + all actions
│   │
│   ├── components/            # Reusable UI — accept config, never hardcode data
│   │   ├── Navbar.js                  # Role-aware top nav, auth controls, hamburger
│   │   ├── Sidebar.js                 # Role-aware collapsible side nav
│   │   ├── Toast.js                   # Non-blocking notifications (success/error/warning/info)
│   │   ├── Modal.js                   # Accessible focus-trapped dialog
│   │   ├── Loader.js                  # Full-screen overlay + inline spinner
│   │   ├── APIErrorDisplay.js         # Structured error block with retry button
│   │   ├── TaskCard.js                # Campaign summary card (status + financial badges)
│   │   ├── TrusteeCard.js             # Slot assignment + key info + signed state
│   │   ├── DonationCard.js            # Invoice card with embedded QR + countdown
│   │   ├── QRCodeViewer.js            # BOLT11 QR code, live expiry countdown, copy
│   │   ├── SignatureStatusTracker.js  # 5-slot 3/5 multi-sig progress bar
│   │   ├── VerificationTracker.js     # 13-step volunteer lifecycle stepper
│   │   ├── VolunteerCard.js           # Compact volunteer list/grid card
│   │   ├── VolunteerProfileCard.js    # Full profile: avatar, bio, stats, payment info
│   │   ├── ReputationBadge.js         # Score + tier chip (New/Active/Trusted/Expert/Elite)
│   │   ├── ApplicationStatusBadge.js  # pending / approved / rejected chip
│   │   ├── TaskFilter.js              # Category + region + status filter bar
│   │   ├── SubmissionUploader.js      # Description + URL list evidence form
│   │   ├── EvidenceGallery.js         # Submission list with inline image previews
│   │   ├── PaymentHistoryTable.js     # Paginated table with per-row step tracker
│   │   └── PaymentSummaryCard.js      # Aggregate: total earned, completed, pending
│   │
│   ├── pages/                 # One render function per route
│   │   ├── signInPage.js              # POST /auth/signin → setSession → role redirect
│   │   ├── registerPage.js            # POST /auth/register → setSession → role redirect
│   │   ├── taskDetailPage.js          # GET /tasks/:slug — creator/public task view
│   │   ├── createCampaignPage.js      # POST /tasks — campaign creator form
│   │   ├── campaignsPage.js           # GET /tasks — creator's own campaigns list
│   │   ├── donationPage.js            # POST /tasks/:slug/donate — QR invoice flow
│   │   ├── donorDonationsPage.js      # Donor browse + filter → donate redirect
│   │   ├── trusteeDashboardPage.js    # POST /tasks/:slug/trustees — key registration
│   │   ├── payoutReviewPage.js        # GET+POST /trustees/payouts/:slug — sign flow
│   │   ├── adminPage.js               # System overview: task counts, flagged items
│   │   ├── volunteerDashboardPage.js  # Profile + app summary + payment summary
│   │   ├── volunteerTaskBrowserPage.js# GET /tasks — filter + card grid
│   │   ├── volunteerTaskDetailsPage.js# Task detail + apply modal
│   │   ├── volunteerProfilePage.js    # GET/PUT /volunteers/profile + payment profile
│   │   ├── applicationHistoryPage.js  # GET /volunteers/applications
│   │   ├── submissionPage.js          # POST /tasks/:slug/submissions + complete button
│   │   ├── volunteerPaymentsPage.js   # GET /volunteers/payments — tabs + table
│   │   └── reputationPage.js          # Derived metrics from profile + payments + apps
│   │
│   └── utils/
│       ├── utils.js           # formatSats, satsToBtc, formatDate, truncateHex,
│       │                      #   isValidEmail, isValidXpub, isHex, $one, $all,
│       │                      #   navigate, getPathParam
│       └── crypto.js          # WebCrypto: generateTrusteeKeyPair, signWithPrivateKey,
│                              #   importPubKeyHex, toBytes
│
└── tests/
    ├── unit/
    │   └── utils.test.js      # formatSats, satsToBtc, truncateHex, isHex, Store
    ├── integration/
    │   └── api.test.js        # fetch-mocked tests: client, taskApi, volunteerApi, escrowApi
    └── e2e/
        └── journeys.test.js   # Playwright: donor, volunteer, creator, trustee, nav journeys
```