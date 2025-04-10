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
                <div class="server-info-line"><div class="server-info-title" id="server-poenskelisten-version-title">Version:</div><div class="server-info-value" id="server-poenskelisten-version">...</div></div>
                <div class="server-info-line"><div class="server-info-title" id="server-poenskelisten-port-title">Port:</div><div class="server-info-value" id="server-poenskelisten-port">...</div></div>
                <div class="server-info-line"><div class="server-info-title" id="server-poenskelisten-database-title">Database:</div><div class="server-info-value" id="server-poenskelisten-database">...</div></div>
                <div class="server-info-line"><div class="server-info-title" id="server-poenskelisten-url-title">External URL:</div><div class="server-info-value" id="server-poenskelisten-url">...</div></div>
                <div class="server-info-line"><div class="server-info-title" id="server-timezone-title">Timezone:</div><div class="server-info-value" id="server-timezone">...</div></div>
                <div class="server-info-line"><div class="server-info-title" id="server-poenskelisten-loglevel-title">Log level:</div><div class="server-info-value" id="server-poenskelisten-loglevel">...</div></div>
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

function place_server_info(server_info) {
    document.getElementById('server-poenskelisten-version').innerHTML = server_info.poenskelisten_version
    document.getElementById('server-timezone').innerHTML = server_info.timezone
    document.getElementById('server-poenskelisten-url').innerHTML = server_info.poenskelisten_external_url
    document.getElementById('server-poenskelisten-database').innerHTML = server_info.database_type
    document.getElementById('server-poenskelisten-port').innerHTML = server_info.poenskelisten_port
    document.getElementById('server-poenskelisten-loglevel').innerHTML = server_info.poenskelisten_log_level
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
            </div>
        </div>

        <div id="user-input" class="user-input" style="width: 100%;">
            <button id="register-button" onClick="deleteUser('${user_object.id}');" type="" href="/">Delete user</button>
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