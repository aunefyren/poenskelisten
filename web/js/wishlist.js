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
                <!-- The Modal -->
                <div id="myModal" class="modal closed">
                    <span class="close clickable" onclick="toggleModal()">&times;</span>
                    <div class="modalContent" id="modalContent">
                    </div>
                    <div id="caption"></div>
                </div>

                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="wishlist-info" id="wishlist-info-box">

                            <div class="loading-icon-wrapper" id="loading-icon-wrapper-wishlist">
                                <img class="loading-icon" src="/assets/loading.svg">
                            </div>

                            <div id="wishlist-title" class="title">
                            </div>

                            <div class="text-body" id="wishlist-description">
                            </div>

                            <div class="text-body" id="wishlist-info">
                            </div>

                            <div class="wishlist-url-wrapper" id="wishlist-url-wrapper" style="display: none;">
                                <div class="wishlist-url-title" id="wishlist-url-title">
                                </div>
                                <div class="wishlist-url" id="wishlist-url">
                                    <span class="wishlist-url-span" id="wishlist-url-span"></span>
                                    <img id="share-wishlist-url-button" class="share-wishlist-url-button clickable hover" src="/assets/copy.svg" style="" title="Click to copy the URL" onclick="copyPublicLink();">
                                </div>
                            </div>

                            <div class="bottom-right-button" id="" style="">
                                <img class="icon-img  clickable" id="collaborators-wishlist" src="/assets/smile.svg" onclick="showWishlistCollaboratorsInWishlist('${wishlist_id}', '${user_id}')" title="Wishlist collaborators" style="margin: 0.25em;">
                                <img class="icon-img  clickable" id="groups-wishlist" src="/assets/users.svg" onclick="showGroupsInWishlist('${wishlist_id}', '${user_id}')" title="Wishlist groups" style="margin: 0.25em; display: none;">
                                <img class="icon-img  clickable" id="edit-wishlist" src="/assets/edit.svg" onclick="editWishlist('${user_id}', '${wishlist_id}')" title="Edit wishlist" style="margin: 0.25em; display: none;">
                                <img class="icon-img  clickable" id="delete-wishlist" src="/assets/trash-2.svg" onclick="deleteWishlist('${wishlist_id}', '${user_id}')" title="Delete wishlist" style="margin: 0.25em; display: none;">
                            </div>

                        </div>

                    </div>

                    <div class="module">

                        <div id="wishlists-title" class="title">
                            Wishes:
                        </div>

                        <div id="wishes-box" class="wishes">
                            <div class="loading-icon-wrapper" id="loading-icon-wrapper">
                                <img class="loading-icon" src="/assets/loading.svg">
                            </div>
                        </div>

                        <div id="wish-input" class="wish-input">
                            <button id="register-button" onClick="createWish('${wishlist_id}', '${user_id}', '{{currency}}');" type="" href="/">Create new wish</button>
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
                placeWishlist(result.wishlist, result.public_url);

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists/" + wishlist_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWishlist(wishlist_object, public_url) {

    try {
        document.getElementById("loading-icon-wrapper-wishlist").style.display = "none"
    } catch(e) {
        console.log("Error: " + e)
    }

    newWishButton = document.getElementById("register-button")
    newWishButton.outerHTML = newWishButton.outerHTML.replace("{{currency}}", wishlist_object.currency)

    document.getElementById("wishlist-title").innerHTML = wishlist_object.name
    document.getElementById("wishlist-description").innerHTML = wishlist_object.description
    document.getElementById("wishlist-info").innerHTML = "<br>By: " + wishlist_object.owner.first_name + " " + wishlist_object.owner.last_name + "."

    try {
        
        var expiration = new Date(Date.parse(wishlist_object.date));
        expiration_string = GetDateString(expiration)

        if(wishlist_object.expires) {
            document.getElementById("wishlist-info").innerHTML += "<br>Expires: " + expiration_string
        } else {
            document.getElementById("wishlist-info").innerHTML += "<br>Does not expire."
        }

        var box = document.getElementById("wishlist-info-box")

        document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_expiration_date}', wishlist_object.date)
        document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_expires}', wishlist_object.expires)

        if(wishlist_object.claimable) {
            document.getElementById("wishlist-info").innerHTML += "<br>Wishes are claimable.";
            box = document.getElementById("wishlist-info-box")
            document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_claimable}', "true");
        } else {
            document.getElementById("wishlist-info").innerHTML += "<br>Wishes are not claimable.";
            box = document.getElementById("wishlist-info-box")
            document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_claimable}', "false");
        }

        if(wishlist_object.public) {
            document.getElementById("wishlist-info").innerHTML += "<br>Wishlist is public to users without accounts.";
            box = document.getElementById("wishlist-info-box")
            document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_public}', "true");

            document.getElementById("wishlist-url-wrapper").style.display = "flex";
            document.getElementById("wishlist-url-title").innerHTML = "Public URL:";
            document.getElementById("wishlist-url-span").innerHTML = public_url + "/wishlists/public/" + wishlist_object.public_hash;
        } else {
            document.getElementById("wishlist-info").innerHTML += "<br>Wishlist is private to shared groups.";
            box = document.getElementById("wishlist-info-box")
            document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_public}', "false");
        }

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
                //console.log(wishes);

                currency = result.currency;
                currency_padding = result.currency_padding;
                currency_left = result.currency_left;
                try {
                    document.getElementById("wish_price").placeholder = "Wish price in " + currency
                } catch(e) {
                    console.log("Failed to update currency help text. Error: " + e)
                }

                placeWishes(wishes, wishlist_id, user_id);

                var collaborator = false;
                for(var i = 0; i < result.collaborators.length; i++) {
                    if(result.collaborators[i] == user_id) {
                        collaborator = true;
                    }
                }

                if(result.owner_id == user_id || collaborator) {
                    show_owner_inputs();
                }

            }

        } else {
            info("Loading wishes...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishes?wishlist=" + wishlist_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWishes(wishes_array, wishlist_id, user_id) {

    var html = ''
    var wish_id_array = []

    for(var i = 0; i < wishes_array.length; i++) {

        var function_result = generate_wish_html(wishes_array[i], wishlist_id, user_id);
        var new_html = function_result[0]
        var wish_image = function_result[1]

        if(wish_image) {
            wish_id_array.push(wishes_array[i].id)
        }

        html += new_html
        
    }

    if(wishes_array.length == 0) {
        info("Looks like this wishlist is empty...");

        try {
            document.getElementById("loading-icon-wrapper").style.display = "none"
        } catch(e) {
            console.log("Error: " + e)
        }
    }

    wishlist_object = document.getElementById("wishes-box")
    wishlist_object.innerHTML = html

    if(wish_id_array.length > 0) {
        for(var i = 0; i < wish_id_array.length; i++) {
            GetWishImageThumbail(wish_id_array[i])
        }
    }

}

function generate_wish_html(wish_object, wishlist_id, user_id) {

    var html = '';
    var wish_with_image = false;

    var owner_id = wish_object.owner_id.id
    var wishlist_ownerID = wish_object.wishlist_owner.id

    var collaborator = false;
    for(var i = 0; i < wish_object.collaborators.length; i++) {
        if(wish_object.collaborators[i].user.id == user_id) {
            collaborator = true;
            break;
        }
    }

    var wishUpdatedAt = new Date(Date.parse(wish_object.updated_at));
    var wishUpdatedAtString = GetDateString(wishUpdatedAt);

    if(wish_object.wishclaim.length > 0 && user_id != owner_id && !collaborator && wish_object.wish_claimable) {
        var transparent = " transparent"
    } else {
        var transparent = ""
    }

    html += '<div class="wish-wrapper ' + transparent + '" id="wish_wrapper_' + wish_object.id + '">'

    html += '<div class="wish" id="wish_' + wish_object.id + '">'

    html += `
        <div class="unselectable wish-updatedat" title="Updated at">
            <div class="wish-updatedat-text">Updated at:</div>
            <div class="wish-updatedat-date">
                ${wishUpdatedAtString}
            </div>
        </div>
    `;
    
    html += '<div class="wish-title">'
    html += '<div class="profile-icon">'
    html += '<img class="icon-img " src="/assets/gift.svg">'
    html += '</div>'

    html += wish_object.name

    if(wish_object.price && wish_object.price != 0) {

        var currency_string = currency
        if(currency_padding) {
            if(currency_left) {
                currency_string = currency_string + " ";
            } else {
                currency_string = " " + currency_string;
            }
        }

        html += '<div class="wish-price unselectable" title="Price">'

        if(currency_left) {
            html += currency_string + wish_object.price
        } else {
            html += wish_object.price + currency_string
        }
        
        html += '</div>'
    }

    html += '</div>'

    html += '<div class="profile">'

    if(wish_object.note !== "" || wish_object.image) {
        html += `<div class="profile-icon clickable" onclick="toggle_wish('${wish_object.id}')" title="Expandable">`;

        if(wish_object.image) {
            html += '<img id="wish_' + wish_object.id + '_arrow" class="icon-img " src="/assets/chevron-down.svg">'
        } else {
            html += '<img id="wish_' + wish_object.id + '_arrow" class="icon-img " src="/assets/chevron-right.svg">'
        }

        html += '</div>'
    }

    if(wish_object.url !== "") {
        html += `<div class="profile-icon clickable" onclick="openURLModal('${wish_object.url}');" title="Go to webpage">`
        html += '<img class="icon-img " src="/assets/link.svg">'
        html += '</div>'
    }

    if(user_id == owner_id || collaborator || user_id == wishlist_ownerID) {
        html += `<div class="profile-icon clickable" title="Edit wish" onclick="editWish('${wish_object.id}', '${wishlist_id}', '${group_id}', '${user_id}')">`;
        html += '<img class="icon-img " src="/assets/edit.svg">'
        html += '</div>'

        html += `<div class="profile-icon clickable" title="Delete wish" onclick="deleteWish('${wish_object.id}', '${wishlist_id}', '${group_id}', '${user_id}')">`;
        html += '<img class="icon-img " src="/assets/trash-2.svg">'
        html += '</div>'
    } else if(wish_object.wishclaim.length > 0 && wish_object.wish_claimable) {
        for(var j = 0; j < wish_object.wishclaim.length; j++) {
            if(user_id !== wish_object.wishclaim[j].user.id) {
                html += `<div class="profile-icon clickable" title="Wish is claimed by ${wish_object.wishclaim[j].user.first_name} ${wish_object.wishclaim[j].user.last_name}." onclick="alert('Wish is claimed by ${wish_object.wishclaim[j].user.first_name} ${wish_object.wishclaim[j].user.last_name}.')">`
                html += '<img class="icon-img " src="/assets/lock.svg">'
                html += '</div>'
            } else {
                html += '<div class="profile-icon clickable" title="Claimed by you, click to unclaim.">'
                html += `<img class="icon-img " src="/assets/unlock.svg" onclick="unclaim_wish('${wish_object.id}', '${wishlist_id}', '${group_id}', '${user_id}')")>`;
                html += '</div>'
            }
        }
    } else if(wish_object.wish_claimable) {
        html += `<div class="profile-icon clickable" title="Claim this gift" onclick="claim_wish('${wish_object.id}', '${wishlist_id}', '${group_id}', '${user_id}')">`;
        html += '<img class="icon-img " src="/assets/check.svg">'
        html += '</div>'
    }
    html += '</div>'

    html += '</div>'

    if(wish_object.image) {
        html += '<div class="wish-note expanded" style="display: flex !important;" id="wish_' + wish_object.id + '_note" title="Note">'
    } else {
        html += '<div class="wish-note collapsed" style="display: none !important;" id="wish_' + wish_object.id + '_note" title="Note">'
    }

    if(wish_object.image) {
        html += `<div class="wish-image-thumbnail clickable" onclick="toggle_wish_modal('${wish_object.id}')">`;
        html += '<img style="width: 100%; height: 100%;" class="wish-image-thumbnail-img" id="wish-image-thumbnail-img-' + wish_object.id  + '" src="/assets/loading.svg">'
        html += '</div>'

        wish_with_image = true
    }

    html += '<div class="wish-note-text">'
    html += wish_object.note
    html += '</div>'

    html += '</div>'

    html += '</div>'

    return [html, wish_with_image];

}

function toggle_wish(wishid) {
    wishnote = document.getElementById("wish_" + wishid + "_note");
    wishnotearrow = document.getElementById("wish_" + wishid + "_arrow");

    if(wishnote.classList.contains("collapsed")) {
        wishnote.classList.remove("collapsed")
        wishnote.classList.add("expanded")
        wishnote.style.display = "flex"
        wishnotearrow.src = "/assets/chevron-down.svg"
    } else {
        wishnote.classList.remove("expanded")
        wishnote.classList.add("collapsed")
        wishnote.style.display = "none"
        wishnotearrow.src = "/assets/chevron-right.svg"
    }
}

function show_owner_inputs() {
    wishinput = document.getElementById("wish-input");
    wishinput.style.display = "inline-block"
    wishlistedit = document.getElementById("edit-wishlist");
    wishlistedit.style.display = "flex"
    wishlistDelete = document.getElementById("delete-wishlist");
    wishlistDelete.style.display = "flex"
    wishlistGroups = document.getElementById("groups-wishlist");
    wishlistGroups.style.display = "flex"
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
                placeWishes(wishes, wishlist_id, user_id);
            }

        } else {
            info("Claiming wish...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishes/" + wish_id + "/claim");
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
            }

        } else {
            info("Un-claiming wish...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishes/" + wish_id + "/unclaim");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function reset_wishlist_info_box(user_id, wishlist_id) {
    var html = `
    <div class="loading-icon-wrapper" id="loading-icon-wrapper-wishlist">
        <img class="loading-icon" src="/assets/loading.svg">
    </div>

    <div id="wishlist-title" class="title">
    </div>

    <div class="text-body" id="wishlist-description">
    </div>

    <div class="text-body" id="wishlist-info">
    </div>

    <div class="wishlist-url-wrapper" id="wishlist-url-wrapper" style="display: none;">
        <div class="wishlist-url-title" id="wishlist-url-title">
        </div>
        <div class="wishlist-url" id="wishlist-url">
            <span class="wishlist-url-span" id="wishlist-url-span"></span>
            <img id="share-wishlist-url-button" class="share-wishlist-url-button clickable hover" src="/assets/copy.svg" style="" title="Click to copy the URL" onclick="copyPublicLink();">
        </div>
    </div>

    <div class="bottom-right-button" id="edit-wishlist" style="display: none;">
        <img class="icon-img  clickable" src="/assets/edit.svg" onclick="wishlist_edit('${user_id}', '${wishlist_id}', '{wishlist_expiration_date}', {wishlist_claimable}, {wishlist_expires}, {wishlist_public});">
    </div>
    `;

    document.getElementById("wishlist-info-box").innerHTML = html;
}

function toggle_wish_modal(wishID) {
    modalHTML = `
        <div class="modalWishImage">
            <img id="modal-img" src="/assets/loading.gif">
        </div>
    `;
    toggleModal(modalHTML);
    GetWishImage(wishID);
}

function GetWishImage(wishID) {

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");

                // Disable modal
                document.getElementById("myModal").style.display = "none";

                return;
            }
            
            if(result.error) {

                error(result.error);
                document.getElementById("myModal").style.display = "none";

            } else {

                PlaceWishImageInModal(result.image)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "both/wishes/" + wishID + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceWishImageInModal(imageBase64) {
    
    document.getElementById("modal-img").src = imageBase64

}

function GetWishImageThumbail(wishID) {

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

                PlaceWishImageThumbail(result.image, wishID)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "both/wishes/" + wishID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceWishImageThumbail(imageBase64, wishID) {

    document.getElementById("wish-image-thumbnail-img-" + wishID).src = imageBase64

}

function copyPublicLink() {

    /* Get the text field */
    var copyText = document.getElementById("wishlist-url-span").innerHTML

    /* Copy the text inside the text field */
    navigator.clipboard.writeText(copyText)

    alert("URL copied to clipboard.")

}

function placeWish(wishObject, wishlistID, groupID, userID) {
    var wish_array = generate_wish_html(wishObject, wishlistID, groupID, userID);
    var wish_html = wish_array[0];
    var wish_image = wish_array[1];

    document.getElementById("wish_wrapper_" + wishObject.id).remove();
    document.getElementById("wishes-box").innerHTML = wish_html + document.getElementById("wishes-box").innerHTML

    if(wish_image) {
        GetWishImageThumbail(result.wish.id)
    }
}

function removeWishlist(wishlistID, userID) {
    window.location.href = "/wishlists";
}