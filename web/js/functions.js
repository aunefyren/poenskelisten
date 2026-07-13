var api_url = window.location.origin + "/api/";

// Load service worker
if('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/service-worker.js')
.then((reg) => {
    // registration worked
    console.log('Registration succeeded. Scope is ' + reg.scope);
})};

// Make XHTTP requests
function makeRequest (method, url, data) {
    return new Promise(function (resolve, reject) {
    var xhr = new XMLHttpRequest();
    xhr.open(method, url);
    xhr.onload = function () {
      if (this.status >= 200 && this.status < 300) {
        resolve(xhr.response);
      } else {
        reject({
          status: this.status,
          statusText: xhr.statusText
        });
      }
    };
    xhr.onerror = function () {
      reject({
        status: this.status,
        statusText: xhr.statusText
      });
    };
    if(method=="POST" && data){
        xhr.send(data);
    }else{
        xhr.send();
    }
    });
}

// Set new browser cookie
function set_cookie(cname, cvalue, exdays) {
    cvalue = encodeURI(cvalue)
    var d = new Date();
    d.setTime(d.getTime() + (exdays*24*60*60*1000));
    var expires = "expires="+ d.toUTCString();
    document.cookie = cname + '=' + cvalue + "; " + expires + "; path=/; samesite=strict;";
}

// Get cookie from browser
function get_cookie(cname) {
    var name = cname + '=';
    var decodedCookie = decodeURIComponent(document.cookie);
    var ca = decodedCookie.split(';');
    for(var i = 0; i <ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' '){
            c = c.substring(1);
        }

        if (c.indexOf(name) == 0) {
            return c.substring(name.length, c.length);
        }
    }
    return "";
}

// OAuth 2.1 first-party client config. The web app is a public PKCE client of
// Pønskelisten's own authorization server.
var OAUTH_CLIENT_ID = "poenskelisten-web";
var oauth_base = window.location.origin + "/oauth/";
var sessionRefreshTimer = null;

// base64url without padding (for PKCE + state).
function base64UrlEncode(bytes) {
    var str = btoa(String.fromCharCode.apply(null, bytes));
    return str.replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
}

function randomString(length) {
    var arr = new Uint8Array(length);
    crypto.getRandomValues(arr);
    return base64UrlEncode(arr).slice(0, length);
}

// Generate a PKCE verifier + S256 challenge.
function generatePKCE() {
    var verifier = randomString(64);
    return crypto.subtle.digest("SHA-256", new TextEncoder().encode(verifier)).then(function(digest) {
        return { verifier: verifier, challenge: base64UrlEncode(new Uint8Array(digest)) };
    });
}

// Pages where an unauthenticated visit should NOT auto-start the OAuth flow
// (they render on their own, or are part of the flow itself).
function isPublicAuthPage() {
    var p = window.location.pathname;
    return p === "/login" || p === "/register" || p === "/verify" || p === "/enroll" ||
        p === "/oauth/callback" || p.indexOf("/wishlists/public") === 0;
}

// Begin the authorization-code + PKCE flow: stash verifier/state and where to
// return, then redirect to /oauth/authorize.
function startAuthorizeFlow() {
    generatePKCE().then(function(pkce) {
        var state = randomString(32);
        sessionStorage.setItem("pkce_verifier", pkce.verifier);
        sessionStorage.setItem("oauth_state", state);
        sessionStorage.setItem("post_login_redirect", window.location.pathname + window.location.search);

        var params = new URLSearchParams({
            client_id: OAUTH_CLIENT_ID,
            response_type: "code",
            redirect_uri: window.location.origin + "/oauth/callback",
            scope: "openid profile email",
            state: state,
            code_challenge: pkce.challenge,
            code_challenge_method: "S256"
        });
        window.location.href = oauth_base + "authorize?" + params.toString();
    }).catch(function(e) {
        console.log("Failed to start authorization flow: " + e);
        if(window.location.pathname !== "/login") {
            window.location.href = "/login";
        }
    });
}

