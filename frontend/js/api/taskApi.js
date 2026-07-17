import client from './client.js';
import { ROUTES } from '../config/routes.js';

const taskApi = {
  list() {
    return client.get(ROUTES.TASKS.LIST());
  },

  create(payload) {
    return client.post(ROUTES.TASKS.CREATE(), payload);
  },

  getBySlug(slug) {
    return client.get(ROUTES.TASKS.GET_BY_SLUG(slug));
  },
};

export default taskApi;
