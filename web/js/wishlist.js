function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
    } else {
        var login_data = false;
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
                    
                        <div class="title">
                            Wishes
                        </div>

                        <div class="text-body" style="text-align: center;">
                            These are wishes.
                        </div>

                        <br>
                        <br>

                        <div id="wishes-box" class="wishes">
                        </div>

                        <div id="wish-input" class="wish-input">
                            <form action="" onsubmit="event.preventDefault(); send_wish(` + wishlist_id + `,` + group_id + `,` + login_data.data.id + `);">
                                <label for="wish_name">Add a new wish:</label><br>
                                <input type="text" name="wish_name" id="wish_name" placeholder="Wish name" autocomplete="off" required />
                                <input type="text" name="wish_note" id="wish_note" placeholder="Wish note" autocomplete="off" />
                                <input type="text" name="wish_url" id="wish_url" placeholder="Wish URL" autocomplete="off" />
                                <button id="register-button" type="submit" href="/">Add wish</button>
                            </form>
                        </div>
      
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Lists...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        

        console.log(wishlist_id);
        console.log(group_id);

        get_wishes(wishlist_id, group_id, login_data.data.id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_wishes(wishlist_id, group_id, user_id){

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
                wishes = result.wishes;
                console.log(wishes);
                place_wishes(wishes, wishlist_id, group_id, user_id);

                if(result.owner_id == user_id) {
                    show_wish_input();
                }

            }

        } else {
            info("Loading wishes...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wish/get/" + wishlist_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_wishes(wishes_array, wishlist_id, group_id, user_id) {

    var html = ''

    for(var i = 0; i < wishes_array.length; i++) {

        html += '<div class="wish-wrapper">'

        html += '<div class="wish" id="wish_' + wishes_array[i].ID + '">'
        
        html += '<div class="wish-title">'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/gift.svg">'
        html += '</div>'
        html += wishes_array[i].name
        html += '</div>'

        html += '<div class="profile">'

        if(wishes_array[i].note !== "") {
            html += '<div class="profile-icon clickable" onclick="toggle_wish(' + wishes_array[i].ID + ')">'
            html += '<img id="wish_' + wishes_array[i].ID + '_arrow" class="icon-img color-invert" src="../../assets/chevron-right.svg">'
            html += '</div>'
        }

        if(wishes_array[i].url !== "") {
            html += '<div class="profile-icon clickable" onclick="window.open(\'' + wishes_array[i].url + '\', \'_blank\')">'
            html += '<img class="icon-img color-invert" src="../../assets/link.svg">'
            html += '</div>'
        }

        if(user_id == wishes_array[i].owner_id.ID) {
            html += '<div class="profile-icon clickable" onclick="delete_wish(' + wishes_array[i].ID + ", " + wishlist_id  + ", " + group_id  + ", " + user_id + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/trash-2.svg">'
            html += '</div>'
        }
        html += '</div>'

        html += '</div>'

        html += '<div class="wish-note collapsed" id="wish_' + wishes_array[i].ID + '_note">'
        html += wishes_array[i].note
        html += '</div>'

        html += '</div>'
    }

    if(wishes_array.length == 0) {
        info("Looks like this list is empty...");
    }

    wishlist_object = document.getElementById("wishes-box")
    wishlist_object.innerHTML = html
}

function toggle_wish(wishid) {
    wishnote = document.getElementById("wish_" + wishid + "_note");
    wishnotearrow = document.getElementById("wish_" + wishid + "_arrow");

    if(wishnote.classList.contains("collapsed")) {
        wishnote.classList.remove("collapsed")
        wishnote.classList.add("expanded")
        wishnote.style.display = "inline-block"
        wishnotearrow.src = "../../assets/chevron-down.svg"
    } else {
        wishnote.classList.remove("expanded")
        wishnote.classList.add("collapsed")
        wishnote.style.display = "none"
        wishnotearrow.src = "../../assets/chevron-right.svg"
    }
}

function show_wish_input() {
    wishinput = document.getElementById("wish-input");
    wishinput.style.display = "inline-block"
}

function send_wish(wishlist_id, group_id, user_id){

    var wish_name = document.getElementById("wish_name").value;
    var wish_note = document.getElementById("wish_note").value;
    var wish_url = document.getElementById("wish_url").value;

    var form_obj = { 
                                    "name" : wish_name,
                                    "note" : wish_note,
                                    "url": wish_url
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
                console.log(result);

                console.log("user id " + user_id);

                wishes = result.wishes;
                place_wishes(wishes, wishlist_id, group_id, user_id);
                clear_data();
                
               
            }

        } else {
            info("Saving wish...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wish/register/" + wishlist_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function clear_data() {
    document.getElementById("wish_name").value = "";
    document.getElementById("wish_note").value = "";
    document.getElementById("wish_url").value = "";
}

function delete_wish(wish_id, wishlist_id, group_id, user_id) {

    if(!confirm("Are you sure you want to delete this wish?")) {
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

                success(result.message);
                console.log(result);

                console.log("user id " + user_id);

                wishes = result.wishes;
                place_wishes(wishes, wishlist_id, group_id, user_id);
                clear_data();
                
               
            }

        } else {
            info("Deleting wish...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wish/" + wish_id + "/delete");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}