function load_page(result) {

    if(result !== false) {
       
        try {

            var login_data = JSON.parse(result);

            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            admin = login_data.data.admin;
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            admin = false;
        }

        showAdminMenu(admin)

    } else {
        var email = ""
        var first_name = ""
        var last_name = ""
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
        <!-- The Modal -->
        <div id="myModal" class="modal closed">
            <span class="close clickable" style="padding: 0 0.25em;" onclick="toggleModal()">&times;</span>
            <div class="modalContent" id="modalContent">
            </div>
            <div id="caption"></div>
        </div>

        <div class="modules" id="admin-page">
            
            <div class="server-info" id="server-info">
                <h3 id="server-info-title">Server info:</h3>
                <div id="server-info-body">
                    <div class="server-info-line"><div class="server-info-title">Loading</div><div class="server-info-value is-muted">…</div></div>
                </div>
            </div>

            <div class="invites" id="invites">
                <h3 id="invitation-module-title">Invites:</h3>
                <div class="invite-list" id="invite-list">
                </div>
                <button type="submit" onclick="generate_invite();" id="generate_invite_button" style=""><img src="assets/plus.svg" class="btn_logo"><p2>Generate</p2></button>
            
            </div>

            <div class="currency-module" id="currency-module">

                <h3 id="currency-module-title">Currency:</h3>

                <input type="text" name="currency" id="currency" placeholder="What currency can wishes be listed in?" value="" autocomplete="off" required />

                <input class="clickable" onclick="" style="margin-top: 0.5em;" type="checkbox" id="currency-padding" name="currency-padding" value="confirm" >
                <label for="currency-padding" class="clickable">Pad the currency string</label><br>

                <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="currency-left" name="currency-left" value="confirm" >
                <label for="currency-left" class="clickable">Currency on the left side</label><br>

                <button type="submit" onclick="update_currency();" id="update_currency_button" style=""><img src="assets/check.svg" class="btn_logo"><p2>Update</p2></button>

            </div>

            <div class="security-module" id="security-module">

                <h3 id="security-module-title">Security:</h3>

                <input class="clickable" style="margin-top: 0.5em;" type="checkbox" id="mfa-enforced" name="mfa-enforced" value="confirm" >
                <label for="mfa-enforced" class="clickable">Require all local users to set up two-factor authentication</label><br>

                <input class="clickable" style="margin-top: 1em;" type="checkbox" id="mfa-recovery-codes" name="mfa-recovery-codes" value="confirm" >
                <label for="mfa-recovery-codes" class="clickable">Issue recovery codes when users enrol</label><br>

                <button type="submit" onclick="update_server_settings();" id="update_security_button" style=""><img src="assets/check.svg" class="btn_logo"><p2>Update</p2></button>

            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Ultimate power';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();

        if(!admin) {
            document.getElementById('content').innerHTML = "...";
            error("You are not an admin.")
        } else {
            get_server_info();
            get_invites();
            get_currency();
        }

    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_server_info() {

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

                place_server_info(result.server)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/server/info");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

// Escape untrusted config strings before injecting them into the panel.
function escapeServerInfo(value) {
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

// Build one label + value chip. kind: "text" | "mono" | "bool".
function serverInfoRow(label, value, kind) {
    var chip;

    if(kind === "bool") {
        chip = value
            ? '<div class="server-info-value is-on">Enabled</div>'
            : '<div class="server-info-value is-off">Disabled</div>';
    } else {
        var text = (value === null || value === undefined) ? "" : String(value);
        if(text.trim() === "") {
            chip = '<div class="server-info-value is-muted">Not set</div>';
        } else if(kind === "mono") {
            chip = '<div class="server-info-value is-mono">' + escapeServerInfo(text) + '</div>';
        } else {
            chip = '<div class="server-info-value">' + escapeServerInfo(text) + '</div>';
        }
    }

    return '<div class="server-info-line"><div class="server-info-title">' + escapeServerInfo(label) + '</div>' + chip + '</div>';
}

// Wrap a set of rows under a small section heading.
function serverInfoGroup(title, rows) {
    return '<div class="server-info-group"><div class="server-info-group-title">' + escapeServerInfo(title) + '</div>' + rows.join("") + '</div>';
}

function place_server_info(server_info) {
    var groups = [];

    // Application
    var application = [
        serverInfoRow("Name", server_info.app_name, "text"),
        serverInfoRow("Version", server_info.poenskelisten_version, "text"),
        serverInfoRow("Environment", server_info.poenskelisten_environment, "text"),
        serverInfoRow("External URL", server_info.poenskelisten_external_url, "mono"),
        serverInfoRow("Port", server_info.poenskelisten_port, "text"),
        serverInfoRow("Timezone", server_info.timezone, "text"),
        serverInfoRow("Log level", server_info.poenskelisten_log_level, "text")
    ];
    if((server_info.poenskelisten_environment || "").toLowerCase() === "test") {
        application.push(serverInfoRow("Test email", server_info.poenskelisten_test_email, "mono"));
    }
    groups.push(serverInfoGroup("Application", application));

    // Database
    var database = [ serverInfoRow("Type", server_info.database_type, "text") ];
    if((server_info.database_type || "").toLowerCase() === "sqlite") {
        database.push(serverInfoRow("File", server_info.database_location, "mono"));
    } else {
        database.push(serverInfoRow("Name", server_info.database_name, "text"));
        database.push(serverInfoRow("Host", server_info.database_host, "mono"));
        database.push(serverInfoRow("Port", server_info.database_port, "text"));
        database.push(serverInfoRow("SSL", server_info.database_ssl, "bool"));
    }
    groups.push(serverInfoGroup("Database", database));

    // Email
    var email = [ serverInfoRow("Status", server_info.smtp_enabled, "bool") ];
    if(server_info.smtp_enabled) {
        email.push(serverInfoRow("Host", server_info.smtp_host, "mono"));
        email.push(serverInfoRow("Port", server_info.smtp_port, "text"));
        email.push(serverInfoRow("From", server_info.smtp_from, "mono"));
    }
    groups.push(serverInfoGroup("Email", email));

    // Single sign-on
    var sso = [ serverInfoRow("Status", server_info.oidc_enabled, "bool") ];
    if(server_info.oidc_enabled) {
        sso.push(serverInfoRow("Provider", server_info.oidc_provider_name, "text"));
        sso.push(serverInfoRow("Issuer", server_info.oidc_issuer_url, "mono"));
        sso.push(serverInfoRow("Client ID", server_info.oidc_client_id, "mono"));
        sso.push(serverInfoRow("Redirect URL", server_info.oidc_redirect_url, "mono"));
        sso.push(serverInfoRow("Auto-create users", server_info.oidc_auto_create_users, "bool"));
    }
    groups.push(serverInfoGroup("Single sign-on", sso));

    // Security
    groups.push(serverInfoGroup("Security", [
        serverInfoRow("Require MFA", server_info.mfa_enforced, "bool"),
        serverInfoRow("Recovery codes", server_info.mfa_recovery_codes_enabled, "bool")
    ]));

    document.getElementById("server-info-body").innerHTML = groups.join("");

    // Keep the editable security toggles in sync with the reported state.
    document.getElementById('mfa-enforced').checked = server_info.mfa_enforced === true
    document.getElementById('mfa-recovery-codes').checked = server_info.mfa_recovery_codes_enabled === true
}

function update_server_settings() {

    var mfaEnforced = document.getElementById('mfa-enforced').checked;
    var mfaRecoveryCodes = document.getElementById('mfa-recovery-codes').checked;

    var form_data = JSON.stringify({ "mfa_enforced" : mfaEnforced, "mfa_recovery_codes_enabled" : mfaRecoveryCodes });

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
                success(result.message)
                document.getElementById('mfa-enforced').checked = result.mfa_enforced === true
                document.getElementById('mfa-recovery-codes').checked = result.mfa_recovery_codes_enabled === true
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/server/settings");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function get_invites() {

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

                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "admin/invites");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_invites(invites_array) {
    var html = ``;
    
    if(invites_array.length == 0) {
        html = `
            <div id="" class="invitation-object">
                <p id="" style="margin: 0.5em; text-align: center;">...</p>
            </div>
        `;
    } else {
        for(var i = 0; i < invites_array.length; i++) {
            html += `
                <div id="" class="invitation-object">
                    <div class="leaderboard-object-code">
                        Code: ` + invites_array[i].invite_code + `
                    </div>
            `;

            if(invites_array[i].invite_used) {
                html += `
                        <div class="leaderboard-object-user clickable" onclick="GetUserData('${invites_array[i].user.id}')">
                            Used by: ` + invites_array[i].user.first_name + ` ` + invites_array[i].user.last_name + `
                        </div>
                    `;
            } else {
                html += `
                        <div class="leaderboard-object-user">
                            Not used
                        </div>
                        <img class="icon-img clickable" onclick="delete_invite('${invites_array[i].id}')" src="/assets/trash-2.svg"></img>
                    `;
            }

            html += `</div>`;

        }
        
    }

    document.getElementById("invite-list").innerHTML = html

    return
}

function generate_invite() {

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

                success(result.message)
                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/invites");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function delete_invite(invide_id) {

    if(!confirm("Are you sure you want to delete this invite?")) {
        return
    }

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

                success(result.message)
                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "admin/invites/" + invide_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function get_currency() {
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
                //console.log(result)
                document.getElementById('currency').value = result.currency;
                document.getElementById('currency-padding').checked = result.padding;
                document.getElementById('currency-left').checked = result.left;
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/currency");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function update_currency() {

    var currency = document.getElementById('currency').value;
    var padding = document.getElementById('currency-padding').checked;
    var left = document.getElementById('currency-left').checked;

    var form_obj = { 
        "poenskelisten_currency" : currency,
        "poenskelisten_currency_pad": padding,
        "poenskelisten_currency_left": left
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

            } else {

                success(result.message)
                document.getElementById('currency').value = result.currency;
                document.getElementById('currency-padding').checked = result.padding;
                document.getElementById('currency-left').checked = result.left;
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/currency/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

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
                PlaceUserDataInModal(result.user)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;
}

function PlaceUserDataInModal(user_object) {
    displayName = "Name:  " + user_object.first_name + " " + user_object.last_name
    email = "Email: " + user_object.email

    // parse date object
    try {
        var date = new Date(Date.parse(user_object.created_at));
        var date_string = GetDateString(date)
    } catch(e) {
        var date_string = "Error"
        console.log("Join date error: " + e)
    }

    joinedDate = "Joined: " + date_string

    if(user_object.admin) {
        var admin_string = "Yes"
    } else {
        var admin_string = "No"
    }

    adminString = "Administrator: " + admin_string

    if(user_object.mfa_enabled) {
        var mfa_string = "Two-factor: Enabled"
    } else {
        var mfa_string = "Two-factor: Disabled"
    }

    var mfaButton = ""
    if(user_object.mfa_enabled) {
        mfaButton = `<button id="delete-mfa-button" onClick="adminDeleteUserMFA('${user_object.id}');" type="button" style="margin-top: 0.5em; padding: 0.75em 1em;">Delete two-factor authentication</button>`
    }

    html = `
        <div class="user-wrapper">
            <div class="profile-icon icon-border icon-background" id="wishlist_owner_image_${user_object.id}" style="width: 5em; height: 5em;">
                <img class="icon-img " src="/assets/user.svg" id="wishlist_owner_image_img_${user_object.id}">
            </div>
            <div class="user-infolist">
                ${displayName}<br>
                ${email}<br>
                ${joinedDate}<br>
                ${adminString}<br>
                ${mfa_string}<br>
            </div>
        </div>

        <div id="user-input" class="user-input" style="width: 100%;">
            <button id="register-button" onClick="deleteUser('${user_object.id}');" type="" href="/">Delete user</button>
            ${mfaButton}
        </div>
    `;

    toggleModal(html);
    GetProfileImage(user_object.id, `wishlist_owner_image_${user_object.id}`)
}

function GetProfileImage(userID, divID) {
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
                    PlaceProfileImage(result.image, divID)
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

function PlaceProfileImage(imageBase64, divID) {
    var image = document.getElementById(divID)
    image.style.backgroundSize = "cover"
    image.innerHTML = ""
    image.style.backgroundImage = `url('${imageBase64}')`
    image.style.backgroundPosition = "center center"
}

function adminDeleteUserMFA(userID) {
    if(!confirm("Remove two-factor authentication for this user? They will be able to log in with just their password.")) {
        return;
    }

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
                toggleModal(false);
                success(result.message);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "admin/users/" + userID + "/mfa");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

function deleteUser(userID) {
    if(!confirm("Are you sure you want to delete this user?")) {
        return;
    }

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
                toggleModal(false);
                get_invites();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "admin/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}