// Handle the /oauth/callback: exchange the code for tokens, then return the user
// to where they started.
function handleOAuthCallback() {
    var params = new URLSearchParams(window.location.search);
    var code = params.get("code");
    var state = params.get("state");
    var oauthError = params.get("error");

    if(oauthError) {
        window.location.href = "/login?error=" + encodeURIComponent(oauthError);
        return;
    }
    if(!code || !state || state !== sessionStorage.getItem("oauth_state")) {
        window.location.href = "/login";
        return;
    }

    var body = new URLSearchParams({
        grant_type: "authorization_code",
        code: code,
        redirect_uri: window.location.origin + "/oauth/callback",
        client_id: OAUTH_CLIENT_ID,
        code_verifier: sessionStorage.getItem("pkce_verifier") || ""
    });

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            if(this.status >= 200 && this.status < 300) {
                try { result = JSON.parse(this.responseText); } catch(e) { result = null; }
            }
            if(result && result.access_token) {
                set_cookie("poenskelisten", result.access_token, 7);
                var dest = sessionStorage.getItem("post_login_redirect") || "/";
                sessionStorage.removeItem("pkce_verifier");
                sessionStorage.removeItem("oauth_state");
                sessionStorage.removeItem("post_login_redirect");
                if(dest === "/oauth/callback" || dest.indexOf("/login") === 0) {
                    dest = "/";
                }
                window.location.href = dest;
                return;
            }
            window.location.href = "/login";
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", oauth_base + "token");
    xhttp.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    xhttp.send(body.toString());
}

// Roll the access token using the refresh cookie via the OAuth token endpoint.
function refreshAccessToken(onSuccess, onFailure) {
    var body = new URLSearchParams({ grant_type: "refresh_token", client_id: OAUTH_CLIENT_ID });

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            if(this.status >= 200 && this.status < 300) {
                var result;
                try { result = JSON.parse(this.responseText); } catch(e) { result = null; }
                if(result && result.access_token) {
                    set_cookie("poenskelisten", result.access_token, 7);
                    jwt = result.access_token;
                    if(onSuccess) onSuccess();
                    return;
                }
            }
            if(onFailure) onFailure();
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", oauth_base + "token");
    xhttp.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    xhttp.send(body.toString());
}

// Access tokens are short-lived (~15 min), so roll them on a timer while a tab is
// open to keep an active session alive between reloads.
function startSessionRefreshTimer() {
    if(isPublicAuthPage()) {
        return;
    }
    if(sessionRefreshTimer !== null) {
        return;
    }
    sessionRefreshTimer = setInterval(function() {
        refreshAccessToken(null, null);
    }, 10 * 60 * 1000);
}

// Bootstrap the page: ensure a valid access token (refresh, or start the OAuth
// flow), then load the current user. triedRefresh guards against a refresh loop.
function get_login(cookie, triedRefresh) {
    triedRefresh = triedRefresh === true;
    jwt = cookie ? cookie : "";

    var recover = function() {
        if(isPublicAuthPage()) {
            load_page(false);
        } else if(!triedRefresh) {
            refreshAccessToken(function() { get_login(jwt, true); }, function() { startAuthorizeFlow(); });
        } else {
            startAuthorizeFlow();
        }
    };

    if(jwt === "") {
        recover();
        return;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            if(this.status >= 200 && this.status < 300) {
                load_page(this.responseText);
                startSessionRefreshTimer();
                return;
            }
            // Token missing/expired/invalid — try to recover.
            set_cookie("poenskelisten", "", 7);
            jwt = "";
            recover();
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/me");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

// Called when login session was rejected, showing no content and an error
function invalid_session() {
    showLoggedOutMenu();
    document.getElementById('content').innerHTML = '';
    document.getElementById('card-header').innerHTML = 'Log in...';
    error('No access.');
}

// Call given URL, get image from API, call place_image() which is a local function, not here
function get_image(url, cookie, info, iteration) {

    if(iteration > 5) {
        return;
    } if (iteration =! null) {
        iteration++;
    }

    var json_jwt = JSON.stringify({});
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            var result;
            if(result = JSON.parse(this.responseText)) { 
                place_image(result.image, info);
            } else {
                place_image(false);
            }
        } else if(this.readyState == 4 && this.status == 404) {
            console.log('Image returned 404. Info: ' . info);

            setTimeout(() => {
                get_image(url, cookie, info, iteration);
            }, 5000);

        } else if(this.readyState == 4 && this.status !== 200 && this.status !== 404) {
            place_image(false);
        }
    };
    xhttp.withCredentials = false;
    xhttp.open("post", url);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", "Bearer " + cookie);
    xhttp.send(json_jwt);
    return;
}

/// Recieves file and returns Base64 string of file
function get_base64(file, onLoadCallback) {
    return new Promise(function(resolve, reject) {
        var reader = new FileReader();
        reader.onload = function() { resolve(reader.result); };
        reader.onerror = reject;
        reader.readAsDataURL(file);
    });
  }

