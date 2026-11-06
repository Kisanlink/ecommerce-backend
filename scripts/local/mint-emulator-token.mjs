#!/usr/bin/env node
/*
 * LOCAL-ONLY Firebase Auth emulator token minter.
 *
 * Creates (or signs in) a user in the Firebase Auth EMULATOR and prints its UID and a fresh
 * ID token. Pass that token as `Authorization: Bearer <token>` to the backend: because the
 * backend runs with FIREBASE_AUTH_EMULATOR_HOST set, its Admin SDK verifies the token
 * against the emulator (no Google contact, no real credentials), and on the first
 * authenticated payment call it auto-creates the matching local user from the emulator user
 * (mapFirebaseUIDToInternalUserID -> firebaseClient.GetUser -> CreateGoogleUser). So you do
 * NOT need to hand-insert a users row.
 *
 * Requires the Auth emulator running:  firebase emulators:start --only auth
 *
 * Usage:
 *   FIREBASE_PROJECT_ID=<same project id the emulator + backend use> \
 *     node scripts/local/mint-emulator-token.mjs [email] [password]
 *
 *   FIREBASE_AUTH_EMULATOR_HOST   default 127.0.0.1:9099
 *   email                         default tester@local.test
 *   password                      default Test123!
 */
// Use 127.0.0.1, not "localhost": Node 18+ resolves "localhost" to IPv6 ::1 first, but the
// emulator binds IPv4 only, so "localhost" gives ECONNREFUSED ::1:9099. Normalize either way.
const host = (process.env.FIREBASE_AUTH_EMULATOR_HOST || '127.0.0.1:9099').replace('localhost', '127.0.0.1');
const project = process.env.FIREBASE_PROJECT_ID;
const email = process.argv[2] || 'tester@local.test';
const password = process.argv[3] || 'Test123!';

if (!project) {
  console.error('Set FIREBASE_PROJECT_ID to the SAME project id the emulator runs as and the backend verifies against.');
  console.error('(The token aud/iss must match the backend\'s Firebase project id, or VerifyIDToken rejects it.)');
  process.exit(1);
}

// The emulator accepts any API key value.
const base = `http://${host}/identitytoolkit.googleapis.com/v1/accounts`;
const key = 'fake-local-api-key';

async function call(path, payload) {
  const r = await fetch(`${base}:${path}?key=${key}`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  });
  return r.json();
}

let out = await call('signUp', { email, password, returnSecureToken: true });
if (out.error) {
  // Already exists (or other) -> try sign-in.
  out = await call('signInWithPassword', { email, password, returnSecureToken: true });
}
if (out.error) {
  console.error('Emulator error:', JSON.stringify(out.error));
  console.error('Is the Auth emulator running on ' + host + ' ?  (firebase emulators:start --only auth)');
  process.exit(1);
}

console.log('Project :', project);
console.log('Email   :', email);
console.log('UID     :', out.localId);
console.log('');
console.log('ID TOKEN:');
console.log(out.idToken);
console.log('');
console.log('Use it as:  Authorization: Bearer <ID TOKEN>');
console.log('The first authenticated payment call auto-creates the local user for this UID.');
