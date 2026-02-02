function load_page(result) {

    // Reset cookie
    set_cookie("poenskelisten", "", 1);

    var html = `
        <div class="" id="forside">

            <div class="module" id="action">
            </div>

            <div class="module" id="change_action">
            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'What\'s the password?';
    clearResponse();

    var reset_mode = false;
    var reset_code = "";
    var errorMessage = "";
    try {
        // Get parameters from URL string
        var url_string = window.location.href
        var url = new URL(url_string);

        var reset_code = url.searchParams.get("reset_code");
        if(reset_code !== null) {
            reset_mode = true;
        }

        var errorMessage = url.searchParams.get("error");
    } catch(e) {
        reset_mode = false;
        reset_code = ""
    }

    showLoggedOutMenu();

    if(reset_mode) {
        clearResponse();
        checkResetCode(reset_code);
    } else {
        clearResponse();
        action_login();
        if(errorMessage) {
            error(errorMessage);
        }
    }
}

function action_login() {
    try {
        var email = document.getElementById("email").value;
    } catch(e) {
        var email = "";
    }

    var html = `
    <div class="title">
        Log in
    </div>

    <div class="text-body">
        To make a wish you need to login...
    </div>

    <br>
    <br>

    <div class="action-block">
        <form action="" class="icon-border" onsubmit="event.preventDefault(); send_log_in();">

            <label id="form-input-icon" for="email"></label>
            <input type="email" name="email" id="email" value="` + email + `" placeholder="Email" required/>

            <label id="form-input-icon" for="password"></label>
            <input type="password" name="password" id="password" placeholder="Password" required/>

            <button id="log-in-button" type="submit" href="/">Log in</button>

        </form>
    </div>
    `;

    var html2 = `
    <a style="font-size:0.75em;cursor:pointer;" onclick="action_newpassword();">I forgot my password!</i>
    `;

    document.getElementById("action").innerHTML = html;
    document.getElementById("change_action").innerHTML = html2;
}

function action_newpassword() {

    try {
        var email = document.getElementById("email").value;
    } catch(e) {
        var email = "";
    }

    var html = `
    <div class="title">
        Reset password
    </div>

    <div class="text-body">
        It's okay to forget.
    </div>

    <br>
    <br>

    <div class="action-block">
        <form action="" class="icon-border" onsubmit="event.preventDefault(); reset_password_request();">

            <label id="form-input-icon" for="email"></label>
            <input type="email" name="email" id="email" value="` + email + `" placeholder="Email" required/>

            <button id="reset-button" type="submit" href="/">Reset password</button>

        </form>
    </div>
    `;

    var html2 = `
    <a style="font-size:0.75em;cursor:pointer;" onclick="action_login();">Log in!</i>
    `;

    document.getElementById("action").innerHTML = html;
    document.getElementById("change_action").innerHTML = html2;

}

function action_resetpassword(reset_code) {

    clearResponse();

    var html = `
    <div class="title">
        Change password
    </div>

    <div class="text-body">
        Pick something you'll remember this time.
    </div>

    <br>
    <br>

    <div class="action-block">
        <form action="" class="icon-border" onsubmit="event.preventDefault(); reset_password();">

            <label id="form-input-icon" for="password"></label>
            <input type="password" name="password" id="password" placeholder="New password" />

            <label id="form-input-icon" for="password_repeat"></label>
            <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />

            <input type="hidden" name="reset_code" id="reset_code" value="` + reset_code + `" />

            <button id="reset-button" type="submit" href="/">Change password</button>

        </form>
    </div>
    `;

    var html2 = `
    <a style="font-size:0.75em;cursor:pointer;" onclick="action_login();">Log in!</i>
    `;

    document.getElementById("action").innerHTML = html;
    document.getElementById("change_action").innerHTML = html2;

}

function send_log_in(){

    var user_email = document.getElementById("email").value;
    var user_password = document.getElementById("password").value;

    var form_obj = { 
        "email" : user_email,
        "password" : user_password
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                clear_data();
                return;
            }
            
            if(result.error) {

                error(result.error);
                clear_data();

            } else {

                // store jwt to cookie
                set_cookie("poenskelisten", result.token, 7);

                // show home page &amp; tell the user it was a successful login
                showLoggedInMenu();
                success(result.message);
                clear_data();
                disable_login_button();

                window.location.href = '/';

            }

        } else {
            info("Logging in...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/tokens/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(form_data);
    return false;
}

function clear_data() {
    try {
        document.getElementById("password").value = "";
    } catch(e) {
        console.log(e)
    }

    try {
        document.getElementById("password_repeat").value = "";
    } catch(e) {
        console.log(e)
    }
    
    try {
        document.getElementById("email").value = "";
    } catch(e) {
        console.log(e)
    }
}

function disable_login_button() {
    document.getElementById("log-in-button").disabled = true;
}

function reset_password_request(){

    var user_email = document.getElementById("email").value;

    var form_obj = { 
        "email" : user_email
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                clear_data();
                return;
            }
            
            if(result.error) {

                error(result.error);
                clear_data();

            } else {

                // store jwt to cookie
                success(result.message)

            }

        } else {
            info("Sending request...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/reset");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(form_data);
    return false;
}

function reset_password(){

    var password = document.getElementById("password").value;
    var password_repeat = document.getElementById("password_repeat").value;
    var reset_code = document.getElementById("reset_code").value;

    var form_obj = { 
        "reset_code": reset_code,
        "password" : password,
        "password_repeat" : password_repeat
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                clear_data();
                return;
            }
            
            if(result.error) {

                error(result.error);
                clear_data();

            } else {

                // store jwt to cookie
                success(result.message)
                clear_data();
                action_login();

            }

        } else {
            info("Changing password...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/password");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(form_data);
    return false;
}

function checkResetCode(resetCode){
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                clear_data();
                return;
            }
            
            if(result.error) {
                error(result.error);
                clear_data();
            } else {
                if(result.expired) {
                    error("Reset link has expired.")
                    action_login();
                } else {
                    action_resetpassword(resetCode);
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "open/users/reset/" + resetCode);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send();
    return false;
}