// Page bootstrap for verify.html, kept out of the page itself because the
// Content-Security-Policy refuses inline script.

var ignoreNav = document.getElementById('nav');
var jwt = "";
var admin = false;
showLoggedOutMenu();
// Reached only when the login (SSO) session needs email verification;
// the form submits with the SSO cookie.
load_verify_account();
