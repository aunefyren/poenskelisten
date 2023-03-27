function load_page(result) {

    if(result !== false) {
        
        try {

            var login_data = JSON.parse(result);
            
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var user_id = login_data.data.id;
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
                    
                        <div class="wishlist-info" id="wishlist-info-box">

                            <div id="wishlist-title" class="title">
                            </div>

                            <div class="text-body" id="wishlist-description">
                            </div>

                            <div class="text-body" id="wishlist-info">
                            </div>

                            <div class="bottom-right-button" id="edit-wishlist" style="display: none;" onclick="wishlist_edit(${user_id}, ${wishlist_id}, '{wishlist_expiration_date}');">
                                <img class="icon-img color-invert clickable" src="../assets/edit.svg">
                            </div>

                        </div>

                    </div>

                    <div class="module">

                        <div id="wishlists-title" class="title">
                            Wishes:
                        </div>

                        <div id="wishes-box" class="wishes">
                        </div>

                        <div id="wish-input" class="wish-input">
                            <form action="" onsubmit="event.preventDefault(); send_wish(` + wishlist_id + `,` + group_id + `,` + user_id + `);">
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

        get_wishlist(wishlist_id)
        get_wishes(wishlist_id, group_id, user_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_wishlist(wishlist_id){

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

                console.log(result);
                place_wishlist(result.wishlist);

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/get/" + wishlist_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_wishlist(wishlist_object) {

    document.getElementById("wishlist-title").innerHTML = wishlist_object.name
    document.getElementById("wishlist-description").innerHTML = wishlist_object.description
    document.getElementById("wishlist-info").innerHTML += "<br>By: " + wishlist_object.owner.first_name + " " + wishlist_object.owner.last_name

    try {
        var expiration = new Date(Date.parse(wishlist_object.date));
        expiration_string = expiration.toLocaleDateString();
        document.getElementById("wishlist-info").innerHTML += "<br>Expires: " + expiration_string
        
        var innerHTML = document.getElementById("wishlist-info-box").innerHTML
        document.getElementById("wishlist-info-box").innerHTML = innerHTML.replace('{wishlist_expiration_date}', wishlist_object.date)
    } catch(err) {
        console.log("Failed to parse datetime. Error: " + err)
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
                    show_owner_inputs();
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

        owner_id = wishes_array[i].owner_id.ID

        if(wishes_array[i].wishclaim.length > 0 && user_id != owner_id) {
            var transparent = " transparent"
        } else {
            var transparent = ""
        }

        html += '<div class="wish-wrapper ' + transparent + '">'

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

        if(user_id == owner_id) {
            html += '<div class="profile-icon clickable" onclick="delete_wish(' + wishes_array[i].ID + ", " + wishlist_id  + ", " + group_id  + ", " + user_id + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/trash-2.svg">'
            html += '</div>'
        } else if(wishes_array[i].wishclaim.length > 0) {
            for(var j = 0; j < wishes_array[i].wishclaim.length; j++) {
                if(user_id !== wishes_array[i].wishclaim[j].user.ID) {
                    html += '<div class="profile-icon" title="Claimed by ' + wishes_array[i].wishclaim[j].user.first_name + ' ' + wishes_array[i].wishclaim[j].user.last_name + '.">'
                    html += '<img class="icon-img color-invert" src="../../assets/lock.svg">'
                    html += '</div>'
                } else {
                    html += '<div class="profile-icon clickable" title="Claimed by you, click to unclaim.">'
                    html += '<img class="icon-img color-invert" src="../../assets/unlock.svg" onclick="unclaim_wish(' + wishes_array[i].ID + ", " + wishlist_id  + ", " + group_id  + ", " + user_id + ')")>'
                    html += '</div>'
                }
            }
        } else {
            html += '<div class="profile-icon clickable" title="Claim this gift." onclick="claim_wish(' + wishes_array[i].ID + ", " + wishlist_id  + ", " + group_id  + ", " + user_id + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/check.svg">'
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

function show_owner_inputs() {
    wishinput = document.getElementById("wish-input");
    wishinput.style.display = "inline-block"
    wishlistedit = document.getElementById("edit-wishlist");
    wishlistedit.style.display = "flex"
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

function claim_wish(wish_id, wishlist_id, group_id, user_id) {

    if(!confirm("Are you sure you want to claim this wish? Other users will not be able to gift the recipient this wish.")) {
        return;
    }

    var form_obj = { 
        "wishlist_id" : wishlist_id
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
            info("Claiming wish...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wish/" + wish_id + "/claim");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function unclaim_wish(wish_id, wishlist_id, group_id, user_id) {

    if(!confirm("Are you sure you want to unclaim this wish?")) {
        return;
    }

    var form_obj = { 
        "wishlist_id" : wishlist_id
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
            info("Un-claiming wish...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wish/" + wish_id + "/unclaim");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function wishlist_edit(user_id, wishlist_id, wishlist_expiration_date) {

    var wishlist_title = document.getElementById("wishlist-title").innerHTML;
    var wishlist_description = document.getElementById("wishlist-description").innerHTML;
    var wishlist_expiration = getDateString(wishlist_expiration_date)

    var html = '';

    html += `
        <form action="" onsubmit="event.preventDefault(); update_wishlist(${wishlist_id}, ` + user_id + `);">
                                
            <label for="wishlist_name">Edit wishlist:</label><br>
            <input type="text" name="wishlist_name" id="wishlist_name" placeholder="Wishlist name" value="${wishlist_title}" autocomplete="off" required />
            
            <input type="text" name="wishlist_description" id="wishlist_description" placeholder="Wishlist description" value="${wishlist_description}" autocomplete="off" required />

            <label for="wishlist_date">When does the wishlist expire?</label><br>
            <input type="date" name="wishlist_date" id="wishlist_date" placeholder="Wishlist expiration" value="${wishlist_expiration}" autocomplete="off" required />
            
            <button id="register-button" type="submit" href="/">Save wishlist</button>

        </form>
    `;

    document.getElementById("wishlist-info-box").innerHTML = html;

}

function update_wishlist(wishlist_id, user_id) {

    if(!confirm("Are you sure you want to update this wishlist?")) {
        return;
    }

    var wishlist_name = document.getElementById("wishlist_name").value;
    var wishlist_description = document.getElementById("wishlist_description").value;
    var wishlist_date = document.getElementById("wishlist_date").value;
    var wishlist_date_object = new Date(wishlist_date)
    var wishlist_date_string = wishlist_date_object.toISOString();

    var form_obj = { 
        "name" : wishlist_name,
        "description" : wishlist_description,
        "date": wishlist_date_string
    };

    var form_data = JSON.stringify(form_obj);

    console.log(form_data)

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
                reset_wishlist_info_box(user_id, wishlist_id);
                place_wishlist(result.wishlist);

            }

        } else {
            info("Updating wishlist...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/" + wishlist_id + "/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function reset_wishlist_info_box(user_id, wishlist_id) {
    var html = `
    <div id="wishlist-title" class="title">
    </div>

    <div class="text-body" id="wishlist-description">
    </div>

    <div class="text-body" id="wishlist-info">
    </div>

    <div class="bottom-right-button" id="edit-wishlist" style="display: none;" onclick="wishlist_edit(${user_id}, ${wishlist_id}, '{wishlist_expiration_date}');">
        <img class="icon-img color-invert clickable" src="../assets/edit.svg">
    </div>
    `;

    document.getElementById("wishlist-info-box").innerHTML = html;
}