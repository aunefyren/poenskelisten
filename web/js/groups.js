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
                            <form action="" class="icon-border" onsubmit="event.preventDefault(); create_group('${user_id}');">
                                
                                <label for="group_name">Create a new group:</label><br>

                                <input type="text" name="group_name" id="group_name" placeholder="Group name" autocomplete="off" required />
                                
                                <input type="text" name="group_description" id="group_description" placeholder="Group description" autocomplete="off" required />

                                <label for="group_members">Select group members:</label><br>
                                <select name="group_members" id="group-input-members" multiple>
                                </select>
                                
                                <button id="register-button" type="submit" href="/">Create group</button>

                            </form>
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
        get_users(user_id);

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
                place_groups(groups, user_id);

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

function place_groups(group_array, user_id) {

    var html = ''

    for(var i = 0; i < group_array.length; i++) {

        var owner_id = group_array[i].owner.id

        console.log("Owner:" + owner_id)

        html += '<div class="group-wrapper">'

        html += '<div class="group">'
        
        html += `<div class="group-title clickable underline" style="margin: 0.5em auto;" onclick="location.href = '/groups/${group_array[i].id}'" title="Go to group">`;
        html += '<div class="profile-icon">'
        html += '<img class="icon-img " src="/assets/users.svg">'
        html += '</div>'
        html += group_array[i].name
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

        if(group_array[i].members.length > 0) {
            html += `<div class="profile-icon clickable" onclick="toggle_group('${group_array[i].id}', '${group_array[i].owner.id}', '${user_id}', ${members_string})" title="Expandable">`;
            html += '<img id="group_' + group_array[i].id + '_arrow" class="icon-img " src="/assets/chevron-right.svg">'
            html += '</div>'
        }

        if(owner_id == user_id) {
            html += `<div class="profile-icon clickable" onclick="delete_group('${group_array[i].id}', '${user_id}')" title="Delete group">`;
            html += '<img class="icon-img " src="/assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'
        
        html += '<div class="group-members collapsed" id="group_' + group_array[i].id + '_members">'
        for(var j = 0; j < group_array[i].members.length; j++) {
            if(j == 0) {
                html += '<div class="text-body">Members in this group:</div>'
            }
            html += '<div class="group-member hoverable-opacity" title="Group member">'

            html += '<div class="group-title">';

            html += `<div class="profile-icon icon-border icon-background" id="group_member_image_${group_array[i].members[j].id}_${group_array[i].id}">`
            html += '<img class="icon-img " src="/assets/user.svg">'
            html += '</div>'

            html += group_array[i].members[j].first_name + " " + group_array[i].members[j].last_name

            html += '</div>'

            if(owner_id == user_id && group_array[i].members[j].id !== user_id) {
                html += `<div class="profile-icon clickable" onclick="remove_member('${group_array[i].id}','${group_array[i].members[j].id}', '${user_id}')" title="Remove member">`
                html += '<img class="icon-img " src="/assets/x.svg">'
                html += '</div>'
            } else if(group_array[i].members[j].id == user_id && owner_id !== user_id){
                html += `<div class="profile-icon clickable" onclick="leave_group('${group_array[i].id}','${user_id}')" title="Leave group">`;
                html += '<img class="icon-img " src="/assets/log-out.svg">'
                html += '</div>'
            } else if(group_array[i].members[j].id == owner_id) {
                html += '<div class="profile-icon" title="Group owner">'
                html += '<img class="icon-img " src="/assets/star.svg">'
                html += '</div>'
            }

            html += '</div>'
        }

        if(owner_id == user_id) {
            html += '<hr style="margin: 1.75em 0.5em;">'
            html += `<form action="" class="" onsubmit="event.preventDefault(); add_members('${group_array[i].id}', '${user_id}');">`;
            html += '<label for="group_members_' + group_array[i].id + '">Select new group members:</label><br>';
            html += '<select name="group_members_' + group_array[i].id + '" id="group-input-members-' + group_array[i].id + '" multiple>';
            html += '</select>';
            html += '<button id="register-button" type="submit" href="/">Add members to group</button>';
            html += '</form>';
        }
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

    for(var i = 0; i < group_array.length; i++) {
        for(var j = 0; j < group_array[i].members.length; j++) {
            GetProfileImage(group_array[i].members[j].id, `group_member_image_${group_array[i].members[j].id}_${group_array[i].id}`)
        }
    }
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

function get_users(user_id){

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
                place_users(users, user_id);

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

function place_users(user_array, user_id) {
    var select_list = document.getElementById("group-input-members")

    for(var i = 0; i < user_array.length; i++) {

        if(user_array[i].id == user_id) {
            continue;
        }

        var option = document.createElement("option");
        option.value = user_array[i].id
        option.text = user_array[i].first_name + " " + user_array[i].last_name
        select_list.add(option, select_list[0]);
    }
}

function create_group(user_id) {
    var selected_members = [];
    var select_list = document.getElementById("group-input-members")

    for (var i=0; i < select_list.options.length; i++) {
        opt = select_list.options[i];
    
        if (opt.selected) {
            selected_members.push(opt.value);
        }
    }

    var group_name = document.getElementById("group_name").value;
    var group_description = document.getElementById("group_description").value;

    var form_obj = { 
        "name" : group_name,
        "description" : group_description,
        "members": selected_members
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

                groups = result.groups;

                console.log("Placing groups after new is created: ")
                place_groups(groups, user_id);

                clear_data();
                
            }

        } else {
            info("Saving group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function clear_data() {
    document.getElementById("group_name").value = "";
    document.getElementById("group_description").value = "";
}

function delete_group(group_id, user_id) {

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

                success(result.message);
                console.log(result);

                console.log("User ID: " + user_id);

                groups = result.groups;

                console.log("Placing groups after one is deleted: ")
                place_groups(groups, user_id);
                
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

function remove_member(group_id, member_id, user_id) {

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

                success(result.message);
                console.log(result);

                console.log("User ID: " + user_id);

                groups = result.groups;

                console.log("Placing groups after member is removed: ")
                place_groups(groups, user_id);
                
            }

        } else {
            info("Removing member...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + group_id + "/remove");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function add_members(group_id, user_id) {

    var selected_members = [];
    var select_list = document.getElementById("group-input-members-" + group_id)

    for (var i=0; i < select_list.options.length; i++) {
        opt = select_list.options[i];
    
        if (opt.selected) {
            selected_members.push(opt.value);
        }
    }

    var form_obj = { 
        "members": selected_members
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

                console.log("User ID: " + user_id);

                groups = result.groups;

                console.log("Placing groups after member is added: ")
                place_groups(groups, user_id);
                
            }

        } else {
            info("Adding members...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + group_id + "/join");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function leave_group(group_id, user_id) {
    
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

                success(result.message);
                console.log(result);

                console.log("User ID: " + user_id);

                groups = result.groups;

                console.log("Placing groups after member is removed: ")
                place_groups(groups, user_id);
                
            }

        } else {
            info("Leaving group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/groups/" + group_id + "/leave");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

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