// Show options that can be access when logged in
function showLoggedInMenu() {
    document.getElementById('login').classList.add('disabled');
    document.getElementById('login').classList.remove('enabled');

    document.getElementById('logout').classList.add('enabled');
    document.getElementById('logout').classList.remove('disabled');

    document.getElementById('groups').classList.add('enabled');
    document.getElementById('groups').classList.remove('disabled');

    document.getElementById('wishlists').classList.add('enabled');
    document.getElementById('wishlists').classList.remove('disabled');

    document.getElementById('account').classList.add('enabled');
    document.getElementById('account').classList.remove('disabled');

    document.getElementById('register').classList.add('disabled');
    document.getElementById('register').classList.remove('enabled');
}

// Remove options not accessable when not logged in
function showLoggedOutMenu() {
    document.getElementById('login').classList.add('enabled');
    document.getElementById('login').classList.remove('disabled');

    document.getElementById('logout').classList.add('disabled');
    document.getElementById('logout').classList.remove('enabled');

    document.getElementById('groups').classList.add('disabled');
    document.getElementById('groups').classList.remove('enabled');

    document.getElementById('wishlists').classList.add('disabled');
    document.getElementById('wishlists').classList.remove('enabled');

    document.getElementById('account').classList.add('disabled');
    document.getElementById('account').classList.remove('enabled');

    document.getElementById('register').classList.add('enabled');
    document.getElementById('register').classList.remove('disabled');
}

function showAdminMenu(admin) {
    if(admin) {
        document.getElementById('admin').classList.add('enabled');
        document.getElementById('admin').classList.remove('disabled');
    } else {
        document.getElementById('admin').classList.add('disabled');
        document.getElementById('admin').classList.remove('enabled');
    }
}

// Toggle navar expansion
function toggle_navbar() {
    var x = document.getElementById("navbar");
    var y = document.getElementById("nav-logo");
    if (!x.classList.contains("responsive")) {
        x.classList.add("responsive");
        x.classList.add("responsive");
        x.classList.remove("unresponsive");
        x.classList.remove("unresponsive");
        freezerScrolling(true);
    } else {
        x.classList.add("unresponsive");
        x.classList.add("unresponsive");
        x.classList.remove("responsive");
        x.classList.remove("responsive");
        freezerScrolling(false);
    }
}

// Toggle navbar if clicked outside
document.addEventListener('click', function(event) {
    var myModal = document.getElementById("myModal")
    if(myModal && myModal.classList.contains("open") && (event.target.id == "myModal" || event.target.id == "caption")) {
        toggleModal();
        return;
    }

    // Some pages (OAuth callback, enrollment gate) have no navbar; guard against it.
    var navbar = document.getElementById('navbar');
    if (!navbar) {
        return;
    }
    var isClickInsideElement = ignoreNav && ignoreNav.contains(event.target);
    if (!isClickInsideElement) {
        if (navbar.classList.contains('responsive')) {
            toggle_navbar();
        }
        return;
    }
});

// Function for checking file extension of file
function return_file_extension(filename) {
    return (/[.]/.exec(filename)) ? /[^.]+$/.exec(filename) : undefined;
}

// Removes notification from response bar
function clearResponse(){
    document.getElementById("response").innerHTML = '';
}

// Displays a blue notification
function info(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-info'>" + message + "</div>";
    window.scrollTo(0, 0);
}

// Displays a green notification
function success(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-success'>" + message + "</div>";
    window.scrollTo(0, 0);
    toggleModal(false);
}

// Displays a red notification
function error(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-danger'>" + message + "</div>";
    window.scrollTo(0, 0);
    toggleModal(false);
}

// When log out button is pressed, revoke the refresh session + SSO cookie (best
// effort), clear the local access token, and return to the login page.
function logout() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            set_cookie("poenskelisten", "", 1);
            window.location.href = '/login';
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", oauth_base + "revoke");
    xhttp.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    xhttp.send();
}

// Return GET parameters in a given URL
function get_url_parameters(url) {
    
    const parameters = {}
    try {
        let paramString = url.split('?')[1];
        let params_arr = paramString.split('&');
        for(let i = 0; i < params_arr.length; i++) {
            let pair = params_arr[i].split('=');
            parameters[pair[0]] = pair[1];
        }
    }
    catch {
        return false
    }

    return parameters;
}

// convert a Unicode string to a string in which
// each 16-bit unit occupies only one byte
function toBinary(string) {
    const codeUnits = Uint16Array.from(
        { length: string.length },
        (element, index) => string.charCodeAt(index)
    );
    const charCodes = new Uint8Array(codeUnits.buffer);

    let result = "";
    charCodes.forEach((char) => {
        result += String.fromCharCode(char);
    });
    return result;
}

