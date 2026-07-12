function load_page(result) {

    var mfa_enrollment_required = false;

    if(result !== false) {

        try {

            var login_data = JSON.parse(result);

            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var user_id = login_data.data.id
            admin = login_data.data.admin;
            mfa_enrollment_required = login_data.mfa_enrollment_required === true;
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            var user_id = 0;
            admin = false;
        }

        showAdminMenu(admin)

    } else {
        var email = ""
        var first_name = ""
        var last_name = ""
        var user_id = 0;
    }

    try {
        string_index = document.URL.lastIndexOf('/');
        wishlist_id = document.URL.substring(string_index+1);

        group_id = 0
    }
    catch {
        group_id = 0
        wishlist_id = 0
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">

                        <div class="user-active-profile-photo">
                            <img style="width: 100%; height: 100%;" class="user-active-profile-photo-img" id="user-active-profile-photo-img" src="/assets/loading.svg">
                        </div>

                        <b><p id="user_name" style="font-size: 1.25em;"></p></b>
                        <p id="join_date" style=""></p>
                        <p id="user_admin" style=""></p>

                        <div class="module color-invert" id="" style="">
                            <hr>
                        </div>
                    
                        <form action="" class="icon-border" style="margin: 0 1em;" onsubmit="event.preventDefault(); send_update();">

                            <label id="form-input-icon" for="email"></label>
                            <input type="email" name="email" id="email" placeholder="Email" value="" required/>

                            <input class="clickable" onclick="change_password_toggle();" style="margin-top: 2em;" type="checkbox" id="password-toggle" name="password-toggle" value="confirm" >
                            <label for="password-toggle" class="clickable">Change my password.</label><br>

                            <div id="change-password-box" style="display:none;">

                                <label id="form-input-icon" for="password"></label>
                                <input type="password" name="password" id="password" placeholder="New password" />

                                <label id="form-input-icon" for="password_repeat"></label>
                                <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />

                            </div>

                            <label id="form-input-icon" for="new_profile_image" style="margin-top: 2em;">Replace profile image:</label>
                            <input type="file" name="new_profile_image" id="new_profile_image" placeholder="" value="" accept="image/png, image/jpeg" />

                            <label id="form-input-icon" for="password_original"></label>
                            <input type="password" name="password_original" id="password_original" placeholder="Your current password" required />

                            <button id="update-button" style="margin-top: 2em;" type="submit" href="/">Update account</button>

                        </form>

                    </div>

                    <div class="module" id="mfa-module">

                        <div class="module color-invert" id="" style="">
                            <hr>
                        </div>

                        <b><p style="font-size: 1.25em;">Two-factor authentication</p></b>
                        <p id="mfa-status" style="margin: 0 1em;">...</p>

                        <div id="mfa-action" style="margin: 1em;"></div>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();

        if(mfa_enrollment_required) {
            // Under enforcement the rest of the API is blocked until the user
            // enrolls, so skip the profile calls (they would 403) and take the
            // user straight to enrollment.
            error("Your administrator requires two-factor authentication. Please set it up to continue.");
            document.getElementById("mfa-status").innerHTML = "Required by your administrator, but not yet set up.";
            mfaEnroll();
        } else {
            GetUserData(user_id);
            GetProfileImage(user_id);
        }
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// Render the two-factor section based on whether MFA is currently enabled.
function renderMFASection(enabled) {
    var statusEl = document.getElementById("mfa-status");
    var actionEl = document.getElementById("mfa-action");
    if(!statusEl || !actionEl) {
        return;
    }

    if(enabled) {
        statusEl.innerHTML = "Enabled. Your account is protected by an authenticator app.";
        actionEl.innerHTML = `
            <button id="mfa-disable-button" onclick="renderMFADisable();" type="button" style="padding: 0.75em 1em;">Disable two-factor authentication</button>
        `;
    } else {
        statusEl.innerHTML = "Disabled. Add an authenticator app for extra protection.";
        actionEl.innerHTML = `
            <button id="mfa-enable-button" onclick="mfaEnroll();" type="button" style="padding: 0.75em 1em;">Enable two-factor authentication</button>
        `;
    }
}

// Begin enrollment: request a secret and show the activation form.
function mfaEnroll() {
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
                renderMFAActivate(result.secret, result.otpauth_url, result.qr_code);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/mfa/enroll");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// Show the QR code, manual secret, and a form to confirm the first code.
function renderMFAActivate(secret, otpauthURL, qrCode) {
    var actionEl = document.getElementById("mfa-action");
    if(!actionEl) {
        return;
    }

    var qrHTML = "";
    if(qrCode) {
        qrHTML = `<img src="${qrCode}" alt="Scan this QR code with your authenticator app" style="width: 12em; height: 12em; max-width: 100%; margin: 0.5em auto; display: block; image-rendering: pixelated;">`;
    }

    actionEl.innerHTML = `
        <p style="margin-bottom: 0.5em;">1. Scan this QR code with your authenticator app:</p>
        ${qrHTML}
        <p style="margin: 0.5em 0; font-size: 0.85em;">Can't scan it? Enter this key manually:</p>
        <p style="font-family: monospace; word-break: break-all; font-size: 1.1em;" id="mfa-secret">${secret}</p>
        <p style="word-break: break-all; font-size: 0.8em;"><a href="${otpauthURL}">Open in authenticator app</a></p>

        <form action="" class="icon-border" style="margin-top: 1em;" onsubmit="event.preventDefault(); mfaActivate();">
            <p style="margin-bottom: 0.5em;">2. Enter the 6-digit code to confirm:</p>
            <label id="form-input-icon" for="mfa_activate_code"></label>
            <input type="text" name="mfa_activate_code" id="mfa_activate_code" placeholder="6-digit code" autocomplete="one-time-code" inputmode="numeric" required/>
            <button id="mfa-activate-button" type="submit" style="padding: 0.75em 1em;">Confirm and enable</button>
        </form>

        <p style="margin-top: 0.5em; font-size: 0.8em; cursor: pointer;"><a onclick="renderMFASection(false);">Cancel</a></p>
    `;
}

// Confirm the code, enabling MFA and revealing recovery codes.
function mfaActivate() {
    var code = document.getElementById("mfa_activate_code").value;

    var form_data = JSON.stringify({ "code" : code });

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
                    document.getElementById("mfa_activate_code").value = "";
                } catch(e) {
                    console.log(e)
                }
            } else {
                success(result.message);
                if(result.recovery_codes && result.recovery_codes.length) {
                    document.getElementById("mfa-status").innerHTML = "Enabled. Save your recovery codes below.";
                    showRecoveryCodes(result.recovery_codes);
                } else {
                    // Recovery codes are disabled by the administrator.
                    renderMFASection(true);
                }
            }
        } else {
            info("Enabling two-factor authentication...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/mfa/activate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

// Display the one-time recovery codes. These are only shown once.
function showRecoveryCodes(codes) {
    var actionEl = document.getElementById("mfa-action");
    if(!actionEl) {
        return;
    }

    var codesHTML = "";
    if(codes && codes.length) {
        for(var i = 0; i < codes.length; i++) {
            codesHTML += `<div style="font-family: monospace; font-size: 1.05em;">${codes[i]}</div>`;
        }
    }

    actionEl.innerHTML = `
        <p style="margin-bottom: 0.5em;"><b>Save these recovery codes.</b> Each can be used once if you lose access to your authenticator. They won't be shown again.</p>
        <div id="mfa-recovery-codes" style="margin: 0.5em auto; padding: 0.5em; border: 1px solid; max-width: 14em; text-align: center;">
            ${codesHTML}
        </div>
        <br>
        <button type="button" onclick="renderMFASection(true);" style="padding: 0.75em 1em;">I've saved my codes</button>
    `;
}

// Show the form required to turn MFA off.
function renderMFADisable() {
    var actionEl = document.getElementById("mfa-action");
    if(!actionEl) {
        return;
    }

    actionEl.innerHTML = `
        <form action="" class="icon-border" onsubmit="event.preventDefault(); mfaDisable();">
            <p style="margin-bottom: 0.5em;">Confirm your password and a current code to disable two-factor authentication.</p>

            <label id="form-input-icon" for="mfa_disable_password"></label>
            <input type="password" name="mfa_disable_password" id="mfa_disable_password" placeholder="Your current password" required/>

            <label id="form-input-icon" for="mfa_disable_code"></label>
            <input type="text" name="mfa_disable_code" id="mfa_disable_code" placeholder="Authenticator or recovery code" autocomplete="one-time-code" required/>

            <button id="mfa-disable-confirm-button" type="submit" style="padding: 0.75em 1em;">Disable</button>
        </form>

        <p style="margin-top: 0.5em; font-size: 0.8em; cursor: pointer;"><a onclick="renderMFASection(true);">Cancel</a></p>
    `;
}

// Send the disable request.
function mfaDisable() {
    var password = document.getElementById("mfa_disable_password").value;
    var code = document.getElementById("mfa_disable_code").value;

    var form_data = JSON.stringify({ "password" : password, "code" : code });

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
                success(result.message);
                renderMFASection(false);
            }
        } else {
            info("Disabling two-factor authentication...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/mfa/disable");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function change_password_toggle() {

    var check_box = document.getElementById("password-toggle").checked;
    var password_box = document.getElementById("change-password-box")

    if(check_box) {
        password_box.style.display = "inline-block"
    } else {
        password_box.style.display = "none"
    }

}

function send_update() {

    var email = document.getElementById("email").value;
    var password = document.getElementById("password").value;
    var password_repeat = document.getElementById("password_repeat").value;
    var password_original = document.getElementById("password_original").value;
    var new_profile_image = document.getElementById('new_profile_image').files[0];

    if(new_profile_image) {

        if(new_profile_image.size > 10000000) {
            error("Image exceeds 10MB size limit.")
            return;
        } else if(new_profile_image.size < 10000) {
            error("Image smaller than 0.01MB size requirement.")
            return;
        }

        new_profile_image = get_base64(new_profile_image);
        
        new_profile_image.then(function(result) {

            var form_obj = { 
                "email" : email,
                "password" : password,
                "password_repeat": password_repeat,
                "profile_image": result,
                "password_original": password_original
            };

            var form_data = JSON.stringify(form_obj);

            document.getElementById("user-active-profile-photo-img").src = 'assets/loading.svg';

            send_update_two(form_data);
        
        });

    } else {

        var form_obj = { 
            "email" : email,
            "password" : password,
            "password_repeat": password_repeat,
            "password_original": password_original,
            "profile_image": ""
        };

        var form_data = JSON.stringify(form_obj);
    
        send_update_two(form_data);
    }

}

function send_update_two(form_data) {
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

                success(result.message);

                // store jwt to cookie
                set_cookie("poenskelisten", result.token, 7);

                if(result.verified) {
                    location.reload();
                } else {
                    location.href = './';
                }
                
            }

        } else {
            info("Updating account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function GetProfileImage(userID) {

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

                PlaceProfileImage(result.image)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImage(imageBase64) {

    document.getElementById("user-active-profile-photo-img").src = imageBase64

}

function GetUserData(userID) {
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

                PlaceUserData(result.user)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;
}

function PlaceUserData(user_object) {
    document.getElementById("user_name").innerHTML = user_object.first_name + " " + user_object.last_name
    document.getElementById("email").value = user_object.email

    // parse date object
    try {
        var date = new Date(Date.parse(user_object.created_at));
        var date_string = GetDateString(date)
    } catch(e) {
        var date_string = "Error"
        console.log("Join date error: " + e)
    }

    document.getElementById("join_date").innerHTML = "Joined: " + date_string

    if(user_object.admin) {
        var admin_string = "Yes"
    } else {
        var admin_string = "No"
    }

    document.getElementById("user_admin").innerHTML = "Administrator: " + admin_string

    // Reflect the current two-factor state.
    renderMFASection(user_object.mfa_enabled === true)
}