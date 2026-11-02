// Page bootstrap for group.html, kept out of the page itself because the
// Content-Security-Policy refuses inline script.

var ignoreNav = document.getElementById('nav');
var groups;
var jwt;
var admin = false;
var groupOwnerID = 0;
info('Loading...');
showLoggedOutMenu();
jwt = "";
get_login(jwt);