function fromBinary(binary) {
    const bytes = Uint8Array.from({ length: binary.length }, (element, index) =>
        binary.charCodeAt(index)
    );
    const charCodes = new Uint16Array(bytes.buffer);

    let result = "";
    charCodes.forEach((char) => {
        result += String.fromCharCode(char);
    });
    return result;
}

function toBASE64(string) {
    var binaryString = toBinary(string);
    var base64String = btoa(binaryString);
    return base64String;
}

function fromBASE64(base64String) {
    var binaryString = atob(base64String);
    var string = fromBinary(binaryString);
    return string;
}

function GetDateString(dateTime, giveWeekday) {
    try {

        var weekDayArray = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]
        var monthArray = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"]
        var weekDay = "";
        var month = "";
        var day = "";
        var year = "";

        var weekDayInt = dateTime.getDay();
        var monthInt = dateTime.getMonth();
        var dayInt = dateTime.getDate();
        var yearInt = dateTime.getYear();

        weekDay = weekDayArray[weekDayInt]
        month = monthArray[monthInt]
        day = padNumber(dayInt, 2)

        if(yearInt >= 100) {
            year = yearInt + 1900
        } else {
            year = 1900 + yearInt
        }

        if(giveWeekday) {
            return weekDay + ", " + day + ". " + month + ", " + year;
        } else {
            return day + ". " + month + ", " + year;
        }

    } catch(e) {
        console.log("Failed to generate string for date time. Error: " + e)
        return "Error"
    }
}

function padNumber(num, size) {
    var s = "000000000" + num;
    return s.substr(s.length-size);
}

function toggleModal(modalHTML) {
    var x = document.getElementById("myModal");
    if(x) {
        if (x.classList.contains("closed") && modalHTML) {
            x.classList.add("open");
            x.classList.remove("closed");
            x.style.display = "block";
            freezerScrolling(true);
        } else if(!modalHTML){
            x.classList.add("closed");
            x.classList.remove("open");
            x.style.display = "none";
            freezerScrolling(false);
        }
        
        if(modalHTML) {
            document.getElementById("modalContent").innerHTML = modalHTML
        }
    } else {
        freezerScrolling(false);
    }
}

function freezerScrolling(freeze) {
    if(freeze) {
        document.getElementsByTagName("BODY")[0].style.overflow = 'hidden';
    } else {
        document.getElementsByTagName("BODY")[0].style.overflow = 'scroll';
    }
}

function enumerateUserDisplayNames(users, nameField = "displayName") {
    const nameCounts = new Map();

    function getUniqueName(name) {
        // Get the current count for this name
        const count = nameCounts.get(name) || 0;

        if (count === 0) {
            // Name is unique, mark it used
            nameCounts.set(name, 1);
            return name;
        }

        // Increment count and generate a new name with a suffix
        const newName = `${name} (${count + 1})`;

        // Update the count for the original name
        nameCounts.set(name, count + 1);

        // Check if the new name itself conflicts
        return getUniqueName(newName);
    }

    // Iterate over the array and update names
    users.forEach(user => {
        if (user.first_name && user.last_name) {
            displayName = user.first_name + " " + user.last_name
            user.last_name = getUniqueName(user[nameField]);
        }
    });

    return users;
}

