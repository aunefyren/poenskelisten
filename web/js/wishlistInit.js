// Page bootstrap for wishlist.html, kept out of the page itself because the
// Content-Security-Policy refuses inline script.

var ignoreNav = document.getElementById('nav');
var groups;
var jwt;
var admin = false;
var currency = "$";
var currency_padding = true;
var currency_left = false;
info('Loading...');
showLoggedOutMenu();
jwt = "";
get_login(jwt);
