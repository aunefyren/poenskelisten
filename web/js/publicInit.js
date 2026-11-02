// Page bootstrap for public.html, kept out of the page itself because the
// Content-Security-Policy refuses inline script.

var ignoreNav = document.getElementById('nav');
var groups;
var jwt;
var admin = false;
var currency = "$"
var currency_pad = true;
info('Loading...');
showLoggedOutMenu();
jwt = "";
get_login(jwt);
