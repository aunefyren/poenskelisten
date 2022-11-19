function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
    } else {
        var login_data = false;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Pønskelisten
                        </div>

                        <div class="body" style="text-align: center;">
                            Choose a group, yo.
                        </div>

                        <br>
                        <br>

                        <div id="groups-box" class="groups">
                        </div>

                        <div id="group-input" class="group-input">
                            <form action="" onsubmit="event.preventDefault(); create_group(` + login_data.data.ID + `);">
                                <input type="text" name="group_name" id="group_name" placeholder="Group name" autocomplete="off" required />
                                
                                <input type="text" name="group_description" id="group_description" placeholder="Group description" autocomplete="off" required />


                                <label for="group_members">Group members:</label><br>
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
        showLoggedInMenu();
        get_groups(login_data.data.id);
        get_users(login_data.data.id);
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
                place_groups(groups, user_id);

            }

        } else {
            info("Loading groups...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/group/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_groups(group_array, user_id) {

    var html = ''

    for(var i = 0; i < group_array.length; i++) {

        html += '<div class="group-wrapper">'

        html += '<div class="group">'
        
        html += '<div class="group-title clickable" onclick="location.href = \'./groups/' + group_array[i].ID + '\'">'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/users.svg">'
        html += '</div>'
        html += group_array[i].name
        html += '</div>'

        html += '<div class="profile">'

        if(group_array[i].members.length > 0) {
            html += '<div class="profile-icon clickable" onclick="toggle_group(' + group_array[i].ID + ')">'
            html += '<img id="group_' + group_array[i].ID + '_arrow" class="icon-img color-invert" src="../../assets/arrow-down.svg">'
            html += '</div>'
        }

        if(group_array[i].owner.ID = user_id) {
            html += '<div class="profile-icon clickable" onclick="add_member(' + group_array[i].ID + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/plus.svg">'
            html += '</div>'

            html += '<div class="profile-icon clickable" onclick="delete_wish(' + group_array[i].ID + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'

        html += '<div class="group-members collapsed" id="group_' + group_array[i].ID + '_members">'
        for(var j = 0; j < group_array[i].members.length; j++) {
            html += '<div class="group-member">'

            html += '<div class="group-title">';

            html += '<div class="profile-icon">'
            html += '<img class="icon-img color-invert" src="../assets/user.svg">'
            html += '</div>'

            html += group_array[i].members[j].first_name + " " + group_array[i].members[j].last_name

            html += '</div>'

            if(group_array[i].owner.ID = user_id && group_array[i].members[j].ID !== user_id) {
                html += '<div class="profile-icon clickable" onclick="remove_member(' + group_array[i].ID + ',' + group_array[i].members[j].ID + ')">'
                html += '<img class="icon-img color-invert" src="../../assets/x.svg">'
                html += '</div>'
            }

            html += '</div>'
        }
        html += '</div>'

        html += '</div>'

    }

    group_object = document.getElementById("groups-box")
    group_object.innerHTML = html
}

function toggle_group(group_id) {
    group_members = document.getElementById("group_" + group_id + "_members");
    group_members_arrow = document.getElementById("group_" + group_id + "_arrow");

    if(group_members.classList.contains("collapsed")) {
        group_members.classList.remove("collapsed")
        group_members.classList.add("expanded")
        group_members.style.display = "inline-block"
        group_members_arrow.src = "../../assets/arrow-right.svg"
    } else {
        group_members.classList.remove("expanded")
        group_members.classList.add("collapsed")
        group_members.style.display = "none"
        group_members_arrow.src = "../../assets/arrow-down.svg"
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

                clearResponse();
                users = result.users;
                console.log(users);
                place_users(users, user_id);

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/user/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_users(user_array, user_id) {
    var select_list = document.getElementById("group-input-members")

    for(var i = 0; i < user_array.length; i++) {

        if(user_array[i].ID == user_id) {
            continue;
        }

        var option = document.createElement("option");
        option.value = user_array[i].ID
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
            selected_members.push(Number(opt.value));
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

                groups = result.groups;
                place_groups(groups, user_id);
                clear_data();
                
            }

        } else {
            info("Saving group...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/group/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function clear_data() {
    document.getElementById("group_name").value = "";
    document.getElementById("group_description").value = "";
}