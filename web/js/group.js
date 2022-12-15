function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
        user_id = login_data.data.id
    } else {
        var login_data = false;
        group_id = 0;
        user_id = 0
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

                        <div class="group-info">

                            <div id="group-title" class="title">
                            </div>

                            <div class="text-body" id="group-description">
                            </div>

                            <div class="text-body" id="group-info">
                            </div>

                        </div>

                    </div>

                    <div class="module">

                        <div id="wishlists-title" class="title">
                            Wishlists:
                        </div>

                        <div id="wishlists-box" class="wishlists">
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

    var html = ''

    for(var i = 0; i < wishlists_array.length; i++) {

        html += '<div class="wishlist-wrapper">'

        html += '<div class="wishlist">'
        
        html += '<div class="wishlist-title clickable" onclick="location.href = \'../wishlists/'+ wishlists_array[i].ID + '\'">'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/list.svg">'
        html += '</div>'
        html += wishlists_array[i].name
        html += '</div>'

        html += '<div class="profile">'
        html += '<div class="profile-name">'
        html += wishlists_array[i].owner.first_name + " " + wishlists_array[i].owner.last_name
        html += '</div>'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/user.svg">'
        html += '</div>'

        if(wishlists_array[i].owner.ID == user_id) {
            html += '<div class="profile-icon clickable" onclick="delete_wishlist(' + wishlists_array[i].ID + ', ' + group_id + ', ' + user_id + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'

        html += '</div>'
    }

    if(wishlists_array.length == 0) {
        info("Looks like this group is empty...");
    }

    wishlist_object = document.getElementById("wishlists-box")
    wishlist_object.innerHTML = html
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