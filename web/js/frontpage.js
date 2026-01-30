function load_page(result) {

    if(result !== false) {

        try {

            var login_data = JSON.parse(result);

            if(login_data.error === "You must verify your account.") {
                verifyRedirect();
                return;
            }

            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            admin = login_data.data.admin;
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            admin = false;
        }

        showAdminMenu(admin)

    } else {
        var login_data = false;
        var email = ""
        var first_name = ""
        var last_name = ""
        admin = false;
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
                    
                        <div class="title">
                            {{.appName}}
                        </div>

                        <div class="text-body" id="main-text" style="text-align: center;">
                            Make a wish.

                            <br>
                            <br>

                            Welcome to the front page. Use to navigation bar and head to Wishlists to manage your wishlists. Head to Groups to manage and view wishlists in groups.

                            <div class="labels">
                                <div class="blue-label clickable" onclick="location.href='/wishlists'">My Wishlists</div>
                                <div class="blue-label clickable" onclick="location.href='/groups'">Groups</div>
                            </div>

                            <div id="wishlists-front-page">
                                <div id="wishlists-title" class="title-two">
                                    Recently updated wishlists:
                                </div>

                                <div id="wishlists-box" class="wishlists-minimal">
                                    <div class="loading-icon-wrapper" id="loading-icon-wrapper">
                                        <img class="loading-icon" src="/assets/loading.svg">
                                    </div>
                                </div>
                            </div>

                        </div>

                        <div id="log-in-button" style="margin-top: 2em; display: none; width: 10em;">
                            <button id="update-button" type="submit" href="#" onclick="window.location = './login';">Log in</button>
                        </div>

                    </div>

                    <hr id="module-split"></hr>

                    <div class="module">

                        <div id="news-title" class="title" style="display: none;">
                            News:
                        </div>

                        <div class="loading-icon-wrapper" id="loading-icon-wrapper-news">
                            <img class="loading-icon" src="/assets/loading.svg">
                        </div>

                        <div id="news-box" class="news">
                        </div>
                        
                    </div>

                    <div class="module" id="new-news" style="display: none;">
                        <button id="register-button" onClick="createNewsPost(${admin});" type="" href="/">Create news post</button>
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {

        showLoggedInMenu();
        get_news(admin);
        getWishlists();

        if(admin) {
            document.getElementById('new-news').style.display = "flex";
        }

    } else {
        showLoggedOutMenu();
        document.getElementById('main-text').innerHTML = "You need to login.";
        document.getElementById('log-in-button').style.display = 'inline-block';

        try {
            document.getElementById("loading-icon-wrapper-news").style.display = "none"
        } catch(e) {
            console.log("Error: " + e)
        }

        try {
            document.getElementById("module-split").style.display = "none"
        } catch(e) {
            console.log("Error: " + e)
        }
    }
}

function get_news(admin){
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
                news = result.news;

                console.log(news);

                console.log("Placing intial news: ")
                placeNewsPosts(news, admin);

            }

        } else {
            info("Loading news...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/news");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeNewsPosts(news_array, admin) {
    try {
        document.getElementById("loading-icon-wrapper-news").style.display = "none"
    } catch(e) {
        console.log("Error: " + e)
    }

    if(news_array.length == 0) {
        return;
    } else {
        document.getElementById("news-title").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < news_array.length; i++) {
        html += buildNewsPostHTML(news_array[i], admin)
    }

    news_object = document.getElementById("news-box")
    news_object.innerHTML = html
}

function removeNewsPost(newsPostID) {
    document.getElementById(`newsPost-${newsPostID}`).remove()
}

function verifyRedirect() {
    window.location = '/verify'
}

function placeNewsPost(newsPostObject, adminStatus) {
    try {
        var now = new Date();
        var date = new Date(Date.parse(newsPostObject.expiry_date));
        if(date.getTime() < now.getTime()) {
            removeNewsPost(newsPostObject.id)
            return
        }
    } catch(error) {
        console.log("Error: " + error)
    }

    document.getElementById(`newsPost-${newsPostObject.id}`).outerHTML = buildNewsPostHTML(newsPostObject, adminStatus)
    reorderPostsByDate();
}

function buildNewsPostHTML(newsPostObject, adminStatus) {
    html = "";

    // parse date object
    var transparentHTML = "";
    try {
        var now = new Date();
        var date = new Date(Date.parse(newsPostObject.date));
        var date_string = date.toLocaleDateString();
        if(date.getTime() > now.getTime()) {
            transparentHTML = "transparent"
        }
    } catch {
        var date_string = "Error"
    }

    html += `<div class="news-post ${transparentHTML}" id="newsPost-${newsPostObject.id}">`

    html += `<input type="hidden" id="newsPostDate-${newsPostObject.id}" value="${newsPostObject.date}"></input>`

    if(adminStatus) {
        html += `
            <div class="profile-icon top-right-button" style="padding-top: 0;">
                <img class="icon-img clickable" src="/assets/edit.svg" title="Edit news post" onclick="editNewsPost('${newsPostObject.id}');">
                <img class="icon-img clickable" src="/assets/trash-2.svg" title="Delete news post" onclick="deleteNewsPost('${newsPostObject.id}');">
            </div>
        `
    }
    
    html += '<div id="news-title" class="title">';
    html += newsPostObject.title
    html += '</div>';

    html += '<div id="news-body" class="text-body">';
    html += newsPostObject.body
    html += '</div>';

    html += '<div id="news-body" class="text-date">';
    html += date_string
    html += '</div>';

    html += '</div>'

    return html
}

function reorderPostsByDate() {
    var wrapperId = "news-box"

    // Get the wrapper element
    const wrapper = document.getElementById(wrapperId);
  
    if (!wrapper) {
      console.error('Wrapper element not found');
      return;
    }
  
    // Convert child elements into an array for sorting
    const posts = Array.from(wrapper.children);
  
    // Sort posts based on the date in the hidden input
    posts.sort((a, b) => {
      const dateA = new Date(a.querySelector('input[type="hidden"]').value);
      const dateB = new Date(b.querySelector('input[type="hidden"]').value);
  
      // Sort in descending order (latest date first)
      return dateB - dateA;
    });
  
    // Append sorted posts back to the wrapper
    posts.forEach(post => wrapper.appendChild(post));
}

function getWishlists(){
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

                if(wishlists.length > 0) {
                    placeWishlists(wishlists);
                } else {
                    document.getElementById('wishlists-front-page').style.display = 'none'
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists?top=5&expired=false");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWishlists(wishlistArray) {
    wishlistsHTML = "";

    for(var i = 0; i < wishlistArray.length; i++) {
        var wishlistHTML = createWishlistHTML(wishlistArray[i])
        wishlistsHTML += wishlistHTML
    }

    document.getElementById('wishlists-box').innerHTML = wishlistsHTML
}

function createWishlistHTML(wishlistObject) {
    return `
        <div class="wishlist-minimal clickable" title="Go to wishlist" onclick="location.href='/wishlists/${wishlistObject.id}'">
            <img class="icon-img" src="/assets/list.svg" style="margin-right: 0.25em;">
            <p id="wishlistName-${wishlistObject.id}">
            ${wishlistObject.name}
            </p>
        </div>
    `;
}