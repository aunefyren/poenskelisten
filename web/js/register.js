function load_page(result) {

    // Reset cookie
    set_cookie("poenskelisten", "", 1);

    var html = `
                <div class="" id="forside">

                    <div class="module" id="news_feed">
                    </div>
                    
                    <div class="module">
                    
                        <div class="title">
                            Register
                        </div>

                        <div class="text-body">
                            Did you get an invitation?
                        </div>

                        <br>
                        <br>

                        <div class="action-block">
                            <form action="" onsubmit="event.preventDefault(); send_registration();">

                                <hr>

                                <label id="form-input-icon" for="email"></label>
                                <input type="email" name="email" id="email" placeholder="Email" required/>

                                <label id="form-input-icon" for="first_name"></label>
                                <input type="text" name="first_name" id="first_name" placeholder="First name" required/>

                                <label id="form-input-icon" for="last_name"></label>
                                <input type="text" name="last_name" id="last_name" placeholder="Last name" required/>

                                <label id="form-input-icon" for="password"></label>
                                <input type="password" name="password" id="password" placeholder="Password" required/>

                                <label id="form-input-icon" for="password_repeat"></label>
                                <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" required/>

                                <label id="form-input-icon" for="invitation_code"></label>
                                <input type="text" name="invitation_code" id="invitation_code" placeholder="Invitation code" required/>
                                
                                <input style="margin-top: 2em;" type="checkbox" id="confirm" name="confirm" value="confirm" required>
                                <label for="confirm"> I confirm that Pønskelisten can store relevant information about me and that I am atleast thirteen years of age.</label><br>

                                <hr>

                                <button id="register-button" type="submit" href="/">Register</button>

                            </form>
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Tell me more about yourself.';
    clearResponse();
    showLoggedOutMenu();
}

function send_registration(){

    var user_email = document.getElementById("email").value;
    var user_password = document.getElementById("password").value;
    var user_password_repeat = document.getElementById("password_repeat").value;
    var user_first_name = document.getElementById("first_name").value;
    var user_last_name = document.getElementById("last_name").value;
    var invitation_code = document.getElementById("invitation_code").value;

    var form_obj = { 
                                    "email" : user_email,
                                    "password" : user_password,
                                    "password_repeat": user_password_repeat,
                                    "first_name": user_first_name,
                                    "last_name": user_last_name,
                                    "invite_code": invitation_code
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

                success(result.message);
                clear_data();
                disable_register_button();

            }

        } else {
            info("Registering...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/user/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(form_data);
    return false;
}

function clear_data() {
    document.getElementById("password").value = "";
    document.getElementById("password_repeat").value = ""
}

function disable_register_button() {
    document.getElementById("register-button").disabled = true;
}