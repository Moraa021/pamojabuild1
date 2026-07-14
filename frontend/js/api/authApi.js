import client from './client.js';
import { ENV } from '../config/env.js';

const BASE = `${ENV.API_BASE_URL}/api/${ENV.API_VERSION}`;

const authApi = {
  register(payload) {
    return client.post(`${BASE}/auth/register`, payload);
  },

  signIn(payload) {
    return client.post(`${BASE}/auth/signin`, payload);
  },

  signOut() {
    return client.post(`${BASE}/auth/signout`, {});
  },
};

export default authApi;
