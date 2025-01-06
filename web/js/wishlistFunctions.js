function createNewWishlist(groupContextID, userID, wishlistObjectBase64) {
    var html = '';

    try {
        var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))
    } catch (error) {
        var now = new Date
        var wishlistDate = now.toISOString().split('T')[0];
        var wishlistObject = {
            "name": "",
            "description" : "",
            "expires": true,
            "date": wishlistDate,
            "claimable": true,
            "public": false
        }
        wishlistObjectBase64 = toBASE64(JSON.stringify(wishlistObject))
    }

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); createNewWishlistTwo('${groupContextID}', '${userID}', '${wishlistObjectBase64}');">      
            <label for="wishlist_name" style="">Create a new wishlist:</label><br>
            <input type="text" name="wishlist_name" id="wishlist_name" placeholder="Wishlist name" autocomplete="off" required value="${wishlistObject.name}" />
            
            <textarea name="wishlist_description" id="wishlist_description" placeholder="Wishlist description" autocomplete="off" rows="3" required>${wishlistObject.description}</textarea>
            
            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createNewWishlistTwo(groupContextID, userID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    try {
        var wishlistName = document.getElementById("wishlist_name").value;
        var wishlistDescription = document.getElementById("wishlist_description").value;
        wishlistObject.name = wishlistName
        wishlistObject.description = wishlistDescription
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }
    
    wishlistObjectBase64 = toBASE64(JSON.stringify(wishlistObject))

    if(wishlistObject.expires && wishlistObject.date) {
        try {
            var wishlist_date_object = new Date(wishlistObject.date)
            var wishlistDate = wishlist_date_object.toISOString().split('T')[0];
        } catch(e) {
            alert("Invalid date selected.");
            return;
        }
    } else {
        var now = new Date
        var wishlistDate = now.toISOString().split('T')[0];
    }

    var checkedHTML = ""
    var extendedHTML = "wishlist-date-wrapper-minimized"
    if(wishlistObject.expires) {
        checkedHTML = "checked"
        extendedHTML = "wishlist-date-wrapper-extended"
    }

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="createNewWishlist('${groupContextID}', '${userID}', '${wishlistObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); createNewWishlistThree('${groupContextID}', '${userID}', '${wishlistObjectBase64}');">
            <input class="clickable" onclick="toggleWishListDate('wishlist_date_wrapper_new')" style="" type="checkbox" id="wishlist_expires" name="wishlist_expires" value="confirm" ${checkedHTML}>
            <label for="wishlist_expires" style="" class="clickable">Does the wishlist expire?</label><br>
            
            <div id="wishlist_date_wrapper_new" class="wishlist-date-wrapper ${extendedHTML}" style="margin-top: 1em;">
                <label for="wishlist_date">When does your wishlist expire?</label><br>
                <input type="date" name="wishlist_date" id="wishlist_date" placeholder="Wishlist expiration" autocomplete="off" value="${wishlistDate}" />
            </div>

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createNewWishlistThree(groupContextID, userID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    try {
        var wishlistExpires = document.getElementById("wishlist_expires").checked;
        var wishlistDate = document.getElementById("wishlist_date").value;

        if(wishlistExpires) {
            try {
                var wishlist_date_object = new Date(wishlistDate)
                var wishlistDate = wishlist_date_object.toISOString();
            } catch(e) {
                alert("Invalid date selected.");
                return;
            }
        } else {
            var now = new Date
            var wishlistDate = now.toISOString().split('T')[0];
        }

        wishlistObject.expires = wishlistExpires
        wishlistObject.date = wishlistDate
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }

    wishlistObjectBase64 = toBASE64(JSON.stringify(wishlistObject))

    var checkedHTML = ""
    if(wishlistObject.claimable) {
        checkedHTML = "checked"
    }
    var checkedTwoHTML = ""
    if(wishlistObject.public) {
        checkedTwoHTML = "checked"
    }

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="createNewWishlistTwo('${groupContextID}', '${userID}', '${wishlistObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); createNewWishlistFour('${groupContextID}', '${userID}', '${wishlistObjectBase64}');">
            <input class="clickable" onclick="" style="" type="checkbox" id="wishlist_claimable" name="wishlist_claimable" value="confirm" ${checkedHTML}>
            <label for="wishlist_claimable" style="margin-bottom: 1em;" class="clickable">Allow users to claim wishes.</label><br>

            <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="wishlist_public" name="wishlist_public" value="confirm" ${checkedTwoHTML}>
            <label for="wishlist_public" style="margin-bottom: 1em;" class="clickable">Make this wishlist public and shareable.</label><br>
            
            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createNewWishlistFour(groupContextID, userID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    try {
        var wishlistClaimable = document.getElementById("wishlist_claimable").checked;
        var wishlistPublic = document.getElementById("wishlist_public").checked;
        wishlistObject.claimable = wishlistClaimable
        wishlistObject.public = wishlistPublic
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }
    
    wishlistObjectBase64 = toBASE64(JSON.stringify(wishlistObject))
    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="createNewWishlistThree('${groupContextID}', '${userID}', '${wishlistObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); createWishlist('${groupContextID}', '${userID}', '${wishlistObjectBase64}');">
            <label for="addToGroups" style="" class="">Add the wishlist to any groups?</label><br>
            <div id="addToGroups">
            </div>

            <button id="register-button" type="submit" href="/">Create wishlist</button>
        </form>
    `;

    toggleModal(html);
    getGroupsForWishlist(groupContextID);
}

function getGroupsForWishlist(groupContextID) {
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
                placeGroupCheckboxes(result.groups, groupContextID);
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

function placeGroupCheckboxes(groupArray, groupContextID) {
    groupOfCheckboxes = document.getElementById("addToGroups");

    if(groupArray.length == 0) {
        groupOfCheckboxes.innerHTML = "You are not a member of any groups :("
        return;
    }
    groupArray.forEach(group => {
        var groupHTML = ""

        if(groupContextID && groupContextID == group.id) {
            groupHTML = "checked"
        }

        html = `
            <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="addGroup-${group.id}" name="addGroup-${group.id}" value="confirm" ${groupHTML}>
            <label for="addGroup-${group.id}" style="margin-bottom: 1em;" class="clickable">${group.name}</label><br>
        `;
        groupOfCheckboxes.innerHTML += html;
    });
}

function createWishlist(groupContextID, userID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    var groupsToAdd = [];
    var groupsToAddChildren = document.getElementById("addToGroups").children
    for(var i = 0; i < groupsToAddChildren.length; i++) {
        if(groupsToAddChildren[i].checked) {
            var groupIDToAdd = groupsToAddChildren[i].name.replace("addGroup-", '');
            groupsToAdd.push(groupIDToAdd);
        }
    }

    var form_obj = { 
        "name" : wishlistObject.name,
        "description" : wishlistObject.description,
        "date": wishlistObject.date,
        "groups": groupsToAdd,
        "claimable": wishlistObject.claimable,
        "expires": wishlistObject.expires,
        "public": wishlistObject.public
    };
    var form_data = JSON.stringify(form_obj);

    var groupContextIDString = ""
    if(groupContextID && groupContextID != "false") {
        groupContextIDString = "?groupContextID=" + groupContextID
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
                wishlists = result.wishlists;
                placeWishlists(wishlists, userID, groupContextID);
            }

        } else {
            info("Creating wishlist...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists" + groupContextIDString);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function toggleWishListDate(wrapperID) {
    try {
        var wrapper = document.getElementById(wrapperID)

        if (wrapper.classList.contains('wishlist-date-wrapper-extended')) {
            wrapper.classList.remove('wishlist-date-wrapper-extended')
            wrapper.classList.add('wishlist-date-wrapper-minimized')
        } else if (wrapper.classList.contains('wishlist-date-wrapper-minimized')) {
            wrapper.classList.remove('wishlist-date-wrapper-minimized')
            wrapper.classList.add('wishlist-date-wrapper-extended')
        }
    } catch(e) {
        console.log("Failed to toggle wishlist date wrapper: " + e)
    }
}

function add_groups(wishlist_id, user_id) {

    var selected_members = [];
    var select_list = document.getElementById("wishlist-input-members-" + wishlist_id)

    for (var i=0; i < select_list.options.length; i++) {
        opt = select_list.options[i];
    
        if (opt.selected) {
            selected_members.push(opt.value);
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
                showWishlistsInGroup(wishlists, user_id);
                
            }

        } else {
            info("Adding groups...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlist_id + "/add");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function deleteWishlist(wishlistID, userID) {
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
                removeWishlist(wishlistID, userID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/wishlists/" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function editWishlist(userID, wishlistID){
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
                editWishlistTwo(userID, wishlistID, toBASE64(JSON.stringify(result.wishlist)))
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists/" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function editWishlistTwo(userID, wishlistID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))
    var html = '';

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); editWishlistThree('${wishlistID}', '${userID}', '${wishlistObjectBase64}');">      
            <label for="wishlist_name" style="">Edit wishlist:</label><br>
            <input type="text" name="wishlist_name" id="wishlist_name" placeholder="Wishlist name" autocomplete="off" required value="${wishlistObject.name}" />
            
            <textarea name="wishlist_description" id="wishlist_description" placeholder="Wishlist description" autocomplete="off" rows="3" required>${wishlistObject.description}</textarea>

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function editWishlistThree(wishlistID, userID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    try {
        var wishlistName = document.getElementById("wishlist_name").value;
        var wishlistDescription = document.getElementById("wishlist_description").value;
        wishlistObject.name = wishlistName
        wishlistObject.description = wishlistDescription
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }
    
    wishlistObjectBase64 = toBASE64(JSON.stringify(wishlistObject))

    if(wishlistObject.expires && wishlistObject.date) {
        try {
            var wishlist_date_object = new Date(wishlistObject.date)
            var wishlistDate = wishlist_date_object.toISOString().split('T')[0];
        } catch(e) {
            alert("Invalid date selected.");
            return;
        }
    } else {
        var now = new Date
        var wishlistDate = now.toISOString().split('T')[0];
    }

    var checkedHTML = ""
    var extendedHTML = "wishlist-date-wrapper-minimized"
    if(wishlistObject.expires) {
        checkedHTML = "checked"
        extendedHTML = "wishlist-date-wrapper-extended"
    }

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="editWishlistTwo('${userID}', '${wishlistID}', '${wishlistObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); editWishlistFour('${wishlistID}', '${userID}', '${wishlistObjectBase64}');">
            <input class="clickable" onclick="toggleWishListDate('wishlist_date_wrapper_new')" style="" type="checkbox" id="wishlist_expires" name="wishlist_expires" value="confirm" ${checkedHTML}>
            <label for="wishlist_expires" style="" class="clickable">Does the wishlist expire?</label><br>
            
            <div id="wishlist_date_wrapper_new" class="wishlist-date-wrapper ${extendedHTML}" style="margin-top: 1em;">
                <label for="wishlist_date">When does your wishlist expire?</label><br>
                <input type="date" name="wishlist_date" id="wishlist_date" placeholder="Wishlist expiration" autocomplete="off" value="${wishlistDate}"/>
            </div>

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function editWishlistFour(wishlistID, userID, wishlistObjectBase64) {
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    try {
        var wishlistExpires = document.getElementById("wishlist_expires").checked;
        var wishlistDate = document.getElementById("wishlist_date").value;
        wishlistObject.expires = wishlistExpires

        if(wishlistExpires) {
            try {
                var wishlist_date_object = new Date(wishlistDate)
                var wishlistDate = wishlist_date_object.toISOString();
            } catch(e) {
                alert("Invalid date selected.");
                return;
            }
        } else {
            var wishlistDate = "2006-01-02T15:04:05.000Z";
        }

        wishlistObject.date = wishlistDate
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }
    
    wishlistObjectBase64 = toBASE64(JSON.stringify(wishlistObject))

    var claimableHTML = ""
    var publicHTML = ""
    if(wishlistObject.claimable) {
        claimableHTML = "checked"
    }
    if(wishlistObject.public) {
        publicHTML = "checked"
    }

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="editWishlistThree('${wishlistID}', '${userID}', '${wishlistObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); editWishlistFive('${wishlistID}', '${userID}', '${wishlistObjectBase64}');">
            <input class="clickable" onclick="" style="" type="checkbox" id="wishlist_claimable" name="wishlist_claimable" value="confirm" ${claimableHTML}>
            <label for="wishlist_claimable" style="margin-bottom: 1em;" class="clickable">Allow users to claim wishes.</label><br>

            <input class="clickable" onclick="" style="margin-top: 1em;" type="checkbox" id="wishlist_public" name="wishlist_public" value="confirm" ${publicHTML}>
            <label for="wishlist_public" style="margin-bottom: 1em;" class="clickable">Make this wishlist public and shareable.</label><br>
            
            <button id="register-button" type="submit" href="/">Update wishlist</button>
        </form>
    `;

    toggleModal(html);
}

function editWishlistFive(wishlistID, userID, wishlistObjectBase64) {
    var wishlistClaimable = document.getElementById("wishlist_claimable").checked;
    var wishlistPublic = document.getElementById("wishlist_public").checked;
    var wishlistObject = JSON.parse(fromBASE64(wishlistObjectBase64))

    if(wishlistPublic && wishlistClaimable) {
        alert("A wishlist cannot have claimable wishes and be public to users without accounts.")
        return;
    }

    var form_obj = { 
        "name" : wishlistObject.name,
        "description" : wishlistObject.description,
        "date": wishlistObject.date,
        "claimable": wishlistClaimable,
        "expires": wishlistObject.expires,
        "public": wishlistPublic
    };
    var form_data = JSON.stringify(form_obj);

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
                success(result.message)
                placeWishlist(result.wishlist, result.public_url);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function showGroupsInWishlist(wishlistID, userID, groupContextID){
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
                showGroupsInWishlistTwo(result.groups, userID, wishlistID, groupContextID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/groups?memberOfWishlistID=" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function showGroupsInWishlistTwo(groupArray, userID, wishlistID, groupContextID) {
    var html = '<div class="group-members" id="group_' + wishlistID + '_members" style="">'
    html += '<div class="text-body">The wishlist is in these groups:</div>'

    if(groupArray.length > 0) {
        for(var j = 0; j < groupArray.length; j++) {
            html += `
                <div class="group-member hoverable-opacity" title="Wishlist">
                    <div class="group-title">
                        <div class="profile-icon" id="group_member_image_${groupArray[j].id}">
                            <img class="icon-img " src="/assets/users.svg">
                        </div>

                        <div class="group-title-text">
                            ${groupArray[j].name}
                        </div>
                    </div>

                    <div class="profile-icon clickable" onclick="removeWishlistFromGroup('${wishlistID}', '${groupArray[j].id}', '${userID}', '${groupContextID}')" title="Remove wishlist from group">
                        <img class="icon-img " src="/assets/x.svg">
                    </div>
                </div>
            `;
        }
    } else {
        html += `None :(`;
    }
    html += `
        <div id="wishlist-input" class="wishlist-input">
            <button id="register-button" onClick="addGroupsToWishlist('${wishlistID}', '${userID}');" type="" href="/">Add wishlist to group</button>
        </div>
    `;

    toggleModal(html);
}

function removeWishlistFromGroup(wishlistID, groupID, userID, groupContextID) {
    if(!confirm("Are you sure you want to remove your wishlist from this group?")) {
        return;
    }

    var form_obj = { 
        "group_id" : groupID
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
                if(groupID == groupContextID) {
                    removeWishlist(wishlistID, userID)
                    toggleModal(false);
                } else {
                    showGroupsInWishlist(wishlistID, userID);
                }
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlistID + "/remove");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function addGroupsToWishlist(wishlistID, userID) {
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
                addGroupsToWishlistTwo(result.groups, userID, wishlistID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/groups?notAMemberOfWishlistID=" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addGroupsToWishlistTwo(groupArray, userID, wishlistID) {
    var groupListHTML = '<datalist id="userList">'
    var groupListIDHTML = '<datalist id="userIDList">'
    for (let index = 0; index < groupArray.length; index++) {
        const displayName = groupArray[index].name;
        const id = groupArray[index].id

        groupListHTML += `<option value="${displayName}">${displayName}</option>`
        groupListIDHTML += `<option value="${id}">${displayName}</option>`
    }
    groupListHTML += '</datalist>'
    groupListIDHTML += '</datalist>'

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="showGroupsInWishlist('${wishlistID}', '${userID}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <label for="newMemberID" style="">Add the wishlist to groups:</label><br>
        <div class="addNewMemberButtons">
            <input type="text" name="newMemberID" id="newMemberID" list="userList" placeholder="My cool group" value="" autocomplete="off" style="margin: 1em 0.5em 1em 1em;" />
            <div class="profile-icon clickable" onclick="addGroupToSelection()" title="Add member" style="margin: 0 1em 0 0;">
                <img class="icon-img" src="/assets/plus.svg">
            </div>
        </div>

        ${groupListHTML}
        ${groupListIDHTML}

        <div id="newMembers" class="newMembers">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); addGroupsToWishlistThree('${wishlistID}', '${userID}');">
            <button id="register-button" type="submit" href="/">Add to group</button>
        </form>
    `;

    toggleModal(html);
}

function addGroupToSelection() {
    var newMemberName = document.getElementById("newMemberID").value
    var newMemberID = ""
    if(!newMemberName || newMemberName == "") {
        return;
    }

    var membersDatalistDiv = document.getElementById("userIDList")
    for (let index = 0; index < membersDatalistDiv.children.length; index++) {
        if(membersDatalistDiv.children[index].innerHTML == newMemberName) {
            newMemberID = membersDatalistDiv.children[index].value
            membersDatalistDiv.removeChild(membersDatalistDiv.children[index])
        }
    }

    if(!newMemberID || newMemberID == "") {
        alert("Invalid group")
        return;
    }

    var membersDatalistDiv = document.getElementById("userList")
    for (let index = 0; index < membersDatalistDiv.children.length; index++) {
        if(membersDatalistDiv.children[index].value == newMemberName) {
            membersDatalistDiv.removeChild(membersDatalistDiv.children[index])
        }
    }

    var membersDiv = document.getElementById("newMembers")
    var membersDivChildren = membersDiv.children

    for (let index = 0; index < membersDivChildren.length; index++) {
        var child = membersDivChildren[index]
        var childString = child.innerText
        if(childString.includes(newMemberName)) {
            return;
        }
    }

    var html = `
        <div class="group-member hoverable-opacity" title="Group" id="newMember-${newMemberID}">
            <input type="hidden" id="newMember-value" value="${newMemberID}">
            <input type="hidden" id="newMember-name" value="${newMemberName}">

            <div class="group-title">
                <div class="profile-icon icon-border icon-background" id="group_member_image_wrapper_${newMemberID}">
                    <img class="icon-img " src="/assets/users.svg" id="group_member_image_${newMemberID}">
                </div>

                <div class="group-title-text">
                    ${newMemberName}
                </div>
            </div>

            <div class="profile-icon clickable" onclick="removeUserFromSelection('${newMemberID}')" title="Remove group">
                <img class="icon-img " src="/assets/x.svg">
            </div>
        </div>
    `;

    membersDiv.innerHTML += html
    document.getElementById("newMemberID").value = ""
}

function addGroupsToWishlistThree(wishlistID, userID) {
    var selectedGroups = [];
    var newMembersDivChildren = document.getElementById("newMembers").children

    if(newMembersDivChildren.length == 0) {
        alert("No groups added :(");
        return;
    }

    for (var i=0; i < newMembersDivChildren.length; i++) {
        var newMemberID = newMembersDivChildren[i].id.replace("newMember-", "")
        selectedGroups.push(newMemberID)
    }

    var form_obj = { 
        "groups": selectedGroups
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
                showGroupsInWishlist(wishlistID, userID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlistID + "/join");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function showWishlistCollaboratorsInWishlist(wishlistID, userID){
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
                showWishlistCollaboratorsInWishlistTwo(result.wishlist, userID, wishlistID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists/" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function showWishlistCollaboratorsInWishlistTwo(wishlistObject, userID, wishlistID) {
    var html = '<div class="group-members" id="group_' + wishlistID + '_members" style="">'
    html += '<div class="text-body">Collaborators in this wishlist:</div>'
    var ownerID = wishlistObject.owner.id;

    if(wishlistObject.collaborators.length > 0) {
        for(var j = 0; j < wishlistObject.collaborators.length; j++) {
            html += '<div class="group-member hoverable-opacity" title="Group member">'

            html += '<div class="group-title">';

                html += `<div class="profile-icon icon-border icon-background" id="group_member_image_${wishlistObject.collaborators[j].user.id}_${wishlistID}">`
                html += '<img class="icon-img " src="/assets/user.svg">'
                html += '</div>'

                html += `
                    <div class="group-title-text">
                        ${wishlistObject.collaborators[j].user.first_name} ${wishlistObject.collaborators[j].user.last_name}
                    </div>
                `;

            html += '</div>'

            if(ownerID == userID && wishlistObject.collaborators[j].id !== userID) {
                html += `<div class="profile-icon clickable" onclick="removeWishlistCollaborator('${wishlistID}','${wishlistObject.collaborators[j].user.id}', '${userID}')" title="Remove collaborator">`
                html += '<img class="icon-img " src="/assets/x.svg">'
                html += '</div>'
            }

            html += '</div>'
        }
    } else {
        html += 'None :('
    }
    html += "</div>"

    if(ownerID == userID) {
        html += '<div id="wishlist-input" class="wishlist-input">';
        html += `<button id="register-button" onClick="addWishlistCollaborator('${wishlistID}', '${userID}');" type="" href="/">Add collaborators</button>`;
        html += '</div>';
    }

    toggleModal(html);

    for(var j = 0; j < wishlistObject.collaborators.length; j++) {
        getGroupMemberProfileImage(wishlistObject.collaborators[j].user.id, `group_member_image_${wishlistObject.collaborators[j].user.id}_${wishlistID}`)
    }
}

function removeWishlistCollaborator(wishlistID, collaboratorUserID, userID) {
    if(!confirm("Are you sure you want to remove this collaborator from your wishlist?")) {
        return;
    }

    var form_obj = { 
        "user_id" : collaboratorUserID
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
                showWishlistCollaboratorsInWishlist(wishlistID, userID);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlistID + "/un-collaborate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function addWishlistCollaborator(wishlistID, userID){
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
                addWishlistCollaboratorTwo(result.users, wishlistID, userID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users?notACollaboratorOfWishlistID=" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addWishlistCollaboratorTwo(usersArray, wishlistID, userID) {
    usersArray = addUniqueDisplayNames(usersArray)

    var userListHTML = '<datalist id="userList">'
    var userListIDHTML = '<datalist id="userIDList">'
    for (let index = 0; index < usersArray.length; index++) {
        if(usersArray[index].id == userID) {
            continue
        }

        const displayName = usersArray[index].displayName;
        const id = usersArray[index].id

        userListHTML += `<option value="${displayName}">${displayName}</option>`
        userListIDHTML += `<option value="${id}">${displayName}</option>`
    }
    userListHTML += '</datalist>'
    userListIDHTML += '</datalist>'

    var html = `
        <div class="profile-icon clickable top-left-button" onclick="showWishlistCollaboratorsInWishlist('${wishlistID}', '${userID}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <label for="newMemberMail" style="">Add collaborators:</label><br>
        <div class="addNewMemberButtons">
            <input type="text" name="newMemberMail" id="newMemberMail" list="userList" placeholder="Ola Nordmann" value="" autocomplete="off" style="margin: 1em 0.5em 1em 1em;" onkeydown="console.log('...')"/>
            <div class="profile-icon clickable" onclick="addUserToSelection()" title="Add member" style="margin: 0 1em 0 0;">
                <img class="icon-img" src="/assets/plus.svg">
            </div>
        </div>

        ${userListHTML}
        ${userListIDHTML}

        <div id="newMembers" class="newMembers">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); addWishlistCollaboratorThree('${wishlistID}', '${userID}');">   
            <button id="register-button" type="submit" href="/">Save</button>
        </form>
    `;

    toggleModal(html);
}

function addWishlistCollaboratorThree(wishlistID, userID) {
    var selectedMembers = [];
    var newMembersDivChildren = document.getElementById("newMembers").children

    if(newMembersDivChildren.length == 0) {
        alert("No members added :(");
        return;
    }

    for (var i=0; i < newMembersDivChildren.length; i++) {
        var newMemberID = newMembersDivChildren[i].id.replace("newMember-", "")
        selectedMembers.push(newMemberID)
    }

    var form_obj = { 
        "users": selectedMembers
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
                showWishlistCollaboratorsInWishlist(wishlistID, userID)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlists/" + wishlistID + "/collaborate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}