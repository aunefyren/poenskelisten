function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
        }
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
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <form action="" onsubmit="event.preventDefault(); send_update();">

                            <label id="form-input-icon" for="email"></label>
                            <input type="email" name="email" id="email" placeholder="Email" value="` + email + `" required/>

                            <label id="form-input-icon" for="first_name"></label>
                            <input type="text" name="first_name" id="first_name" placeholder="First name" value="` + first_name + `" required disabled />

                            <label id="form-input-icon" for="last_name"></label>
                            <input type="text" name="last_name" id="last_name" placeholder="Last name" value="` + last_name + `" disabled required/>

                            <input onclick="change_password_toggle();" style="margin-top: 2em;" type="checkbox" id="password-toggle" name="confirm" value="confirm" >
                            <label for="confirm">Change my password.</label><br>

                            <div id="change-password-box" style="display:none;">

                                <label id="form-input-icon" for="password"></label>
                                <input type="password" name="password" id="password" placeholder="New password" />

                                <label id="form-input-icon" for="password_repeat"></label>
                                <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />

                            </div>

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

    var form_obj = { 
                        "email" : email,
                        "password" : password,
                        "password_repeat": password_repeat
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
    xhttp.open("post", api_url + "auth/user/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}