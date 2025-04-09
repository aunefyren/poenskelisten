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
        wishlist_hash = document.URL.substring(string_index+1);
    }
    catch {
        wishlist_hash = 0
    }

    var html = `
        <!-- The Modal -->
        <div id="myModal" class="modal">
            <span class="close clickable">&times;</span>
            <img class="modal-content" id="modal-img" src="/assets/loading.gif">
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

            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'A wishlist shared with you!';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
    } else {
        showLoggedOutMenu();
    }

    getPublicWishlist(wishlist_hash);
}

function getPublicWishlist(wishlist_hash){

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

                currency = result.currency;
                currency_padding = result.padding;
                
                placeWishlist(result.wishlist);
                placeWishes(result.wishlist.wishes)

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "open/wishlists/public/" + wishlist_hash);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWishlist(wishlist_object) {

    try {
        document.getElementById("loading-icon-wrapper-wishlist").style.display = "none"
    } catch(e) {
        console.log("Error: " + e)
    }

    document.getElementById("wishlist-title").innerHTML = wishlist_object.name
    document.getElementById("wishlist-description").innerHTML = wishlist_object.description
    document.getElementById("wishlist-info").innerHTML += "<br>By: " + wishlist_object.owner.first_name + " " + wishlist_object.owner.last_name + "."

    try {
        
        var expiration = new Date(Date.parse(wishlist_object.date));
        expiration_string = expiration.toLocaleDateString();

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
        } else {
            document.getElementById("wishlist-info").innerHTML += "<br>Wishlist is private to shared groups.";
            box = document.getElementById("wishlist-info-box")
            document.getElementById("wishlist-info-box").innerHTML = box.innerHTML.replace('{wishlist_public}', "false");
        }

    } catch(err) {
        console.log("Failed to parse datetime. Error: " + err)
    }

}

function placeWishes(wishes_array, wishlist_id, group_id, user_id) {

    var html = ''
    var wish_id_array = []

    for(var i = 0; i < wishes_array.length; i++) {

        var function_result = generate_wish_html(wishes_array[i], wishlist_id, group_id, user_id);
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

function generate_wish_html(wish_object, wishlist_id, group_id, user_id) {
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

    if(wish_object.wishclaim.length > 0 && user_id != owner_id && !collaborator && wish_object.wish_claimable) {
        var transparent = " transparent"
    } else {
        var transparent = ""
    }

    html += '<div class="wish-wrapper ' + transparent + '" id="wish_wrapper_' + wish_object.id + '">'

    html += '<div class="wish" id="wish_' + wish_object.id + '">'
    
    html += '<div class="wish-title">'
    html += '<div class="profile-icon">'
    html += '<img class="icon-img " src="/assets/gift.svg">'
    html += '</div>'

    html += wish_object.name

    if(wish_object.price != 0) {

        var currency_string = currency
        if(currency_padding) {
            currency_string = " " + currency_string;
        }

        html += '<div class="wish-price unselectable" title="Price">'
        html += wish_object.price + currency_string
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
        html += `<div class="profile-icon clickable" onclick="window.open('${wish_object.url}', \'_blank\')" title="Go to webpage">`
        html += '<img class="icon-img " src="/assets/link.svg">'
        html += '</div>'
    }

    if(user_id == owner_id || collaborator || user_id == wishlist_ownerID) {

        var b64_wish_name = toBASE64(wish_object.name)
        var b64_wish_note = toBASE64(wish_object.note)
        var b64_wish_url = toBASE64(wish_object.url)
        var b64_wish_price = toBASE64(wish_object.price.toString())

        html += `<div class="profile-icon clickable" title="Edit wish" onclick="edit_wish('${wish_object.id}', '${wishlist_id}', '${group_id}', '${user_id}', '${b64_wish_name}', '${b64_wish_note}', '${b64_wish_url}', '${b64_wish_price}', '${owner_id}')">`;
        html += '<img class="icon-img " src="/assets/edit.svg">'
        html += '</div>'

        html += `<div class="profile-icon clickable" title="Delete wish" onclick="delete_wish('${wish_object.id}', '${wishlist_id}', '${group_id}', '${user_id}')">`;
        html += '<img class="icon-img " src="/assets/trash-2.svg">'
        html += '</div>'
    } else if(wish_object.wishclaim.length > 0 && wish_object.wish_claimable) {
        for(var j = 0; j < wish_object.wishclaim.length; j++) {
            if(user_id !== wish_object.wishclaim[j].user.id) {
                html += '<div class="profile-icon" title="Claimed by ' + wish_object.wishclaim[j].user.first_name + ' ' + wish_object.wishclaim[j].user.last_name + '">'
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
        html += '<div class="wish-note collapsed" id="wish_' + wish_object.id + '_note" title="Note">'
    }

    if(wish_object.image) {
        html += `<div class="wish-image-thumbnail clickable" onclick="toggle_wish_modal('${wish_object.id}')">`;
        html += '<img style="width: 100%; height: 100%;" class="wish-image-thumbnail-img" id="wish-image-thumbnail-img-' + wish_object.id  + '" src="/assets/loading.gif">'
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

function toggle_wish_modal(wishID) {

    document.getElementById("myModal").style.display = "block";
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