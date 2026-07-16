import client from './client.js';
import { ROUTES } from '../config/routes.js';

const lightningApi = {
  requestInvoice(slug, payload) {
    return client.post(ROUTES.LIGHTNING.REQUEST_INVOICE(slug), payload);
  },
};

export default lightningApi;
