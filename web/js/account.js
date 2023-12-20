function load_page(result) {

    if(result !== false) {
        
        try {

            var login_data = JSON.parse(result);
            
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var user_id = login_data.data.id
            admin = login_data.data.admin;
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
                    
                        <form action="" class="icon-border" style="margin: 0 1em;" onsubmit="event.preventDefault(); send_update();">

                            <label id="form-input-icon" for="email"></label>
                            <input type="email" name="email" id="email" placeholder="Email" value="` + email + `" required/>

                            <label id="form-input-icon" for="first_name"></label>
                            <input type="text" name="first_name" id="first_name" placeholder="First name" value="` + first_name + `" required disabled />

                            <label id="form-input-icon" for="last_name"></label>
                            <input type="text" name="last_name" id="last_name" placeholder="Last name" value="` + last_name + `" disabled required/>

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

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        GetProfileImage(user_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
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