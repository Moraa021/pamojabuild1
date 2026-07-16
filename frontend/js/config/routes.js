import { ENV } from './env.js';

const BASE = `${ENV.API_BASE_URL}/api/${ENV.API_VERSION}`;

export const ROUTES = Object.freeze({
  TASKS: {
    LIST:         ()     => `${BASE}/tasks`,
    CREATE:       ()     => `${BASE}/tasks`,
    GET_BY_SLUG:  (slug) => `${BASE}/tasks/${slug}`,
  },

  TRUSTEES: {
    REGISTER_KEYS: (slug) => `${BASE}/tasks/${slug}/trustees`,
  },

  LIGHTNING: {
    REQUEST_INVOICE: (slug) => `${BASE}/tasks/${slug}/donate`,
  },

  ESCROW: {
    GET_PAYOUT_MANIFEST: (slug) => `${BASE}/trustees/payouts/${slug}`,
    SUBMIT_COSIGNATURES: (slug) => `${BASE}/trustees/payouts/${slug}/sign`,
  },

  VOLUNTEERS: {
    GET_PROFILE:          () => `${BASE}/volunteers/profile`,
    UPDATE_PROFILE:       () => `${BASE}/volunteers/profile`,
    APPLY:                (slug) => `${BASE}/tasks/${slug}/apply`,
    GET_APPLICATIONS:     () => `${BASE}/volunteers/applications`,
    CREATE_SUBMISSION:    (slug) => `${BASE}/tasks/${slug}/submissions`,
    GET_SUBMISSIONS:      (slug) => `${BASE}/tasks/${slug}/submissions`,
    COMPLETE_TASK:        (slug) => `${BASE}/tasks/${slug}/complete`,
    GET_PAYMENTS:         () => `${BASE}/volunteers/payments`,
    SAVE_PAYMENT_PROFILE: () => `${BASE}/volunteers/payment-profile`,
    GET_PAYMENT_PROFILE:  () => `${BASE}/volunteers/payment-profile`,
  },
});
