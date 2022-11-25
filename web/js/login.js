function load_page(result) {

    // Reset cookie
    set_cookie("poenskelisten", "", 1);

    var html = `
                <div class="" id="forside">

                    <div class="module" id="news_feed">
                    </div>
                    
                    <div class="module">
                    
                        <div class="title">
                            Log in
                        </div>

                        <div class="text-body">
                            To view your wishes you need to login in...
                        </div>

                        <br>
                        <br>

                        <div class="action-block">
                            <form action="" onsubmit="event.preventDefault(); send_log_in();">

                                <hr>

                                <label id="form-input-icon" for="email"></label>
                                <input type="email" name="email" id="email" placeholder="Email" required/>

                                <label id="form-input-icon" for="password"></label>
                                <input type="password" name="password" id="password" placeholder="Password" required/>
                                
                                <hr>

                                <button id="log-in-button" type="submit" href="/">Log in</button>

                            </form>
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'What\'s the password?';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        info('Laster inn nyheter...');
        get_news();
    } else {
        showLoggedOutMenu();
    }
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

                window.location.href = '../../';

            }

        } else {
            info("Logging in...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/token/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(form_data);
    return false;
}

function clear_data() {
    document.getElementById("password").value = "";
    document.getElementById("email").value = "";
}

function disable_login_button() {
    document.getElementById("log-in-button").disabled = true;
}