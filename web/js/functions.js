var api_url = "http://localhost:8080/api/"

// Load service worker
window.addEventListener("load", () => {
    if ("serviceWorker" in navigator) {
        navigator.serviceWorker.register("jss/service-worker.js");
    }
});

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
    var d = new Date();
    d.setTime(d.getTime() + (exdays*24*60*60*1000));
    var expires = "expires="+ d.toUTCString();
    document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/;samesite=strict";
}

// Get cookie from browser
function get_cookie(cname) {
    var name = cname + "=";
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

// Validate login token and get login details
function get_login(cookie) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            var result;
            if(result = JSON.parse(this.responseText)) { 
                load_page(this.responseText);
            } else {
                load_page(false);
            }
        } else if(this.readyState == 4 && this.status !== 200) {
            load_page(false);
        }
    };
    xhttp.withCredentials = false;
    xhttp.open("post", api_url + "auth/token/validate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", cookie);
    xhttp.send();
    return;
}

// Called when login session was rejected, showing no content and an error
function invalid_session() {
    showLoggedOutMenu();
    document.getElementById('content').innerHTML = '';
    document.getElementById('card-header').innerHTML = 'Nope...';
    error('Ingen tilgang.');
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

    document.getElementById('account').classList.add('disabled');
    document.getElementById('account').classList.remove('enabled');

    document.getElementById('register').classList.add('enabled');
    document.getElementById('register').classList.remove('disabled');
}

// Toggle navar expansion
function toggle_navbar() {
    var x = document.getElementById("navbar");
    var y = document.getElementById("nav-logo");
    if (x.className === "navbar") {
      x.className += " responsive";
      y.className += " responsive";
    } else {
      x.className = "navbar";
      y.className = "nav-logo";
    }
}

// Toggle navbar if clicked outside
document.addEventListener('click', function(event) {
    var isClickInsideElement = ignoreNav.contains(event.target);
    if (!isClickInsideElement) {
        var nav_classlist = document.getElementById('navbar').classList;
        if (nav_classlist.contains('responsive')) {
            toggle_navbar();
        }
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
}

// Displays a red notification
function error(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-danger'>" + message + "</div>";
    window.scrollTo(0, 0);
}

// When log out button is pressed, remove cookie and redirect to home page
function logout() {
    set_cookie("poenskelisten", "", 1);
    window.location.href = './';
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