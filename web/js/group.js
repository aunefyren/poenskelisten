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
                            Choose a wishlist, yo.
                        </div>

                        <br>
                        <br>

                        <div id="wishlists-box" class="wishlists">
                        </div>
      
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Lists...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        string_index = document.URL.lastIndexOf('/');
        group_id = document.URL.substring(string_index+1);
        console.log(group_id);
        get_wishlists(group_id, login_data.data.id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_wishlists(group_id, user_id){

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
                console.log(wishlists);
                place_wishlists(wishlists, group_id, user_id);

            }

        } else {
            info("Loading wishlists...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishlist/get/group/" + group_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_wishlists(wishlists_array, group_id, user_id) {

    var html = ''

    for(var i = 0; i < wishlists_array.length; i++) {

        html += '<div class="wishlist-wrapper">'

        html += '<div class="wishlist">'
        
        html += '<div class="wishlist-title clickable" onclick="location.href = \'./' + group_id + "/" + wishlists_array[i].ID + '\'">'
        html += wishlists_array[i].name
        html += '</div>'

        html += '<div class="profile">'
        html += '<div class="profile-name">'
        html += wishlists_array[i].owner.first_name + " " + wishlists_array[i].owner.last_name
        html += '</div>'
        html += '<div class="profile-icon">'
        html += '<img class="icon-img color-invert" src="../assets/user.svg">'
        html += '</div>'

        if(wishlists_array[i].owner.ID = user_id) {
            html += '<div class="profile-icon clickable" onclick="delete_wish(' + wishlists_array[i].ID + ')">'
            html += '<img class="icon-img color-invert" src="../../assets/trash-2.svg">'
            html += '</div>'
        }

        html += '</div>'

        html += '</div>'

        html += '</div>'
    }

    wishlist_object = document.getElementById("wishlists-box")
    wishlist_object.innerHTML = html
}