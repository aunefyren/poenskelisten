function createNewsPost(adminStatus) {
    var html = '';
    now = new Date()
    nowDate = now.toISOString().split('T')[0]

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); createNewsPostTwo(${adminStatus});">
            <label for="news_title">Create post:</label><br>
            <input type="text" name="news_title" id="news_title" placeholder="Post title" autocomplete="off" required />
            
            <textarea rows="3" name="news_body" id="news_body" placeholder="Post body" autocomplete="off" required /></textarea>

            <label for="news_date">Release date</label><br>
            <input type="date" name="news_date" id="news_date" placeholder="Post date" autocomplete="off" required value="${nowDate}" />

            <label for="news_date_two">Optional expiry date</label><br>
            <input type="date" name="news_date_two" id="news_date_two" placeholder="Post date" autocomplete="off" />
            
            <button id="register-button" type="submit" href="/">Create post</button>
        </form>
    `;

    toggleModal(html);
}

function createNewsPostTwo(adminStatus) {
    var news_title = document.getElementById("news_title").value;
    var news_body = document.getElementById("news_body").value;

    try {
        var news_date = document.getElementById("news_date").value;
        var news_date_object = new Date(Date.parse(news_date));
        var news_date_str = news_date_object.toISOString();
    } catch(e) {
        error("Failed to parse date request.")
        console.log("Error: " + e)
        return;
    }

    try {
        var news_date = document.getElementById("news_date_two").value;
        var news_date_object = new Date(Date.parse(news_date));
        var news_expiry_date_str = news_date_object.toISOString();
    } catch(e) {
        var news_expiry_date_str = null
    }

    var form_obj = { 
        "title" : news_title,
        "body" : news_body,
        "date": news_date_str,
        "expiry_date": news_expiry_date_str
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
                news = result.news;
                placeNewsPosts(news, adminStatus);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/news");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function deleteNewsPost(newsPostID) {
    if(!confirm("Are you sure you want to delete this news post?")) {
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
                removeNewsPost(newsPostID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "admin/news/" + newsPostID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function editNewsPost(newsPostID) {
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
                editNewsPostTwo(result.news, newsPostID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/news/" + newsPostID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function editNewsPostTwo(newsObject, newsPostID) {
    var html = '';
    
    if(newsObject.date) {
        var newsPostDateObject = new Date(newsObject.date)
        var newsPostDate = newsPostDateObject.toISOString().split('T')[0];
    } else {
        var now = new Date
        var newsPostDate = now.toISOString().split('T')[0];
    }

    if(newsObject.expiry_date) {
        var newsPostExpiryDateObject = new Date(newsObject.expiry_date)
        var newsPostExpiryDate = newsPostExpiryDateObject.toISOString().split('T')[0];
    } else {
        var newsPostExpiryDate = ""
    }

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); editNewsPostThree('${newsPostID}');">
            <label for="news_title">Create post:</label><br>
            <input type="text" name="news_title" id="news_title" placeholder="Post title" autocomplete="off" value="${newsObject.title}" required />
            
            <textarea rows="3" name="news_body" id="news_body" placeholder="Post body" autocomplete="off" required />${newsObject.body}</textarea>

            <label for="news_date">Release date</label><br>
            <input type="date" name="news_date" id="news_date" placeholder="Post date" autocomplete="off" required value="${newsPostDate}" />

            <label for="news_date_two">Optional expiry date</label><br>
            <input type="date" name="news_date_two" id="news_date_two" placeholder="Post date" autocomplete="off" value="${newsPostExpiryDate}" />
            
            <button id="register-button" type="submit" href="/">Save news post</button>
        </form>
    `;

    toggleModal(html);
}

function editNewsPostThree(newsPostID) {
    var news_title = document.getElementById("news_title").value;
    var news_body = document.getElementById("news_body").value;

    try {
        var news_date = document.getElementById("news_date").value;
        var news_date_object = new Date(Date.parse(news_date));
        var news_date_str = news_date_object.toISOString();
    } catch(e) {
        error("Failed to parse date request.")
        console.log("Error: " + e)
        return;
    }

    try {
        var news_date = document.getElementById("news_date_two").value;
        var news_date_object = new Date(Date.parse(news_date));
        var news_expiry_date_str = news_date_object.toISOString();
    } catch(e) {
        var news_expiry_date_str = null
    }

    var form_obj = { 
        "title" : news_title,
        "body" : news_body,
        "date": news_date_str,
        "expiry_date": news_expiry_date_str
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
                news = result.news;
                placeNewsPost(news, true);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/news/" + newsPostID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}