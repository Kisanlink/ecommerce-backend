// Simple script to generate Firebase ID token for testing
// Run with: node test_token.js

const admin = require('firebase-admin');

// Initialize Firebase Admin (using the same configuration as your Go app)
const serviceAccount = require('./internal/config/firebase_service_account.json');

admin.initializeApp({
  credential: admin.credential.cert(serviceAccount)
});

// Create a custom token for testing
const uid = 'test-user-id';
admin.auth().createCustomToken(uid)
  .then((customToken) => {
    console.log('Custom token:', customToken);
    console.log('Use this token to exchange for ID token via Firebase Auth REST API');
  })
  .catch((error) => {
    console.log('Error creating custom token:', error);
  });
