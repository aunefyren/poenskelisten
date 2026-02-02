function load_page(result) {

    if(result !== false) {

        try {
            var login_data = JSON.parse(result);

            if(login_data.error && login_data.error.toLowerCase().includes("you must verify your account")) {
                console.log("validate flow")
                load_verify_account();
                return;
            } else {
                console.log("front page redirect")
                console.log(result)
                // frontPageRedirect();
            }

            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            admin = login_data.data.admin;
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            admin = false;

            console.log("failed to parse response from validation API")
        }

        showAdminMenu(admin)

    } else {
        var login_data = false;
        var email = ""
        var first_name = ""
        var last_name = ""
        admin = false;
        console.log("no response from validation API")
    }

    var html = `
        <div class="" id="front-page">
            
            ...

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {

        showLoggedInMenu();
  
    } else {
        showLoggedOutMenu();
        document.getElementById('main-text').innerHTML = "You need to login.";
        document.getElementById('log-in-button').style.display = 'inline-block';
    }
}

function load_verify_account() {

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Pønskelisten
                        </div>

                        <div class="text-body" style="text-align: center;">
                            You must verify your account by giving us the access code we e-mailed you.
                        </div>

                    </div>

                    <div class="module">

                        <form action="" class="icon-border" onsubmit="event.preventDefault(); verify_account();">
                            <label for="email_code">Code:</label><br>
                            <input type="text" name="email_code" id="email_code" placeholder="Code" autocomplete="off" required />
                            <button id="verify-button" type="submit" href="/">Verify</button>
                        </form>

                    </div>

                    <div class="module">
                        <a style="font-size:0.75em;cursor:pointer;" onclick="new_code();">Send me a new code!</i>
                    </div>

                </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Robot or human?';
    clearResponse();
    showLoggedInMenu();
    document.getElementById('navbar').style.display = 'none';

}

function verify_account(){

    var email_code = document.getElementById("email_code").value;

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

                // store jwt to cookie
                set_cookie("poenskelisten", result.token, 7);
                frontPageRedirect();

            }

        } else {
            info("Verifying account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/verify/" + email_code);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}

function new_code(){

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

            }

        } else {
            info("Sending new code...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/verification");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}

function frontPageRedirect() {

    window.location = '/'

}