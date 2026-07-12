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

        <div id="oidc-login" style="margin-top: 1em;"></div>
    </div>
    `;

    var html2 = `
    <a style="font-size:0.75em;cursor:pointer;" onclick="action_newpassword();">I forgot my password!</i>
    `;

    document.getElementById("action").innerHTML = html;
    document.getElementById("change_action").innerHTML = html2;

    renderOIDCLoginOption();
}

// Fetch the public OIDC config and, if single sign-on is enabled, show a button
// that starts the flow. Runs quietly: any failure just leaves password login.
function renderOIDCLoginOption() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log("Failed to parse OIDC config. Error: " + e);
                return;
            }

            var container = document.getElementById("oidc-login");
            if(!container || !result.enabled) {
                return;
            }

            var providerName = result.provider_name || "single sign-on";
            container.innerHTML = `
                <div class="text-body" style="font-size: 0.8em; margin-bottom: 0.5em;">or</div>
                <button type="button" style="padding: 0.75em 1em;" onclick="window.location.href='${result.login_url}';">Log in with ${providerName}</button>
            `;
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "open/oidc/config");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send();
    return;
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

            } else if(result.mfa_required) {

                // Password accepted, but a second factor is required. Show the
                // code entry step carrying the short-lived challenge token.
                clear_data();
                action_mfa(result.mfa_token);

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

function action_mfa(mfaToken) {

    clearResponse();

    var html = `
    <div class="title">
        Two-factor authentication
    </div>

    <div class="text-body">
        Enter the code from your authenticator app. You can also use one of your recovery codes.
    </div>

    <br>
    <br>

    <div class="action-block">
        <form action="" class="icon-border" onsubmit="event.preventDefault(); send_mfa_code();">

            <label id="form-input-icon" for="mfa_code"></label>
            <input type="text" name="mfa_code" id="mfa_code" placeholder="Authenticator or recovery code" autocomplete="one-time-code" inputmode="text" required autofocus/>

            <input type="hidden" name="mfa_token" id="mfa_token" value="` + mfaToken + `" />

            <button id="log-in-button" type="submit" href="/">Verify</button>

        </form>
    </div>
    `;

    var html2 = `
    <a style="font-size:0.75em;cursor:pointer;" onclick="action_login();">Back to log in</i>
    `;

    document.getElementById("action").innerHTML = html;
    document.getElementById("change_action").innerHTML = html2;
}

function send_mfa_code() {

    var mfa_token = document.getElementById("mfa_token").value;
    var mfa_code = document.getElementById("mfa_code").value;

    var form_obj = {
        "mfa_token" : mfa_token,
        "code" : mfa_code
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
                return;
            }

            if(result.error) {

                error(result.error);
                try {
                    document.getElementById("mfa_code").value = "";
                } catch(e) {
                    console.log(e)
                }

            } else {

                // store jwt to cookie
                set_cookie("poenskelisten", result.token, 7);

                showLoggedInMenu();
                success(result.message);
                disable_login_button();

                window.location.href = '/';

            }

        } else {
            info("Verifying code...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/tokens/mfa");
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