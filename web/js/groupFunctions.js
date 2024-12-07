function editGroup(user_id, group_id){
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
                editGroupTwo(user_id, group_id, result.group)
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

function editGroupTwo(user_id, group_id, groupObject) {
    var html = '';

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); updateGroup('${group_id}', '${user_id}');">
                                
            <label for="group_name" style="">Edit group:</label><br>
            <input type="text" name="group_name" id="group_name" placeholder="Group name" value="${groupObject.name}" autocomplete="off" required />
            
            <textarea type="text" name="group_description" id="group_description" placeholder="Group description" value="${groupObject.description}" autocomplete="off" required >${groupObject.description}</textarea>

            <button id="register-button" type="submit" href="/">Save group</button>

        </form>
    `;

    toggleModal(html);
}

function updateGroup(group_id, user_id) {
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
            
            toggleModal();
            if(result.error) {
                error(result.error);
            } else {
                success(result.message);
                placeGroup(result.group, true);                
            }

        } else {
            info("Updating group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function groupMembers(groupID, userID){
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
                groupMembersTwo(result.group, userID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/groups/" + groupID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function groupMembersTwo(groupObject, userID) {
    var html = '<div class="group-members" id="group_' + groupObject.id + '_members" style="">'
    html += '<div class="text-body">Members in this group:</div>'
    var ownerID = groupObject.owner.id;

    for(var j = 0; j < groupObject.members.length; j++) {
        html += '<div class="group-member hoverable-opacity" title="Group member">'

        html += '<div class="group-title">';

        html += `<div class="profile-icon icon-border icon-background" id="group_member_image_${groupObject.members[j].id}_${groupObject.id}">`
        html += '<img class="icon-img " src="/assets/user.svg">'
        html += '</div>'

        html += groupObject.members[j].first_name + " " + groupObject.members[j].last_name

        html += '</div>'

        if(ownerID == userID && groupObject.members[j].id !== userID) {
            html += `<div class="profile-icon clickable" onclick="removeGroupMember('${groupObject.id}','${groupObject.members[j].id}', '${userID}')" title="Remove member">`
            html += '<img class="icon-img " src="/assets/x.svg">'
            html += '</div>'
        } else if(groupObject.members[j].id == userID && ownerID !== userID){
            html += `<div class="profile-icon clickable" onclick="leaveGroup('${groupObject.id}','${userID}')" title="Leave group">`;
            html += '<img class="icon-img " src="/assets/log-out.svg">'
            html += '</div>'
        } else if(groupObject.members[j].id == ownerID) {
            html += '<div class="profile-icon" title="Group owner">'
            html += '<img class="icon-img " src="/assets/star.svg">'
            html += '</div>'
        }

        html += '</div>'
    }
    html += "</div>"

    if(ownerID == userID) {
        html += '<div id="wishlist-input" class="wishlist-input">';
        html += `<button id="register-button" onClick="getUsersForGroupMembers('${groupObject.id}', '${userID}');" type="" href="/">Add members</button>`;
        html += '</div>';
    }

    toggleModal(html);

    for(var j = 0; j < groupObject.members.length; j++) {
        getGroupMemberProfileImage(groupObject.members[j].id, `group_member_image_${groupObject.members[j].id}_${groupObject.id}`)
    }
}

function getGroupMemberProfileImage(userID, divID) {
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
                    placeGroupMemberProfileImage(result.image, divID)
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

function placeGroupMemberProfileImage(imageBase64, divID) {
    var image = document.getElementById(divID)
    image.style.backgroundSize = "cover"
    image.innerHTML = ""
    image.style.backgroundImage = `url('${imageBase64}')`
    image.style.backgroundPosition = "center center"
}

function getUsersForGroupMembers(groupID, userID){
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
                getUsersForGroupMembersTwo(result.users, groupID, userID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users?notAMemberOfGroupID=" + groupID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function getUsersForGroupMembersTwo(usersArray, groupID, userID) {
    var userListHTML = '<datalist id="userList">'
    for (let index = 0; index < usersArray.length; index++) {
        const displayName = usersArray[index].first_name + " " + usersArray[index].last_name;
        const email = usersArray[index].email

        userListHTML += `<option value="${email}">${displayName}</option>`
    }
    userListHTML += '</datalist>'

    var html = `
        <div class="profile-icon clickable top-left-button" onclick="groupMembers('${groupID}', '${userID}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <label for="newMemberMail" style="">Add members:</label><br>
        <div class="addNewMemberButtons">
            <input type="text" name="newMemberMail" id="newMemberMail" list="userList" placeholder="Ola Nordmann" value="" autocomplete="off" style="margin: 1em 0.5em 1em 1em;" />
            <div class="profile-icon clickable" onclick="addUserToSelection()" title="Add member" style="margin: 0 1em 0 0;">
                <img class="icon-img" src="/assets/plus.svg">
            </div>
        </div>

        ${userListHTML}

        <div id="newMembers" class="newMembers">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); addGroupMembers('${groupID}', '${userID}');">   
            <button id="register-button" type="submit" href="/">Save</button>
        </form>
    `;

    toggleModal(html);
}

function addUserToSelection() {
    var newMemberMail = document.getElementById("newMemberMail").value
    if(!newMemberMail || newMemberMail == "") {
        return;
    }

    var html = `
        <div class="group-member hoverable-opacity" title="Group member" id="newMember-${newMemberMail}">
            <div class="group-title">
                <div class="profile-icon icon-border icon-background" id="group_member_image_">
                    <img class="icon-img " src="/assets/user.svg">
                </div>

                ${newMemberMail}
            </div>

            <div class="profile-icon clickable" onclick="removeUserFromSelection('${newMemberMail}')" title="Remove member">
                <img class="icon-img " src="/assets/x.svg">
            </div>
        </div>
    `;

    var membersDiv = document.getElementById("newMembers")
    var membersDivChildren = membersDiv.children

    for (let index = 0; index < membersDivChildren.length; index++) {
        var child = membersDivChildren[index]
        var childString = child.innerText
        if(childString.includes(newMemberMail)) {
            return;
        }
    }

    var membersDatalistDiv = document.getElementById("userList")
    for (let index = 0; index < membersDatalistDiv.children.length; index++) {
        if(membersDatalistDiv.children[index].value == newMemberMail) {
            membersDatalistDiv.removeChild(membersDatalistDiv.children[index])
        }
    }

    membersDiv.innerHTML += html
    document.getElementById("newMemberMail").value = ""
}

function removeUserFromSelection(userMail) {
    var membersDiv = document.getElementById("newMembers")
    var membersDivChildren = membersDiv.children

    for (let index = 0; index < membersDivChildren.length; index++) {
        var child = membersDivChildren[index]
        var childString = child.innerText
        if(childString.includes(userMail)) {
            child.remove();
        }
    }
}

function addGroupMembers(groupID, userID) {
    var selectedMembers = [];
    var newMembersDivChildren = document.getElementById("newMembers").children

    if(newMembersDivChildren.length == 0) {
        alert("No members added :(");
        return;
    }

    for (var i=0; i < newMembersDivChildren.length; i++) {
        var newMemberMail = newMembersDivChildren[i].id.replace("newMember-", "")
        selectedMembers.push(newMemberMail)
    }

    var form_obj = { 
        "members": selectedMembers
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
                groupMembers(groupID, userID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + groupID + "/join");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function removeGroupMember(group_id, member_id, user_id) {
    if(!confirm("Are you sure you want to remove this member?")) {
        return;
    }

    var form_obj = { 
        "member_id" : member_id
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
                groupMembers(group_id, user_id);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + group_id + "/remove");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function leaveGroup(group_id, user_id) {
    if(!confirm("Are you sure you want to leave this group?")) {
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
                window.location.href = "/groups";
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + group_id + "/leave");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function deleteGroup(group_id, user_id) {
    if(!confirm("Are you sure you want to delete this group?")) {
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
                window.location.href = "/groups"
            }
        } else {
            info("Deleting group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/groups/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function createGroup(userID) {
    var html = '';

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); createGroupTwo('${userID}');">  
            <label for="group_name" style="">Create a new group:</label><br>

            <input type="text" name="group_name" id="group_name" placeholder="Group name" autocomplete="off" min="5" required />
            
            <textarea type="text" name="group_description" id="group_description" placeholder="Group description" min="5" autocomplete="off" required /></textarea>

            <div id="newMembers" class="newMembers">
            </div>
            
            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createGroupTwo(userID){
    var groupName = document.getElementById("group_name").value;
    var groupDescription = document.getElementById("group_description").value;

    if(groupName.length < 5 || groupDescription.length < 5) {
        alert("Name and description must be five or more characters.")
        return
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
                createGroupThree(result.users, userID, groupName, groupDescription)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function createGroupThree(usersArray, userID, groupName, groupDescription) {
    var userListHTML = '<datalist id="userList">'
    for (let index = 0; index < usersArray.length; index++) {
        const displayName = usersArray[index].first_name + " " + usersArray[index].last_name;
        const email = usersArray[index].email

        userListHTML += `<option value="${email}">${displayName}</option>`
    }
    userListHTML += '</datalist>'

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="createGroup('${userID}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <label for="newMemberMail" style="">Add group members:</label><br>
        <div class="addNewMemberButtons">
            <input type="text" name="newMemberMail" id="newMemberMail" list="userList" placeholder="Ola Nordmann" value="" autocomplete="off" style="margin: 1em 0.5em 1em 1em;" />
            <div class="profile-icon clickable" onclick="addUserToSelection()" title="Add member" style="margin: 0 1em 0 0;">
                <img class="icon-img" src="/assets/plus.svg">
            </div>
        </div>

        ${userListHTML}

        <div id="newMembers" class="newMembers">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); createGroupFour('${userID}');">
            <input type="hidden" id="group_name" value="${groupName}">
            <input type="hidden" id="group_description" value="${groupDescription}">

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createGroupFour(userID){
    var groupName = document.getElementById("group_name").value;
    var groupDescription = document.getElementById("group_description").value;
    var newMembersDivChildren = document.getElementById("newMembers").children
    var selectedMembers = [];

    for (var i=0; i < newMembersDivChildren.length; i++) {
        var newMemberMail = newMembersDivChildren[i].id.replace("newMember-", "")
        selectedMembers.push(newMemberMail)
    }

    var selectedMembersBase64 = toBASE64(JSON.stringify(selectedMembers))

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
                createGroupFive(result.wishlists, userID, groupName, groupDescription, selectedMembersBase64)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function createGroupFive(wishlistArray, userID, groupName, groupDescription, selectedMembersBase64) {
    var wishlistListHTML = '<datalist id="userList">'
    for (let index = 0; index < wishlistArray.length; index++) {
        const displayName = wishlistArray[index].name;
        const id = wishlistArray[index].id

        wishlistListHTML += `<option value="${id}">${displayName}</option>`
    }
    wishlistListHTML += '</datalist>'

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="createGroupTwo('${userID}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <label for="newMemberID" style="">Add your wishlists to your new group:</label><br>
        <div class="addNewMemberButtons">
            <input type="text" name="newMemberID" id="newMemberID" list="userList" placeholder="My cool wishlist" value="" autocomplete="off" style="margin: 1em 0.5em 1em 1em;" />
            <div class="profile-icon clickable" onclick="addWishlistToSelection()" title="Add member" style="margin: 0 1em 0 0;">
                <img class="icon-img" src="/assets/plus.svg">
            </div>
        </div>

        ${wishlistListHTML}

        <div id="newMembers" class="newMembers">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); createGroupSix('${userID}');">
            <input type="hidden" id="group_name" value="${groupName}">
            <input type="hidden" id="group_description" value="${groupDescription}">
            <input type="hidden" id="group_members" value="${selectedMembersBase64}">

            <button id="register-button" type="submit" href="/">Create group</button>
        </form>
    `;

    toggleModal(html);
}


