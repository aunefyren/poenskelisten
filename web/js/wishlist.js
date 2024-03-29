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
                <div id="myModal" class="modal">
                    <span class="close selectable">&times;</span>
                    <img class="modal-content" id="modal-img" src="/assets/loading.svg">
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

                            <div class="bottom-right-button" id="edit-wishlist" style="display: none;" title="Edit wishlist">
                                <img class="icon-img  clickable" src="/assets/edit.svg" onclick="wishlist_edit('${user_id}', '${wishlist_id}', '{wishlist_expiration_date}', {wishlist_claimable}, {wishlist_expires}, {wishlist_public});">
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
                            <form action="" class="icon-border" onsubmit="event.preventDefault(); send_wish('${wishlist_id}','${group_id}','${user_id}');">
                                <label for="wish_name">Add a new wish:</label><br>
                                <input type="text" name="wish_name" id="wish_name" placeholder="Wish name" autocomplete="off" required />
                                <label for="wish_note" style="margin-top: 2em;">Optional details:</label><br>
                                <input type="text" name="wish_note" id="wish_note" placeholder="Wish note" autocomplete="off" />
                                <input type="text" name="wish_url" id="wish_url" placeholder="Wish URL" autocomplete="off" />
                                <input type="number" name="wish_price" id="wish_price" placeholder="Wish price in ${currency}" autocomplete="off" />
                                <label id="form-input-icon" for="wish_image" style="margin-top: 2em;">Optional image:</label>
                                <input type="file" name="wish_image" id="wish_image" placeholder="" value="" accept="image/png, image/jpeg" />
                                <button id="register-button" type="submit" href="/">Add wish</button>
                            </form>
                        </div>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Lists...';
    clearResponse();

    // Get the <span> element that closes the modal
    var span = document.getElementsByClassName("close")[0];

    // When the user clicks on <span> (x), close the modal
    span.onclick = function() { 
        document.getElementById("myModal").style.display = "none";
        document.getElementById("modal-img").src = "/assets/loading.svg"
    }

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
                place_wishlist(result.wishlist, result.public_url);

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

