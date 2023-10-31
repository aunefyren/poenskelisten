function load_page(result) {

    if(result !== false) {

        try {
            
            var login_data = JSON.parse(result);

            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            user_id = login_data.data.id;
            admin = login_data.data.admin;
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            group_id = 0;
            user_id = 0;
            admin = false;
        }

        showAdminMenu(admin)

    } else {
        var login_data = false;
        group_id = 0;
        user_id = 0
        var email = ""
        var first_name = ""
        var last_name = ""
    }

    try {
        string_index = document.URL.lastIndexOf('/');
        group_id = document.URL.substring(string_index+1);
        console.log(group_id);
    } catch {
        group_id = 0;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">

                        <div class="group-info" id="group-info-box">

                            <div id="group-title" class="title">
                            </div>

                            <div class="text-body" id="group-description">
                            </div>

                            <div class="text-body" id="group-info">
                            </div>

                            <div class="bottom-right-button" id="edit-group" style="display: none;">
                                <img class="icon-img color-invert clickable" src="/assets/edit.svg" onclick="group_edit(${user_id}, ${group_id});" title="Edit group">
                            </div>

                        </div>

                    </div>

                    <div class="module">

                        <div id="wishlists-title" class="title">
                            Wishlists:
                        </div>

                        <div id="wishlists-box" class="wishlists">
                        </div>

                        <div id="wishlists-box-expired-wrapper" class="wishlist-wrapper wishlist-expired" style="display: none;">
                            <div class="wishlist-title">
                                <div class="profile-icon">
                                    <img class="icon-img color-invert" src="/assets/list.svg">
                                </div>
                                Expired wishlists
                            </div>
                            <div class="profile-icon clickable" onclick="toggle_expired_wishlists()" title="Expandable">
                                <img id="wishlist_expired_arrow" class="icon-img color-invert" src="/assets/chevron-right.svg">
                            </div>
                            <div id="wishlists-box-expired" class="wishlists collapsed" style="display:none;">
                            </div>
                        </div>

                        <div id="wishlist-input" class="wishlist-input">
                            <form action="" onsubmit="event.preventDefault(); create_wishlist(` + group_id + `, ` + user_id + `);">
                                
                                <label for="wishlist_name">Create a new wishlist in this group:</label><br>

                                <input type="text" name="wishlist_name" id="wishlist_name" placeholder="Wishlist name" autocomplete="off" required />
                                
                                <input type="text" name="wishlist_description" id="wishlist_description" placeholder="Wishlist description" autocomplete="off" required />

                                <label for="wishlist_date">When does your wishlist expire?</label><br>
                                <input type="date" name="wishlist_date" id="wishlist_date" placeholder="Wishlist expiration" autocomplete="off" required />
                                
                                <button id="register-button" type="submit" href="/">Create wishlist in this group</button>

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
        
        get_group(group_id);
        get_wishlists(group_id, login_data.data.id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_group(group_id){

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
                place_group(result.group);

                if(result.group.owner.ID == user_id) {
                    show_owner_inputs();
                }

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/group/get/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_group(group_object) {

    document.getElementById("group-title").innerHTML = group_object.name
    document.getElementById("group-description").innerHTML = group_object.description
    document.getElementById("group-info").innerHTML += "<br>Owner: " + group_object.owner.first_name + " " + group_object.owner.last_name

}

function get_wishlists(group_id, user_id){

    console.log(group_id + ", " + user_id);

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
                wishlists = result.wishlists;
                console.log(result);
                place_wishlists(wishlists, group_id, user_id);

            }

        } else {
            info("Loading wishlists...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/get/group/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_wishlists(wishlists_array, group_id, user_id) {

    var html_regular = ''
    var html_expired = ''
    var html = ''
    var wishlists_array_length = wishlists_array.length
    var wishlists_expired_length = 0

    for(var i = 0; i < wishlists_array.length; i++) {

        var expired = false;
        html = ''

        try {
            var expiration = new Date(Date.parse(wishlists_array[i].date));
            var now = new Date
            console.log("Times: " + expiration.toISOString() + " & " + now.toISOString())
            if(expiration.getTime() < now.getTime()) {
                console.log("Expired wishlist.")
                expired = true;
                wishlists_array_length -= 1
                wishlists_expired_length += 1
            } else {
                console.log("Not skipping wishlist.")
            }
        } catch(err) {
            console.log("Failed to parse datetime. Error: " + err)
        }

        html += '<div class="wishlist-wrapper">'

        html += '<div class="wishlist hoverable-light">'
        
        html += '<div class="wishlist-title clickable" onclick="location.href = \'../wishlists/'+ wishlists_array[i].ID + '\'" title="Go to wishlist">'
        html += `<div class="profile-icon">`
        html += '<img class="icon-img color-invert" src="/assets/list.svg">'
        html += '</div>'
        html += wishlists_array[i].name
        html += '</div>'

        html += '<div class="profile" title="Wishlist owner">'
        html += `<div class="profile-name">`
        html += wishlists_array[i].owner.first_name + " " + wishlists_array[i].owner.last_name
        html += '</div>'
        html += `<div class="profile-icon" id="group_owner_image_${wishlists_array[i].owner.ID}_${wishlists_array[i].ID}">`
        html += '<img class="icon-img color-invert" src="/assets/user.svg">'
        html += '</div>'

        if(wishlists_array[i].owner.ID == user_id) {
            html += '<div class="profile-icon clickable" onclick="delete_wishlist(' + wishlists_array[i].ID + ', ' + group_id + ', ' + user_id + ')" title="Delete wishlist">'
            html += '<img class="icon-img color-invert" src="/assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'

        html += '</div>'

        if(expired) {
            html_expired += html;
        } else {
            html_regular += html;
        }

    }

    if(wishlists_array_length < 1) {
        info("Looks like this list is empty...");
    }

    if(wishlists_expired_length > 0) {
        document.getElementById("wishlists-box-expired-wrapper").style.display = "flex"
    } else {
        document.getElementById("wishlists-box-expired-wrapper").style.display = "none"
    }

    wishlist_object = document.getElementById("wishlists-box")
    wishlist_object.innerHTML = html_regular

    wishlist_object_expired = document.getElementById("wishlists-box-expired")
    wishlist_object_expired.innerHTML = html_expired

    for(var i = 0; i < wishlists_array.length; i++) {
        GetProfileImage(wishlists_array[i].owner.ID, `group_owner_image_${wishlists_array[i].owner.ID}_${wishlists_array[i].ID}`)
    }
}

function create_wishlist(group_id, user_id) {

    var wishlist_name = document.getElementById("wishlist_name").value;
    var wishlist_description = document.getElementById("wishlist_description").value;
    var wishlist_date = document.getElementById("wishlist_date").value;
    var wishlist_date_object = new Date(wishlist_date)
    var wishlist_date_string = wishlist_date_object.toISOString();

    try {
        group_id_int = parseInt(group_id);
    } catch {
        alert("Failed. Invalid group")
        return
    }

    var form_obj = { 
                                    "name" : wishlist_name,
                                    "description" : wishlist_description,
                                    "date": wishlist_date_string,
                                    "group": group_id_int
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
                console.log(result);

                console.log("User ID: " + user_id);

                wishlists = result.wishlists;
                place_wishlists(wishlists, group_id, user_id);
                clear_data();
                
            }

        } else {
            info("Saving wishlist...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function clear_data() {
    document.getElementById("wishlist_name").value = "";
    document.getElementById("wishlist_description").value = "";
    document.getElementById("wishlist_date").value = "";
}

function delete_wishlist(wishlist_id, group_id, user_id) {

    if(!confirm("Are you sure you want to delete this wishlist?")) {
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

                console.log("User ID: " + user_id);

                wishlists = result.wishlists;
                place_wishlists(wishlists, group_id, user_id);
                
            }

        } else {
            info("Deleting group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/" + wishlist_id + "/delete");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function toggle_expired_wishlists() {

    wishlist_expired = document.getElementById("wishlists-box-expired");
    wishlist_expired_arrow = document.getElementById("wishlist_expired_arrow");

    if(wishlist_expired.classList.contains("collapsed")) {
        wishlist_expired.classList.remove("collapsed")
        wishlist_expired.classList.add("expanded")
        wishlist_expired.style.display = "inline-block"
        wishlist_expired_arrow.src = "/assets/chevron-down.svg"
    } else {
        wishlist_expired.classList.remove("expanded")
        wishlist_expired.classList.add("collapsed")
        wishlist_expired.style.display = "none"
        wishlist_expired_arrow.src = "/assets/chevron-right.svg"
    }
}

function reset_group_info_box(user_id, group_id) {
    var html = `
    <div id="group-title" class="title">
    </div>

    <div class="text-body" id="group-description">
    </div>

    <div class="text-body" id="group-info">
    </div>

    <div class="bottom-right-button" id="edit-group" style="display: none;">
        <img class="icon-img color-invert clickable" src="/assets/edit.svg" onclick="group_edit(${user_id}, ${group_id});">
    </div>
    `;

    document.getElementById("group-info-box").innerHTML = html;
}

function show_owner_inputs() {
    wishlistedit = document.getElementById("edit-group");
    wishlistedit.style.display = "flex"
}

function group_edit(user_id, group_id) {

    var group_title = document.getElementById("group-title").innerHTML;
    var group_description = document.getElementById("group-description").innerHTML;

    var html = '';

    html += `
        <div class="bottom-right-button" id="edit-group" style="" onclick="cancel_edit_group(${group_id}, ${user_id});" title="Cancel edit">
            <img class="icon-img color-invert clickable" style="" src="/assets/x.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); update_group(${group_id}, ` + user_id + `);">
                                
            <label for="group_name">Edit group:</label><br>
            <input type="text" name="group_name" id="group_name" placeholder="Group name" value="${group_title}" autocomplete="off" required />
            
            <input type="text" name="group_description" id="group_description" placeholder="Group description" value="${group_description}" autocomplete="off" required />

            <button id="register-button" type="submit" href="/">Save group</button>

        </form>
    `;

    document.getElementById("group-info-box").innerHTML = html;

}

function update_group(group_id, user_id) {

    if(!confirm("Are you sure you want to update this group?")) {
        return;
    }

    var group_title = document.getElementById("group_name").value;
    var group_description = document.getElementById("group_description").value;

    var form_obj = { 
        "name" : group_title,
        "description" : group_description
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
                reset_group_info_box(user_id, group_id);
                place_group(result.group);
                show_owner_inputs();

            }

        } else {
            info("Updating group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/group/" + group_id + "/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function cancel_edit_group(group_id, user_id){

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

                reset_group_info_box(user_id, group_id);
                place_group(result.group);
                show_owner_inputs();

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/group/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
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

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/user/get/" + userID + "/image?thumbnail=true");
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

}