function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
    } else {
        var login_data = {}
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
                            <input type="email" name="email" id="email" placeholder="Email" value="` + login_data.data.email + `" required/>

                            <label id="form-input-icon" for="first_name"></label>
                            <input type="text" name="first_name" id="first_name" placeholder="First name" value="` + login_data.data.first_name + `" required disabled />

                            <label id="form-input-icon" for="last_name"></label>
                            <input type="text" name="last_name" id="last_name" placeholder="Last name" value="` + login_data.data.last_name + `" disabled required/>

                            <input onclick="change_password_toggle();" style="margin-top: 2em;" type="checkbox" id="password-toggle" name="confirm" value="confirm" >
                            <label for="confirm">Change my password.</label><br>

                            <div id="change-password-box" style="display:none; transition: 2s;">

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

    alert("Not finished :(")

}