function getGroupMemberProfileImage(userID, divID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                if(!result.default) {
                    placeGroupMemberProfileImage(result.image, divID)
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

function placeGroupMemberProfileImage(imageBase64, divID) {
    var image = document.getElementById(divID)
    image.style.backgroundSize = "cover"
    image.innerHTML = ""
    image.style.backgroundImage = `url('${imageBase64}')`
    image.style.backgroundPosition = "center center"
}

function addUniqueDisplayNames(users, nameField = "displayName") {
    const nameCounts = new Map();

    function getUniqueName(name) {
        const count = nameCounts.get(name) || 0;

        if (count === 0) {
            nameCounts.set(name, 1);
            return name;
        }

        const newName = `${name} (${count + 1})`;
        nameCounts.set(name, count + 1);
        return getUniqueName(newName);
    }

    function processUser(user) {
        if (user.first_name && user.last_name) {
            const baseName = `${user.first_name} ${user.last_name}`;
            user[nameField] = getUniqueName(baseName);
        } else if (user.first_name) {
            user[nameField] = getUniqueName(user.first_name);
        } else if (user.last_name) {
            user[nameField] = getUniqueName(user.last_name);
        } else {
            user[nameField] = getUniqueName("Anonymous");
        }
    }

    if (Array.isArray(users)) {
        users.forEach(user => processUser(user));
    } else if (typeof users === "object" && users !== null) {
        Object.values(users).forEach(user => processUser(user));
    }

    return users;
}

function addUniqueDisplayNamesForObjects(objects, nameField = "displayName") {
    const nameCounts = new Map();

    function getUniqueName(name) {
        const count = nameCounts.get(name) || 0;

        if (count === 0) {
            nameCounts.set(name, 1);
            return name;
        }

        const newName = `${name} (${count + 1})`;
        nameCounts.set(name, count + 1);
        return getUniqueName(newName);
    }

    function processUser(object) {
        if (object.name) {
            const baseName = object.name;
            user[nameField] = getUniqueName(baseName);
        } else {
            user[nameField] = getUniqueName("Anonymous");
        }
    }

    if (Array.isArray(objects)) {
        objects.forEach(user => processUser(objects));
    } else if (typeof objects === "object" && users !== null) {
        Object.values(objects).forEach(object => processUser(object));
    }

    return objects;
}

function addUserToSelection() {
    var newMemberName = document.getElementById("newMemberMail").value
    var newMemberID = ""
    if(!newMemberName || newMemberName == "") {
        return;
    }

    var membersDatalistDiv = document.getElementById("userIDList")
    for (let index = 0; index < membersDatalistDiv.children.length; index++) {
        if(membersDatalistDiv.children[index].innerHTML == newMemberName) {
            newMemberID = membersDatalistDiv.children[index].value
            membersDatalistDiv.removeChild(membersDatalistDiv.children[index])
        }
    }

    if(!newMemberID || newMemberID == "") {
        alert("Invalid user")
        return;
    }

    var membersDatalistDiv = document.getElementById("userList")
    for (let index = 0; index < membersDatalistDiv.children.length; index++) {
        if(membersDatalistDiv.children[index].value == newMemberName) {
            membersDatalistDiv.removeChild(membersDatalistDiv.children[index])
        }
    }

    var membersDiv = document.getElementById("newMembers")
    var membersDivChildren = membersDiv.children

    for (let index = 0; index < membersDivChildren.length; index++) {
        var child = membersDivChildren[index]
        var childString = child.innerText
        if(childString.includes(newMemberName)) {
            return;
        }
    }

    var html = `
        <div class="group-member hoverable-opacity" title="User" id="newMember-${newMemberID}">
            <input type="hidden" id="newMember-value" value="${newMemberID}">
            <input type="hidden" id="newMember-name" value="${newMemberName}">

            <div class="group-title">
                <div class="profile-icon icon-border icon-background" id="group_member_image_wrapper_${newMemberID}">
                    <img class="icon-img " src="/assets/user.svg" id="group_member_image_${newMemberID}">
                </div>
                
                <div class="group-title-text">
                    ${newMemberName}
                </div>
            </div>

            <div class="profile-icon clickable" onclick="removeUserFromSelection('${newMemberID}')" title="Remove user">
                <img class="icon-img " src="/assets/x.svg">
            </div>
        </div>
    `;

    membersDiv.innerHTML += html
    document.getElementById("newMemberMail").value = ""

    getGroupMemberProfileImage(newMemberID, `group_member_image_wrapper_${newMemberID}`)
}

function removeUserFromSelection(userID) {
    var membersDiv = document.getElementById("newMembers")
    var membersDivChildren = membersDiv.children

    for (let index = 0; index < membersDivChildren.length; index++) {
        var child = membersDivChildren[index]
        var childProperties = child.children

        var displayName = childProperties[1].value

        if(childProperties[0].value.includes(userID)) {
            child.remove();

            var membersDatalistDiv = document.getElementById("userList")
            var optionHTML = `<option value="${displayName}">${displayName}</option>`
            membersDatalistDiv.innerHTML += optionHTML

            var membersDatalistDiv = document.getElementById("userIDList")
            var optionHTML = `<option value="${userID}">${displayName}</option>`
            membersDatalistDiv.innerHTML += optionHTML
        }
    }
}

function logInPageRedirect(errorMessage) {
    if(window.location.pathname !== "/login") {
        url = '/login'
        if(errorMessage) {
            url += '?error=' + encodeURI(errorMessage)
        }

        window.location = url;
        return true
    }
    return false
}

function verifyPageRedirect() {
    if(window.location.pathname !== "/verify") {
        window.location = '/verify';
        return true
    }
    return false
}

function accountPageRedirect() {
    if(window.location.pathname !== "/account") {
        window.location = '/account';
        return true
    }
    return false
}