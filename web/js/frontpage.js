function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        if(login_data.error === "You must verify your account.") {
            load_verify_account();
            return;
        }

    } else {
        var login_data = false;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Pønskelisten
                        </div>

                        <div class="text-body" style="text-align: center;">
                            Make a wish.

                            <br>
                            <br>

                            Welcome to the front page. Not much to see here currently. Use to navigation bar and head to 'Wishlists' to manage your wishlists. Head to 'Groups' to manage and view wishlists in groups.
                        </div>

                    </div>

                    <div class="module">

                        <div id="divider-1" class="divider" style="display: none;">
                            <hr></hr>
                        </div>


                        <div id="news-title" class="title" style="display: none;">
                            News:
                        </div>

                        <div id="divider-2" class="divider" style="display: none;">
                            <hr></hr>
                        </div>

                        <div id="news-box" class="news">
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        get_news(login_data.admin);
    } else {
        showLoggedOutMenu();
    }
}

function get_news(admin){

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

                clearResponse();
                news = result.news;

                console.log(news);

                console.log("Placing intial news: ")
                place_news(news, admin);

            }

        } else {
            info("Loading news...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/news/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_news(news_array, admin) {

    if(news_array.length == 0) {
        return;
    } else {
        document.getElementById("news-title").style.display = "inline-block"
        document.getElementById("divider-1").style.display = "inline-block"
        document.getElementById("divider-2").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < news_array.length; i++) {

        // parse date object
        try {
            var date = new Date(Date.parse(news_array[i].date));
            var date_string = date.toLocaleDateString();
        } catch {
            var date_string = "Error"
        }

        html += '<div class="news-post">'
        
        html += '<div id="news-title" class="title">';
        html += news_array[i].title
        html += '</div>';

        html += '<div id="news-body" class="text-body">';
        html += news_array[i].body
        html += '</div>';

        html += '<div id="news-body" class="text-date">';
        html += date_string
        html += '</div>';

        html += '</div>'

        html += '<div class="divider">'
        html += '<hr></hr>'
        html += '</div>'

    }

    news_object = document.getElementById("news-box")
    news_object.innerHTML = html

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

                        <form action="" onsubmit="event.preventDefault(); verify_account();">
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
                location.reload();

            }

        } else {
            info("Verifying account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/user/verify/" + email_code);
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
    xhttp.open("post", api_url + "open/user/verification");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}