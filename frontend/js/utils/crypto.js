const ALGORITHM = { name: 'ECDSA', namedCurve: 'P-256' };
const SIGN_PARAMS = { name: 'ECDSA', hash: { name: 'SHA-256' } };

export async function generateTrusteeKeyPair() {
  const keyPair = await crypto.subtle.generateKey(ALGORITHM, false, ['sign', 'verify']);

  const rawPub   = await crypto.subtle.exportKey('raw', keyPair.publicKey);
  const pubKeyHex = Array.from(new Uint8Array(rawPub))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');

  return { keyPair, pubKeyHex };
}

export async function signWithPrivateKey(privateKey, data) {
  const sigBuffer  = await crypto.subtle.sign(SIGN_PARAMS, privateKey, data);
  return Array.from(new Uint8Array(sigBuffer))
    .map(b => b.toString(16).padStart(2, '0'))
    .join('');
}

export function toBytes(str) {
  return new TextEncoder().encode(str);
}

export async function importPubKeyHex(pubKeyHex) {
  const raw = new Uint8Array(pubKeyHex.match(/.{1,2}/g).map(b => parseInt(b, 16)));
  return crypto.subtle.importKey('raw', raw, ALGORITHM, true, ['verify']);
}