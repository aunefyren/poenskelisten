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
        admin = false;
        var user_id = 0;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Wishlists
                        </div>

                        <div class="text-body" style="text-align: center;">
                            These are your wishlists. You can add different wishlists to different groups, allowing others to see your wishes. At the bottom of this page you can create a new wishlist. Make sure to add it to a group afterward.
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
                                    <img class="icon-img color-invert" src="../assets/list.svg">
                                </div>
                                Expired wishlists
                            </div>
                            <div class="profile-icon clickable" onclick="toggle_expired_wishlists()" title="Expandable">
                                <img id="wishlist_expired_arrow" class="icon-img color-invert" src="../../assets/chevron-right.svg">
                            </div>
                            <div id="wishlists-box-expired" class="wishlists collapsed" style="display:none;">
                            </div>
                        </div>

                        <div id="wishlist-input" class="wishlist-input">
                            <form action="" onsubmit="event.preventDefault(); create_wishlist(` + user_id + `);">
                                
                                <label for="wishlist_name">Create a new wishlist:</label><br>

                                <input type="text" name="wishlist_name" id="wishlist_name" placeholder="Wishlist name" autocomplete="off" required />
                                
                                <input type="text" name="wishlist_description" id="wishlist_description" placeholder="Wishlist description" autocomplete="off" required />

                                <label for="wishlist_date">When does your wishlist expire?</label><br>
                                <input type="date" name="wishlist_date" id="wishlist_date" placeholder="Wishlist expiration" autocomplete="off" required />

                                <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="wishlist_claimable" name="wishlist_claimable" value="confirm" checked>
                                <label for="wishlist_claimable" style="margin-bottom: 1em;" class="clickable">Allow users to claim wishes.</label><br>
                                
                                <button id="register-button" type="submit" href="/">Create wishlist</button>

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
        
        get_wishlists(user_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_wishlists(user_id){

    console.log(user_id);

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
                place_wishlists(wishlists, user_id);

            }

        } else {
            info("Loading wishlists...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_wishlists(wishlists_array, user_id) {

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

        console.log("Wishlist ID: " + wishlists_array[i].ID)

        owner_id = wishlists_array[i].owner.ID

        html += '<div class="wishlist-wrapper">'

        html += '<div class="wishlist">'
        
        html += '<div class="wishlist-title clickable" onclick="location.href = \'./wishlists/' + wishlists_array[i].ID + '\'" title="Go to wishlist">'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/list.svg">'
        html += '</div>'
        html += wishlists_array[i].name
        html += '</div>'

        html += '<div class="profile" title="Wishlist owner">'
        html += '<div class="profile-name">'
        html += wishlists_array[i].owner.first_name + " " + wishlists_array[i].owner.last_name
        html += '</div>'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/user.svg">'
        html += '</div>'

        var members_string="["
        for(var j = 0; j < wishlists_array[i].members.length; j++) {
            if(j !== 0) {
                members_string += ','
            }
            members_string += wishlists_array[i].members[j].ID
            
            console.log(wishlists_array[i].ID + " " + wishlists_array[i].members[j].ID)
            console.log(wishlists_array[i].members)
        }
        members_string += ']'

        if(owner_id == user_id) {
            html += '<div class="profile-icon clickable" onclick="toggle_wishlist(' + user_id + ', ' + wishlists_array[i].ID + ', ' + owner_id + ', ' + members_string + ')" title="Expandable">'
            html += '<img id="wishlist_' + wishlists_array[i].ID + '_arrow" class="icon-img color-invert" src="../../assets/chevron-right.svg">'
            html += '</div>'
        }

        if(owner_id == user_id) {
            html += '<div class="profile-icon clickable" onclick="delete_wishlist(' + wishlists_array[i].ID + ', ' + user_id + ')" title="Delete wishlist">'
            html += '<img class="icon-img color-invert" src="../../assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'

        html += '<div class="group-members collapsed" id="wishlist_' + wishlists_array[i].ID + '_members">'
        for(var j = 0; j < wishlists_array[i].members.length; j++) {
            if(j == 0) {
                html += '<div class="text-body">Available in these groups:</div>'
            }

            html += '<div class="group-member hoverable-dark" title="Group">'

            html += '<div class="group-title">';

            html += '<div class="profile-icon">'
            html += '<img class="icon-img color-invert" src="../assets/users.svg">'
            html += '</div>'

            html += wishlists_array[i].members[j].name

            html += '</div>'

            if(owner_id == user_id) {
                html += '<div class="profile-icon clickable" onclick="remove_member(' + wishlists_array[i].ID + ',' + wishlists_array[i].members[j].ID + ', ' + user_id +')" title="Remove wishlist from group">'
                html += '<img class="icon-img color-invert" src="../../assets/x.svg">'
                html += '</div>'
            }
            html += '</div>'
        }

        if(owner_id == user_id) {
            html += '<form action="" onsubmit="event.preventDefault(); add_groups(' + wishlists_array[i].ID + ', ' + user_id + ');">';
            html += '<label for="wishlist-input-members-' + wishlists_array[i].ID + '">Select groups:</label><br>';
            html += '<select name="wishlist_members_' + wishlists_array[i].ID + '" id="wishlist-input-members-' + wishlists_array[i].ID + '" multiple>';
            html += '</select>';
            html += '<button id="register-button" type="submit" href="/">Add wishlist to groups</button>';
            html += '</form>';
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
}

function create_wishlist(user_id) {

    var wishlist_name = document.getElementById("wishlist_name").value;
    var wishlist_description = document.getElementById("wishlist_description").value;
    var wishlist_date = document.getElementById("wishlist_date").value;
    var wishlist_date_object = new Date(wishlist_date)
    var wishlist_date_string = wishlist_date_object.toISOString();
    var wishlist_claimable = document.getElementById("wishlist_claimable").checked;

    var form_obj = { 
                                    "name" : wishlist_name,
                                    "description" : wishlist_description,
                                    "date": wishlist_date_string,
                                    "claimable": wishlist_claimable
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
                place_wishlists(wishlists, user_id);
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

function delete_wishlist(wishlist_id, user_id) {

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
                place_wishlists(wishlists, user_id);
                
            }

        } else {
            info("Deleting wishlist...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/" + wishlist_id + "/delete");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function toggle_wishlist(user_id, wishlist_id, owner_id, member_array) {
    wishlist_members = document.getElementById("wishlist_" + wishlist_id + "_members");
    wishlist_members_arrow = document.getElementById("wishlist_" + wishlist_id + "_arrow");

    console.log(member_array);

    if(wishlist_members.classList.contains("collapsed")) {
        wishlist_members.classList.remove("collapsed")
        wishlist_members.classList.add("expanded")
        wishlist_members.style.display = "inline-block"
        wishlist_members_arrow.src = "../../assets/chevron-down.svg"

        if(user_id == owner_id) {
            get_groups(owner_id, wishlist_id, user_id, member_array)
        }
    } else {
        wishlist_members.classList.remove("expanded")
        wishlist_members.classList.add("collapsed")
        wishlist_members.style.display = "none"
        wishlist_members_arrow.src = "../../assets/chevron-right.svg"

        if(user_id == owner_id) {
            var select_list = document.getElementById("wishlist-input-members-" + wishlist_id)
            if(select_list.options.length > 0) {
                var options = [];
                for (var i = 0; i < select_list.options.length; i++) {
                    options.push(select_list.options[i]);
                }
                for (var i = 0; i < options.length; i++) {
                    select_list.remove(options[i]);
                }
            }
        }
    }
}

function toggle_expired_wishlists() {

    wishlist_expired = document.getElementById("wishlists-box-expired");
    wishlist_expired_arrow = document.getElementById("wishlist_expired_arrow");

    if(wishlist_expired.classList.contains("collapsed")) {
        wishlist_expired.classList.remove("collapsed")
        wishlist_expired.classList.add("expanded")
        wishlist_expired.style.display = "inline-block"
        wishlist_expired_arrow.src = "../../assets/chevron-down.svg"
    } else {
        wishlist_expired.classList.remove("expanded")
        wishlist_expired.classList.add("collapsed")
        wishlist_expired.style.display = "none"
        wishlist_expired_arrow.src = "../../assets/chevron-right.svg"
    }
}

function get_groups(owner_id, wishlist_id, user_id, member_array){

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
                groups = result.groups;
                console.log(groups);
                place_groups(groups, wishlist_id, owner_id, user_id, member_array);

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/group/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_groups(group_array, wishlist_id, owner_id, user_id, member_array) {
    var select_list = document.getElementById("wishlist-input-members-" + wishlist_id)

    console.log(group_array)

    for(var i = 0; i < group_array.length; i++) {

        var found = false;
        for(var j = 0; j < group_array.length; j++) {
            if(member_array[j] == group_array[i].ID) {
                found = true;
                break;
            }
        }
        if(found) {
            continue;
        }

        var option = document.createElement("option");
        option.value = group_array[i].ID
        option.text = group_array[i].name
        select_list.add(option, select_list[0]);
    }
}

function add_groups(wishlist_id, user_id) {

    var selected_members = [];
    var select_list = document.getElementById("wishlist-input-members-" + wishlist_id)

    for (var i=0; i < select_list.options.length; i++) {
        opt = select_list.options[i];
    
        if (opt.selected) {
            selected_members.push(Number(opt.value));
        }
    }

    var form_obj = { 
                                    "groups": selected_members
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

                wishlists = result.wishlists;

                console.log("Placing wishlists after member is added: ")
                place_wishlists(wishlists, user_id);
                
            }

        } else {
            info("Adding groups...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/" + wishlist_id + "/join");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function remove_member(wishlist_id, group_id, user_id) {

    if(!confirm("Are you sure you want to remove your wishlist from this group?")) {
        return;
    }

    var form_obj = { 
        "group_id" : group_id
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

                console.log("Placing groups after member is removed: ")
                place_wishlists(wishlists, user_id);
                
            }

        } else {
            info("Removing member...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/" + wishlist_id + "/remove");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}