function createGroupSix(userID) {
    var groupName = document.getElementById("group_name").value;
    var groupDescription = document.getElementById("group_description").value;
    var groupMembersBase64 = document.getElementById("group_members").value;
    var groupMembers = JSON.parse(fromBASE64(groupMembersBase64))
    var newMembersDivChildren = document.getElementById("newMembers").children
    var selectedWishlists = [];

    for (var i=0; i < newMembersDivChildren.length; i++) {
        var newMemberMail = newMembersDivChildren[i].id.replace("newMember-", "")
        selectedWishlists.push(newMemberMail)
    }

    var form_obj = { 
        "name" : groupName,
        "description" : groupDescription,
        "members": groupMembers,
        "wishlists": selectedWishlists
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
                groups = result.groups;
                placeGroups(groups, userID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function addWishlistToSelection() {
    var newMemberID = document.getElementById("newMemberID").value
    if(!newMemberID || newMemberID == "") {
        return;
    }

    var html = `
        <div class="group-member hoverable-opacity" title="Group member" id="newMember-${newMemberID}">
            <div class="group-title">
                <div class="profile-icon icon-background" id="group_member_image_">
                    <img class="icon-img " src="/assets/list.svg">
                </div>

                ${newMemberID}
            </div>

            <div class="profile-icon clickable" onclick="removeUserFromSelection('${newMemberID}')" title="Remove wishlist">
                <img class="icon-img " src="/assets/x.svg">
            </div>
        </div>
    `;

    var membersDiv = document.getElementById("newMembers")
    var membersDivChildren = membersDiv.children

    for (let index = 0; index < membersDivChildren.length; index++) {
        var child = membersDivChildren[index]
        var childString = child.innerText
        if(childString.includes(newMemberID)) {
            return;
        }
    }

    var membersDatalistDiv = document.getElementById("userList")
    for (let index = 0; index < membersDatalistDiv.children.length; index++) {
        if(membersDatalistDiv.children[index].value == newMemberID) {
            membersDatalistDiv.removeChild(membersDatalistDiv.children[index])
        }
    }

    membersDiv.innerHTML += html
    document.getElementById("newMemberID").value = ""
}

function showWishlistsInGroup(groupID, userID){
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
                showWishlistsInGroupTwo(result.wishlists, userID, groupID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists?owned=true&group=" + groupID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function showWishlistsInGroupTwo(wishlistArray, userID, groupID) {
    var html = '<div class="group-members" id="group_' + groupID + '_members" style="">'
    html += '<div class="text-body">The wishlists present in this group:</div>'

    for(var j = 0; j < wishlistArray.length; j++) {
        html += `
            <div class="group-member hoverable-opacity" title="Wishlist">
                <div class="group-title">
                    <div class="profile-icon" id="group_member_image_${wishlistArray[j].id}">
                        <img class="icon-img " src="/assets/list.svg">
                    </div>

                    ${wishlistArray[j].name}

                </div>
                <div class="profile-icon clickable" onclick="removeWishlistFromGroup('${wishlistArray[j].id}', '${groupID}', '${userID}')" title="Remove wishlist from group">
                    <img class="icon-img " src="/assets/x.svg">
                </div>
            </div>
        `;
    }
    html += `
        <div id="wishlist-input" class="wishlist-input">
            <button id="register-button" onClick="addWishlistToGroup('${groupID}', '${userID}');" type="" href="/">Add wishlist to group</button>
        </div>
    `;

    toggleModal(html);
}

function addWishlistToGroup(groupID, userID){
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
                addWishlistToGroupTwo(result.wishlists, userID, groupID)
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists?notAMemberOfGroupID=" + groupID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addWishlistToGroupTwo(wishlistArray, userID, groupID) {
    var wishlistListHTML = '<datalist id="userList">'
    for (let index = 0; index < wishlistArray.length; index++) {
        const displayName = wishlistArray[index].name;
        const id = wishlistArray[index].id

        wishlistListHTML += `<option value="${id}">${displayName}</option>`
    }
    wishlistListHTML += '</datalist>'

    var html = '';

    html += `
        <div class="profile-icon clickable top-left-button" onclick="showWishlistsInGroup('${groupID}', '${userID}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <label for="newMemberID" style="">Add your wishlists to the group:</label><br>
        <div class="addNewMemberButtons">
            <input type="text" name="newMemberID" id="newMemberID" list="userList" placeholder="My cool wishlist" value="" autocomplete="off" style="margin: 1em 0.5em 1em 1em;" />
            <div class="profile-icon clickable" onclick="addWishlistToSelection()" title="Add member" style="margin: 0 1em 0 0;">
                <img class="icon-img" src="/assets/plus.svg">
            </div>
        </div>

        ${wishlistListHTML}

        <div id="newMembers" class="newMembers">
        </div>

        <form action="" class="" onsubmit="event.preventDefault(); addWishlistToGroupThree('${groupID}', '${userID}');">
            <button id="register-button" type="submit" href="/">Add to group</button>
        </form>
    `;

    toggleModal(html);
}

function addWishlistToGroupThree(groupID, userID) {
    var newMembersDivChildren = document.getElementById("newMembers").children
    var selectedWishlists = [];

    for (var i=0; i < newMembersDivChildren.length; i++) {
        var newMemberMail = newMembersDivChildren[i].id.replace("newMember-", "")
        selectedWishlists.push(newMemberMail)
    }

    if(selectedWishlists.length < 1) {
        alert("You must provide one or more wishlists.")
        return;
    }

    var form_obj = { 
        "wishlists": selectedWishlists
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
                showWishlistsInGroup(groupID, userID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + groupID + "/add");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function removeWishlistFromGroup(wishlistID, groupID, userID) {
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
                showWishlistsInGroup(groupID, userID);
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