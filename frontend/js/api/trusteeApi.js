import client from './client.js';
import { ROUTES } from '../config/routes.js';

const trusteeApi = {
  registerKeys(slug, payload) {
    return client.post(ROUTES.TRUSTEES.REGISTER_KEYS(slug), payload);
  },
};

export default trusteeApi;