function place_wishlist(wishlist_object, public_url) {

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
                console.log(wishes);

                currency = result.currency;
                currency_padding = result.padding;
                try {
                    document.getElementById("wish_price").placeholder = "Wish price in " + currency
                } catch(e) {
                    console.log("Failed to update currency help text. Error: " + e)
                }

                place_wishes(wishes, wishlist_id, group_id, user_id);

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

function place_wishes(wishes_array, wishlist_id, group_id, user_id) {

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
        html += '<div class="wish-note collapsed" id="wish_' + wish_object.id + '_note" title="Note">'
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
}

function send_wish(wishlist_id, group_id, user_id){

    var wish_name = document.getElementById("wish_name").value;
    var wish_note = document.getElementById("wish_note").value;
    var wish_url = document.getElementById("wish_url").value;
    var wish_price = parseFloat(document.getElementById("wish_price").value);
    var wish_image = document.getElementById('wish_image').files[0];

    if(wish_image) {

        if(wish_image.size > 10000000) {
            error("Image exceeds 10MB size limit.")
            return;
        } else if(wish_image.size < 10000) {
            error("Image smaller than 0.01MB size requirement.")
            return;
        }

        wish_image = get_base64(wish_image);
        
        wish_image.then(function(result) {

            var form_obj = { 
                "name" : wish_name,
                "note" : wish_note,
                "url": wish_url,
                "price": wish_price,
                "image_data": result
            };

            var form_data = JSON.stringify(form_obj);

            send_wish_two(form_data, wishlist_id, group_id, user_id);
        
        });

    } else {

        var form_obj = { 
                "name" : wish_name,
                "note" : wish_note,
                "url": wish_url,
                "price": wish_price,
                "image_data": ""
            };

        var form_data = JSON.stringify(form_obj);

        send_wish_two(form_data, wishlist_id, group_id, user_id);

    }

}

function send_wish_two(form_data, wishlist_id, group_id, user_id) {

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
    xhttp.open("post", api_url + "auth/wishes?wishlist=" + wishlist_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function clear_data() {
    document.getElementById("wish_name").value = "";
    document.getElementById("wish_note").value = "";
    document.getElementById("wish_url").value = "";
    document.getElementById("wish_price").value = "";
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
    xhttp.open("delete", api_url + "auth/wishes/" + wish_id);
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
                clear_data();
                
               
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

function wishlist_edit(user_id, wishlist_id, wishlist_expiration_date, wishlist_claimable, wishlist_expires, wishlist_public) {

    var wishlist_title = document.getElementById("wishlist-title").innerHTML;
    var wishlist_description = document.getElementById("wishlist-description").innerHTML;
    var wishlist_expiration = getDateString(wishlist_expiration_date)

    var checked_string = ""
    if(wishlist_claimable) {
        checked_string = "checked"
    }

    var expires_string = ""
    var expireDateClass = ""
    if(wishlist_expires) {
        expires_string = "checked"
        expireDateClass = "wishlist-date-wrapper-extended"
    } else {
        expireDateClass = "wishlist-date-wrapper-minimized"
    }

    var public_string = ""
    if(wishlist_public) {
        public_string = "checked"
    }

    var html = '';

    html += `
        <div class="bottom-right-button" id="edit-wishlist" style="" onclick="cancel_edit_wishlist('${wishlist_id}', '${user_id}');" title="Cancel edit">
            <img class="icon-img  clickable" style="" src="/assets/x.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); update_wishlist('${wishlist_id}', '${user_id}');">
                                
            <label for="wishlist_name">Edit wishlist:</label><br>
            <input type="text" name="wishlist_name" id="wishlist_name" placeholder="Wishlist name" value="${wishlist_title}" autocomplete="off" required />
            
            <input type="text" name="wishlist_description" id="wishlist_description" placeholder="Wishlist description" value="${wishlist_description}" autocomplete="off" required />

            <input class="clickable" onclick="toggeWishListDate('wishlist_date_wrapper_${wishlist_id}')" style="margin-top: 2em;" type="checkbox" id="wishlist_expires" name="wishlist_expires" value="confirm" ${expires_string}>
            <label for="wishlist_expires" style="margin-bottom: 2em;" class="clickable">Does the wishlist expire?</label><br>
            
            <div id="wishlist_date_wrapper_${wishlist_id}" class="wishlist-date-wrapper ${expireDateClass}">
                <label for="wishlist_date">When does your wishlist expire?</label><br>
                <input type="date" name="wishlist_date" id="wishlist_date" placeholder="Wishlist expiration" value="${wishlist_expiration}" autocomplete="off" />
            </div>

            <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="wishlist_claimable" name="wishlist_claimable" value="confirm" ${checked_string}>
            <label for="wishlist_claimable" style="margin-bottom: 1em;" class="clickable">Allow users to claim wishes.</label><br>

            <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="wishlist_public" name="wishlist_public" value="confirm" ${public_string}>
            <label for="wishlist_public" style="margin-bottom: 1em;" class="clickable">Make this wishlist public and shareable.</label><br>
            
            <button id="register-button" type="submit" href="/">Save wishlist</button>

        </form>
    `;

    document.getElementById("wishlist-info-box").innerHTML = html;

}

function update_wishlist(wishlist_id, user_id) {
    var wishlist_name = document.getElementById("wishlist_name").value;
    var wishlist_description = document.getElementById("wishlist_description").value;
    var wishlist_date = document.getElementById("wishlist_date").value;
    var wishlist_date_object = new Date(wishlist_date)
    var wishlist_date_string = wishlist_date_object.toISOString();
    var wishlist_claimable = document.getElementById("wishlist_claimable").checked;
    var wishlist_expires = document.getElementById("wishlist_expires").checked;
    var wishlist_public = document.getElementById("wishlist_public").checked;

    if(wishlist_public && wishlist_claimable) {
        alert("A wishlist cannot have claimable wishes and be public to users without accounts.")
        return;
    }

    var form_obj = { 
        "name" : wishlist_name,
        "description" : wishlist_description,
        "date": wishlist_date_string,
        "claimable": wishlist_claimable,
        "expires": wishlist_expires,
        "public": wishlist_public
    };

    var form_data = JSON.stringify(form_obj);

    console.log(form_data)

    if(!confirm("Are you sure you want to update this wishlist?")) {
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
                reset_wishlist_info_box(user_id, wishlist_id);
                place_wishlist(result.wishlist, result.public_url);
                show_owner_inputs();

            }

        } else {
            info("Updating wishlist...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlist_id + "/update");
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

function edit_wish(wish_id, wishlist_id, group_id, user_id, b64_wish_name, b64_wish_note, b64_wish_url, b64_wish_price, owner_id) {

    var wish_name = fromBASE64(b64_wish_name)
    var wish_note = fromBASE64(b64_wish_note)
    var wish_url = fromBASE64(b64_wish_url)
    var wish_price = fromBASE64(b64_wish_price)

    var html = '';

    html += `

        <div class="bottom-right-button" id="edit-wish" style="" onclick="cancel_edit_wish('${wish_id}', '${wishlist_id}', '${group_id}', '${user_id}');" title="Cancel edit">
            <img class="icon-img  clickable" style="margin: 1em 1em 0 0;" src="/assets/x.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); update_wish('${wish_id}', '${user_id}', '${wishlist_id}', '${group_id}');">
                                
            <label for="wish_name_${wish_id}">Edit wish:</label><br>
            <input type="text" name="wish_name_${wish_id}" id="wish_name_${wish_id}" placeholder="Wish name" value="" autocomplete="off" required />
    
            <label for="wish_note_${wish_id}" style="margin-top: 2em;">Optional details:</label><br>

            <input type="text" name="wish_note_${wish_id}" id="wish_note_${wish_id}" placeholder="Wish note" value="" autocomplete="off" />

            <input type="text" name="wish_url_${wish_id}" id="wish_url_${wish_id}" placeholder="Wish URL" value="" autocomplete="off" />

            <input type="number" name="wish_price_${wish_id}" id="wish_price_${wish_id}" placeholder="Wish price in ${currency}" value="" autocomplete="off" />

            <label id="form-input-icon" for="wish_image_${wish_id}" style="margin-top: 2em;">Replace optional image:</label>
            <input type="file" name="wish_image_${wish_id}" id="wish_image_${wish_id}" placeholder="" value="" accept="image/png, image/jpeg" />
            
            <button id="register-button" type="submit" href="/">Save wish</button>

        </form>
    `;

    document.getElementById("wish_wrapper_" + wish_id).innerHTML = html;

    document.getElementById("wish_name_" + wish_id).value = wish_name;
    document.getElementById("wish_note_" + wish_id).value = wish_note;
    document.getElementById("wish_url_" + wish_id).value = wish_url;
    document.getElementById("wish_price_" + wish_id).value = wish_price;

}

function update_wish(wish_id, user_id, wishlist_id, group_id) {

    if(!confirm("Are you sure you want to update this wish?")) {
        return;
    }

    var wish_name = document.getElementById("wish_name_" + wish_id).value;
    var wish_note = document.getElementById("wish_note_" + wish_id).value;
    var wish_url = document.getElementById("wish_url_" + wish_id).value;
    var wish_price = parseFloat(document.getElementById("wish_price_"+ wish_id).value);
    var wish_image = document.getElementById('wish_image_' + wish_id).files[0];

    if(wish_image) {

        if(wish_image.size > 10000000) {
            error("Image exceeds 10MB size limit.")
            return;
        } else if(wish_image.size < 10000) {
            error("Image smaller than 0.01MB size requirement.")
            return;
        }

        wish_image = get_base64(wish_image);
        
        wish_image.then(function(result) {

            var form_obj = { 
                "name" : wish_name,
                "note" : wish_note,
                "url": wish_url,
                "price": wish_price,
                "image_data": result
            };

            var form_data = JSON.stringify(form_obj);

            update_wish_two(form_data, wish_id, user_id, wishlist_id, group_id);
        
        });

    } else {

        var form_obj = { 
            "name" : wish_name,
            "note" : wish_note,
            "url": wish_url,
            "price": wish_price,
            "image_data": ""
        };

        var form_data = JSON.stringify(form_obj);
        update_wish_two(form_data, wish_id, user_id, wishlist_id, group_id)

    }

}

function update_wish_two(form_data, wish_id, user_id, wishlist_id, group_id) {

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

                var wish_array = generate_wish_html(result.wish, wishlist_id, group_id, user_id);
                var wish_html = wish_array[0];
                var wish_image = wish_array[1];

                document.getElementById("wish_wrapper_" + wish_id).outerHTML = wish_html;

                if(wish_image) {
                    GetWishImageThumbail(result.wish.id)
                }

            }

        } else {
            info("Updating wishlist...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishes/" + wish_id + "/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function cancel_edit_wish(wish_id, wishlist_id, group_id, user_id) {

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

                var wish_array = generate_wish_html(result.wish, wishlist_id, group_id, user_id);
                var wish_html = wish_array[0];
                var wish_image = wish_array[1];

                document.getElementById("wish_wrapper_" + wish_id).outerHTML = wish_html;

                if(wish_image) {
                    GetWishImageThumbail(result.wish.id)
                }

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishes/" + wish_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function cancel_edit_wishlist(wishlist_id, user_id) {

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

                reset_wishlist_info_box(user_id, wishlist_id);
                place_wishlist(result.wishlist, result.public_url);
                show_owner_inputs();

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

function copyPublicLink() {

    /* Get the text field */
    var copyText = document.getElementById("wishlist-url-span").innerHTML

    /* Copy the text inside the text field */
    navigator.clipboard.writeText(copyText)

    alert("URL copied to clipboard.")

}