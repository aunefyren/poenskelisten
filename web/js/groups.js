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

        showAdminMenu(admin);

    } else {
        var login_data = false;
        group_id = 0;
        user_id = 0
        var email = ""
        var first_name = ""
        var last_name = ""
        admin = false;
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
                            Groups
                        </div>

                        <div class="text-body" style="text-align: center;">
                            These are groups you either own or are member of. Groups allow people to share wishlists between eachother. You can create a new group at the bottom of this page.
                        </div>

                    </div>

                    <div class="module">

                        <div id="groups-title" class="title">
                            Groups:
                        </div>

                        <div id="groups-box" class="groups">
                            <div class="loading-icon-wrapper" id="loading-icon-wrapper">
                                <img class="loading-icon" src="/assets/loading.svg">
                            </div>
                        </div>

                        <div id="group-input" class="group-input">
                            <button id="register-button" onClick="createGroup('${user_id}');" type="" href="/">Create new group</button>
                        </div>
                    </div>
                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Come together.';
    clearResponse();

    if(result !== false) {
        console.log("User: " + user_id)
        showLoggedInMenu();
        get_groups(user_id);
    } else {

        showLoggedOutMenu();
        invalid_session();
        
    }
}

function get_groups(user_id){

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

                console.log("Placing intial groups: ")
                placeGroups(groups, user_id);

            }

        } else {
            info("Loading groups...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/groups");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeGroups(group_array, user_id) {

    var html = ''

    for(var i = 0; i < group_array.length; i++) {

        var owner_id = group_array[i].owner.id

        console.log("Owner:" + owner_id)

        html += `<div class="group-wrapper" id="groupWrapper-${group_array[i].id}">`

        html += '<div class="group">'
        
        html += `<div class="group-title clickable underline" style="" onclick="location.href = '/groups/${group_array[i].id}'" title="Go to group">`;
        html += '<div class="profile-icon">'
        html += '<img class="icon-img " src="/assets/users.svg">'
        html += `</div>`

        html += `<div id="groupName-${group_array[i].id}">`
        html += group_array[i].name
        html += '</div>'

        html += '</div>'

        html += '<div class="profile">'

        var members_string="["
        for(var j = 0; j < group_array[i].members.length; j++) {
            if(j !== 0) {
                members_string += ','
            }
            members_string += "'" + group_array[i].members[j].id + "'"
        }
        members_string += ']'

        html += `<div class="profile-icon clickable" onclick="groupMembers('${group_array[i].id}', '${user_id}')" title="Configure members">`;
        html += '<img class="icon-img " src="/assets/user.svg">'
        html += '</div>'

        html += `<div class="profile-icon clickable" onclick="showWishlistsInGroup('${group_array[i].id}', '${user_id}')" title="Configure wishlists">`;
        html += '<img class="icon-img " src="/assets/list.svg">'
        html += '</div>'

        if(owner_id == user_id) {
            html += `<div class="profile-icon clickable" onclick="editGroup('${user_id}', '${group_array[i].id}')" title="Edit group">`;
            html += '<img class="icon-img " src="/assets/edit.svg">'
            html += '</div>'

            html += `<div class="profile-icon clickable" onclick="deleteGroup('${group_array[i].id}', '${user_id}')" title="Delete group">`;
            html += '<img class="icon-img " src="/assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'
        

        html += '</div>'

        html += '</div>'

    }

    if(group_array.length == 0) {
        info("Looks like this list is empty... Maybe the group owner needs to add you to a group?");

        try {
            document.getElementById("loading-icon-wrapper").style.display = "none"
        } catch(e) {
            console.log("Error: " + e)
        }
    }

    group_object = document.getElementById("groups-box")
    group_object.innerHTML = html
}

function toggle_group(group_id, owner_id, user_id, member_array) {
    
    group_members = document.getElementById("group_" + group_id + "_members");
    group_members_arrow = document.getElementById("group_" + group_id + "_arrow");

    console.log(member_array);

    if(group_members.classList.contains("collapsed")) {
        group_members.classList.remove("collapsed")
        group_members.classList.add("expanded")
        group_members.style.display = "inline-block"
        group_members_arrow.src = "/assets/chevron-down.svg"

        if(user_id == owner_id) {
            get_users_group(group_id, owner_id, user_id, member_array)
        }
    } else {
        group_members.classList.remove("expanded")
        group_members.classList.add("collapsed")
        group_members.style.display = "none"
        group_members_arrow.src = "/assets/chevron-right.svg"

        if(user_id == owner_id) {
            var select_list = document.getElementById("group-input-members-" + group_id)
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

function get_users_group(group_id, owner_id, user_id, member_array){

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

                users = result.users;
                place_users_groups(users, group_id, owner_id, user_id, member_array);

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

function place_users_groups(user_array, group_id, owner_id, user_id, member_array) {
    var select_list = document.getElementById("group-input-members-" + group_id)

    for(var i = 0; i < user_array.length; i++) {

        if(user_array[i].id == user_id) {
            continue;
        } else {
            var found = false;
            for(var j = 0; j < user_array.length; j++) {
                if(member_array[j] == user_array[i].id) {
                    found = true;
                    break;
                }
            }
            if(found) {
                continue;
            }
        }

        var option = document.createElement("option");
        option.value = user_array[i].id
        option.text = user_array[i].first_name + " " + user_array[i].last_name
        select_list.add(option, select_list[0]);
    }
}

function placeGroup(group_object) {
    document.getElementById("groupName-" + group_object.id).innerHTML = group_object.name
}

function removeGroup(groupID, userID) {
    document.getElementById(`groupWrapper-${groupID}`).remove();
}