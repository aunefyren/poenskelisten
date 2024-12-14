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
                <!-- The Modal -->
                <div id="myModal" class="modal closed">
                    <span class="close selectable" style="padding: 0 0.25em;" onclick="toggleModal()">&times;</span>
                    <div class="modalContent" id="modalContent">
                    </div>
                    <div id="caption"></div>
                </div>

                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Wishlists
                        </div>

                        <div class="text-body" style="text-align: center;">
                            These are wishlists where you are an owner or collaborator. You can add different wishlists to different groups, allowing others to see your wishes. At the bottom of this page you can create a new wishlist. Make sure to add it to a group afterward.
                        </div>

                    </div>

                    <div class="module">

                        <div id="wishlists-title" class="title">
                            Wishlists:
                        </div>

                        <div id="wishlists-box" class="wishlists">
                            <div class="loading-icon-wrapper" id="loading-icon-wrapper">
                                <img class="loading-icon" src="/assets/loading.svg">
                            </div>
                        </div>

                        <div id="wishlists-box-expired-wrapper" class="wishlist-wrapper wishlist-expired" style="display: none;">
                            <div class="wishlist-title" style="margin: 0.5em 0 !important;">
                                <div class="profile-icon">
                                    <img class="icon-img " src="/assets/list.svg">
                                </div>
                                Expired wishlists
                            </div>
                            <div class="profile-icon clickable" onclick="toggle_expired_wishlists()" title="Expandable">
                                <img id="wishlist_expired_arrow" class="icon-img " src="/assets/chevron-right.svg">
                            </div>
                            <div id="wishlists-box-expired" class="wishlists collapsed" style="display:none;">
                            </div>
                        </div>

                        <div id="wishlist-input" class="wishlist-input">
                            <button id="register-button" onClick="createNewWishlist(false, '${user_id}');" type="" href="/">Create new wishlist</button>
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
                placeWishlists(wishlists, user_id);

            }

        } else {
            info("Loading wishlists...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWishlists(wishlists_array, user_id) {

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
            if(expiration.getTime() < now.getTime() && wishlists_array[i].expires) {
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

        var wishUpdatedAt = new Date(Date.parse(wishlists_array[i].wish_updated_at));
        var wishUpdatedAtString = GetDateString(wishUpdatedAt)

        owner_id = wishlists_array[i].owner.id

        html += `<div class="wishlist-wrapper" id="wishlistWrapper-${wishlists_array[i].id}">`

        html += '<div class="wishlist">'

        if(wishlists_array[i].wish_updated_at) {
            html += `
                <div class="unselectable wish-updatedat" title="Updated at">
                    <div class="wish-updatedat-text">Updated at:</div>
                    <div class="wish-updatedat-date" id="wishlistUpdatedAt-${wishlists_array[i].id}">
                        ${wishUpdatedAtString}
                    </div>
                </div>
            `;
        }
        
        html += `<div class="wishlist-title clickable underline" onclick="location.href = '/wishlists/${wishlists_array[i].id}'" title="Go to wishlist">`;
        html += '<div class="profile-icon">'
        html += '<img class="icon-img" src="/assets/list.svg">'
        html += `</div><b id="wishlistName-${wishlists_array[i].id}">`
        html += wishlists_array[i].name
        html += '</b></div>'

        html += '<div class="profile" title="Wishlist owner">'

        html += '<div class="profile-wrapper">'

        html += '<div class="profile-name">'
        html += wishlists_array[i].owner.first_name + " " + wishlists_array[i].owner.last_name
        html += '</div>'
        

        html += `<div class="profile-icon icon-border icon-background" id="wishlist_owner_image_${owner_id}_${wishlists_array[i].id}">`
        html += `<img class="icon-img " src="/assets/user.svg" id="wishlist_owner_image_img_${owner_id}_${wishlists_array[i].id}">`
        html += '</div>'

        html += '</div>'

        html += '<div class="icons-wrapper">'

        html += `
            <div class="profile-icon clickable" onclick="showGroupsInWishlist('${wishlists_array[i].id}', '${user_id}')" title="Wishlist groups">
                <img class="icon-img " src="/assets/users.svg">
            </div>
        `;

        html += `
            <div class="profile-icon clickable" onclick="showWishlistCollaboratorsInWishlist('${wishlists_array[i].id}', '${user_id}')" title="Wishlist collaborators">
                <img class="icon-img " src="/assets/smile.svg">
            </div>
        `;

        if(owner_id == user_id) {
            html += `
                <div class="profile-icon clickable" onclick="editWishlist('${user_id}', '${wishlists_array[i].id}')" title="Edit wishlist">
                    <img class="icon-img " src="/assets/edit.svg">
                </div>
            `;

            html += `<div class="profile-icon clickable" onclick="deleteWishlist('${wishlists_array[i].id}', '${user_id}')" title="Delete wishlist">`
            html += '<img class="icon-img " src="/assets/trash-2.svg">'
            html += '</div>' 
        }

        html += '</div>'
        html += '</div>'
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
        info("Looks like this list is empty... Ready to create your first wishlist?");

        try {
            document.getElementById("loading-icon-wrapper").style.display = "none"
        } catch(e) {
            console.log("Error: " + e)
        }
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
        GetProfileImage(wishlists_array[i].owner.id, `wishlist_owner_image_${wishlists_array[i].owner.id}_${wishlists_array[i].id}`)
    }
    for(var i = 0; i < wishlists_array.length; i++) {
        for(var j = 0; j < wishlists_array[i].collaborators.length; j++) {
            GetProfileImage(wishlists_array[i].collaborators[j].user.id, `wishlist_${wishlists_array[i].id}_collaborator_${wishlists_array[i].collaborators[j].user.id}`)
        }
    }
}

function toggle_wishlist(user_id, wishlist_id, owner_id, member_array, collaboratorArray) {
    wishlist_members = document.getElementById("wishlist_" + wishlist_id + "_members");
    wishlist_members_arrow = document.getElementById("wishlist_" + wishlist_id + "_arrow");

    console.log(member_array);

    if(wishlist_members.classList.contains("collapsed")) {
        wishlist_members.classList.remove("collapsed")
        wishlist_members.classList.add("expanded")
        wishlist_members.style.display = "inline-block"
        wishlist_members_arrow.src = "/assets/chevron-down.svg"

        if(user_id == owner_id) {
            get_groups(owner_id, wishlist_id, user_id, member_array)
        }
        if(user_id == owner_id) {
            getCollaborators(owner_id, wishlist_id, user_id, collaboratorArray)
        }
    } else {
        wishlist_members.classList.remove("expanded")
        wishlist_members.classList.add("collapsed")
        wishlist_members.style.display = "none"
        wishlist_members_arrow.src = "/assets/chevron-right.svg"

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
        wishlist_expired_arrow.src = "/assets/chevron-down.svg"
    } else {
        wishlist_expired.classList.remove("expanded")
        wishlist_expired.classList.add("collapsed")
        wishlist_expired.style.display = "none"
        wishlist_expired_arrow.src = "/assets/chevron-right.svg"
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
    xhttp.open("get", api_url + "auth/groups");
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
            console.log(member_array[j])
            if(member_array[j] == group_array[i].id) {
                found = true;
                break;
            }
        }
        if(found) {
            continue;
        }

        var option = document.createElement("option");
        option.value = group_array[i].id
        option.text = group_array[i].name
        select_list.add(option, select_list[0]);
    }
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
    xhttp.open("get", api_url + "auth/users/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;
}

function PlaceProfileImage(imageBase64, divID) {

    try {
        var image = document.getElementById(divID)
        image.style.backgroundSize = "cover"
        image.innerHTML = ""
        image.style.backgroundImage = `url('${imageBase64}')`
        image.style.backgroundPosition = "center center"
    } catch(e) {
        console.log("Failed to place image at div ID: " + divID)
        console.log("Error: " + e)
    }
}

function placeWishlist(wishlistOject, publicURL) {
    document.getElementById("wishlistName-" + wishlistOject.id).innerHTML = wishlistOject.name
    var wishUpdatedAt = new Date(Date.parse(wishlistOject.wish_updated_at));
    var wishUpdatedAtString = GetDateString(wishUpdatedAt)
    document.getElementById(`wishlistUpdatedAt-${wishlistOject.id}`).innerHTML = wishUpdatedAtString

    var wishlist = document.getElementById(`wishlistWrapper-${wishlistOject.id}`)
    var wishlistHTML = wishlist.outerHTML
    wishlist.remove()
    
    if(wishlistOject.expires && wishlistOject.date) {
        var expiration = new Date(Date.parse(wishlistOject.date));
        var now = new Date

        if(expiration.getTime() < now.getTime()) {
            var wishlists = document.getElementById(`wishlists-box-expired`)
            wishlists.innerHTML = wishlistHTML + wishlists.innerHTML
        } else {
            var wishlists = document.getElementById(`wishlists-box`)
            wishlists.innerHTML = wishlistHTML + wishlists.innerHTML
        }
    } else {
        var wishlists = document.getElementById(`wishlists-box`)
        wishlists.innerHTML = wishlistHTML + wishlists.innerHTML
    }
}

function removeWishlist(wishlistID, userID) {
    document.getElementById(`wishlistWrapper-${wishlistID}`).remove();
}