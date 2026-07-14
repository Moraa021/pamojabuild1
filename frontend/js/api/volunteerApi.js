import client from './client.js';
import { ENV } from '../config/env.js';

const BASE = `${ENV.API_BASE_URL}/api/${ENV.API_VERSION}`;

const volunteerApi = {
  listTasks() {
    return client.get(`${BASE}/tasks`);
  },

  applyToTask(slug, payload) {
    return client.post(`${BASE}/tasks/${slug}/apply`, payload);
  },

  getApplications() {
    return client.get(`${BASE}/volunteers/applications`);
  },

  getProfile() {
    return client.get(`${BASE}/volunteers/profile`);
  },

  updateProfile(payload) {
    return client.put(`${BASE}/volunteers/profile`, payload);
  },

  createSubmission(slug, payload) {
    return client.post(`${BASE}/tasks/${slug}/submissions`, payload);
  },

  getSubmissions(slug) {
    return client.get(`${BASE}/tasks/${slug}/submissions`);
  },

  completeTask(slug) {
    return client.post(`${BASE}/tasks/${slug}/complete`, {});
  },

  getPayments() {
    return client.get(`${BASE}/volunteers/payments`);
  },

  savePaymentProfile(payload) {
    return client.post(`${BASE}/volunteers/payment-profile`, payload);
  },

  getPaymentProfile() {
    return client.get(`${BASE}/volunteers/payment-profile`);
  },
};

export default volunteerApi;
