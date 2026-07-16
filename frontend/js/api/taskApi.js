import client from './client.js';
import { ROUTES } from '../config/routes.js';

const taskApi = {
  create(payload) {
    return client.post(ROUTES.TASKS.CREATE(), payload);
  },

  getBySlug(slug) {
    return client.get(ROUTES.TASKS.GET_BY_SLUG(slug));
  },
};

export default taskApi;
