// Page bootstrap for login.html, kept out of the page itself because the
// Content-Security-Policy refuses inline script.

var ignoreNav = document.getElementById('nav');
var jwt;
var admin = false;
info('Loading...');
showLoggedOutMenu();
jwt = "";
get_login(jwt);
