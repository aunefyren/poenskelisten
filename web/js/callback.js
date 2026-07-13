// The /oauth/callback page: exchange the authorization code for tokens (via
// handleOAuthCallback in functions.js), then return to where the user started.
function runCallback() {
    handleOAuthCallback();
}
