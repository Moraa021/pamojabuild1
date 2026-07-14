export const ROLES = Object.freeze({
  GUEST:     'guest',
  DONOR:     'donor',
  VOLUNTEER: 'volunteer',
  CREATOR:   'creator',
  TRUSTEE:   'trustee',
  ADMIN:     'admin',
});

export function hasAccess(userRole, allowed) {
  return allowed.includes(userRole) || userRole === ROLES.ADMIN;
}

export const ROLE_LABELS = {
  [ROLES.GUEST]:     'Guest',
  [ROLES.DONOR]:     'Donor',
  [ROLES.VOLUNTEER]: 'Volunteer',
  [ROLES.CREATOR]:   'Campaign Creator',
  [ROLES.TRUSTEE]:   'Trustee',
  [ROLES.ADMIN]:     'Administrator',
};

export const ROLE_NAV = {
  [ROLES.GUEST]: [
    { label: 'Browse campaigns', href: '/volunteer/tasks' },
    { label: 'Sign in',          href: '/signin' },
  ],
  [ROLES.DONOR]: [
    { label: 'Browse campaigns', href: '/volunteer/tasks' },
    { label: 'My donations',     href: '/donor/donations' },
  ],
  [ROLES.VOLUNTEER]: [
    { label: 'Browse tasks',     href: '/volunteer/tasks' },
    { label: 'My dashboard',     href: '/volunteer/dashboard' },
    { label: 'Applications',     href: '/volunteer/applications' },
    { label: 'Payments',         href: '/volunteer/payments' },
  ],
  [ROLES.CREATOR]: [
    { label: 'My campaigns',     href: '/campaigns' },
    { label: 'Create campaign',  href: '/campaigns/new' },
  ],
  [ROLES.TRUSTEE]: [
    { label: 'Payout review',    href: '/trustee/payout' },
    { label: 'Register keys',    href: '/trustee/register' },
  ],
  [ROLES.ADMIN]: [
    { label: 'Admin overview',   href: '/admin' },
    { label: 'All tasks',        href: '/volunteer/tasks' },
    { label: 'Trustee panel',    href: '/trustee/payout' },
  ],
};
