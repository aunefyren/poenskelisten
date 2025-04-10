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
                <!-- The Modal -->
                <div id="myModal" class="modal closed">
                    <span class="close clickable" style="padding: 0 0.25em;" onclick="toggleModal()">&times;</span>
                    <div class="modalContent" id="modalContent">
                    </div>
                    <div id="caption"></div>
                </div>

                <div class="" id="front-page">
                    
                    <div class="module">

                        <div class="group-info" id="group-info-box">

                            <div class="loading-icon-wrapper" id="loading-icon-wrapper-group">
                                <img class="loading-icon" src="/assets/loading.svg">
                            </div>

                            <div id="group-title" class="title">
                            </div>

                            <div class="text-body" id="group-description">
                            </div>

                            <div class="text-body" id="group-info">
                            </div>

                            <div class="bottom-right-button" id="" style="">
                                <img class="icon-img clickable" src="/assets/user.svg" onclick="groupMembers('${group_id}', '${user_id}')" title="Configure members" style="margin: 0.25em;">
                                <img class="icon-img clickable" src="/assets/list.svg" onclick="showWishlistsInGroup('${group_id}', '${user_id}')" title="Configure wishlists" style="margin: 0.25em;">
                                <img class="icon-img clickable" id="edit-group" src="/assets/edit.svg" onclick="editGroup('${user_id}', '${group_id}');" title="Edit group" style="margin: 0.25em; display: none;">
                                <img class="icon-img clickable" id="delete-group" src="/assets/trash-2.svg" onclick="deleteGroup('${group_id}');" title="Delete group" style="margin: 0.25em; display: none;"></img>
                            </div>

                        </div>

                    </div>

                    <div class="module">

                        <div id="wishlists-title" class="title-two">
                            Wishlists:
                        </div>

                        <div class="profile-icon top-left-button" style="">
                            <input class="clickable" onclick="toggleExpiredWishlists()" title="Toggle showing expired wishlists" style="" type="checkbox" id="toggle_expired_wishlists" name="toggle_expired_wishlists" value="confirm">
                            <label for="toggle_expired_wishlists" style="" class="clickable">Show expired wishlists</label><br>
                        </div>

                        <div id="wishlists-box" class="wishlists">
                            <div class="loading-icon-wrapper" id="loading-icon-wrapper">
                                <img class="loading-icon" src="/assets/loading.svg">
                            </div>
                        </div>

                        <div id="wishlist-input" class="wishlist-input">
                            <button id="register-button" onClick="createNewWishlist('${group_id}', '${user_id}');" type="" href="/">Create new wishlist</button>
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
                placeGroup(result.group);

                if(result.group.owner.id == user_id) {
                    showOwnerInputs();
                    groupOwnerID = result.group.owner.id;
                }

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/groups/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeGroup(group_object, showOwnerInput) {
    try {
        document.getElementById("loading-icon-wrapper-group").style.display = "none"
    } catch(e) {
        console.log("Error: " + e)
    }

    document.getElementById("group-title").innerHTML = group_object.name
    document.getElementById("group-description").innerHTML = group_object.description
    document.getElementById("group-info").innerHTML = "<br>Owner: " + group_object.owner.first_name + " " + group_object.owner.last_name
    
    if(showOwnerInput) {
        console.log("Showing")
        showOwnerInputs();
    }
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
                placeWishlists(wishlists, user_id, group_id, false);

            }

        } else {
            info("Loading wishlists...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists?group=" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWishlists(wishlists_array, user_id, group_id, showExpired) {

    var html_regular = ''
    var html = ''
    var wishlists_array_length = wishlists_array.length
    var wishlists_expired_length = 0

    for(var i = 0; i < wishlists_array.length; i++) {

        var expired = false;
        var expiredHTMLClass = ''
        var wishlistDisplayStyle = 'flex'
        html = ''

        try {
            var expiration = new Date(Date.parse(wishlists_array[i].date));
            var now = new Date
            if(expiration.getTime() < now.getTime() && wishlists_array[i].expires) {
                expired = true;
                expiredHTMLClass = 'wishlist-expired'

                wishlists_array_length -= 1
                wishlists_expired_length += 1
            }
        } catch(err) {
            console.log("Failed to parse datetime. Error: " + err)
        }

        if(expired && !showExpired) {
            wishlistDisplayStyle = 'none';
        }

        var wishUpdatedAt = new Date(Date.parse(wishlists_array[i].wish_updated_at));
        var wishUpdatedAtString = GetDateString(wishUpdatedAt)

        html += `<div class="wishlist-wrapper ${expiredHTMLClass}" style="display: ${wishlistDisplayStyle}" id="wishlistWrapper-${wishlists_array[i].id}">`

        html += '<div class="wishlist hoverable-light">'

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
        html += `<div class="profile-icon">`
        html += '<img class="icon-img " src="/assets/list.svg">'
        html += `</div><b id="wishlistName-${wishlists_array[i].id}">`
        html += wishlists_array[i].name
        html += '</b></div>'

        html += '<div class="profile" title="Wishlist owner">'

        html += '<div class="profile-wrapper">'

        html += `<div class="profile-name">`
        html += wishlists_array[i].owner.first_name + " " + wishlists_array[i].owner.last_name
        html += '</div>'
        html += `<div class="profile-icon icon-border icon-background" id="group_owner_image_${wishlists_array[i].owner.id}_${wishlists_array[i].id}">`
        html += '<img class="icon-img " src="/assets/user.svg">'
        html += '</div>'

        html += '</div>'

        html += '<div class="icons-wrapper">'

        html += `
            <div class="profile-icon clickable" onclick="showWishlistCollaboratorsInWishlist('${wishlists_array[i].id}', '${user_id}')" title="Wishlist collaborators">
                <img class="icon-img " src="/assets/smile.svg">
            </div>
        `;

        if(wishlists_array[i].owner.id == user_id) {
            html += `
                <div class="profile-icon clickable" onclick="showGroupsInWishlist('${wishlists_array[i].id}', '${user_id}', '${group_id}')" title="Wishlist groups">
                    <img class="icon-img " src="/assets/users.svg">
                </div>
            `;
        }

        if(wishlists_array[i].owner.id == user_id) {
            html += `
                <div class="profile-icon clickable" onclick="editWishlist('${user_id}', '${wishlists_array[i].id}')" title="Edit wishlist">
                    <img class="icon-img " src="/assets/edit.svg">
                </div>
            `;

            html += `<div class="profile-icon clickable" onclick="deleteWishlist('${wishlists_array[i].id}', '${user_id}')" title="Delete wishlist">`;
            html += '<img class="icon-img " src="/assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'
        html += '</div>'
        html += '</div>'
        html += '</div>'

        html_regular += html;
    }

    if(wishlists_array_length < 1) {
        info("Looks like this list is empty... Someone needs to add their wishlist to this group!");

        try {
            document.getElementById("loading-icon-wrapper").style.display = "none"
        } catch(e) {
            console.log("Error: " + e)
        }
    }

    wishlist_object = document.getElementById("wishlists-box")
    wishlist_object.innerHTML = html_regular

    for(var i = 0; i < wishlists_array.length; i++) {
        GetProfileImage(wishlists_array[i].owner.id, `group_owner_image_${wishlists_array[i].owner.id}_${wishlists_array[i].id}`)
    }
}

function showOwnerInputs() {
    groupEdit = document.getElementById("edit-group");
    groupEdit.style.display = "flex";
    groupDelete = document.getElementById("delete-group");
    groupDelete.style.display = "flex";
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

    var image = document.getElementById(divID)
    image.style.backgroundSize = "cover"
    image.innerHTML = ""
    image.style.backgroundImage = `url('${imageBase64}')`
    image.style.backgroundPosition = "center center"

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
                placeWishlists(wishlists, user_id, group_id);
                
            }

        } else {
            info("Removing member...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlist_id + "/remove?group=" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function placeWishlist(wishlistOject, publicURL) {
    document.getElementById("wishlistName-" + wishlistOject.id).innerHTML = wishlistOject.name
    var wishUpdatedAt = new Date(Date.parse(wishlistOject.wish_updated_at));
    var wishUpdatedAtString = GetDateString(wishUpdatedAt)
    document.getElementById(`wishlistUpdatedAt-${wishlistOject.id}`).innerHTML = wishUpdatedAtString

    var wishlist = document.getElementById(`wishlistWrapper-${wishlistOject.id}`)
    var wishlistHTML = wishlist.outerHTML
    wishlist.remove()
    
    var wishlists = document.getElementById(`wishlists-box`)
    wishlists.innerHTML = wishlistHTML + wishlists.innerHTML
}

function removeWishlist(wishlistID, userID) {
    document.getElementById(`wishlistWrapper-${wishlistID}`).remove();
}

function removeGroup(groupID, userID) {
    window.location.href = "/groups"
}

function toggleExpiredWishlists() {
    toggleButtonState = document.getElementById('toggle_expired_wishlists').checked
    wishlistElements = document.getElementById('wishlists-box').children

    for(var i = 0; i < wishlistElements.length; i++) {
        if(wishlistElements[i].classList.contains('wishlist-expired')) {
            if(toggleButtonState) {
                wishlistElements[i].style.display = 'flex'
            } else {
                wishlistElements[i].style.display = 'none'
            }
        }
    }
}