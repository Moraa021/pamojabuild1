import client from './client.js';
import { ROUTES } from '../config/routes.js';

const escrowApi = {

  getPayoutManifest(slug) {
    return client.get(ROUTES.ESCROW.GET_PAYOUT_MANIFEST(slug));
  },

  submitCoSignatures(slug, payload) {
    return client.post(ROUTES.ESCROW.SUBMIT_COSIGNATURES(slug), payload);
  },
};

export default escrowApi;
