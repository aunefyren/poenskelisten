function openURLModal(url) {
    var html = '';

    html += `
        <form action="" class="" onsubmit="event.preventDefault(); openURLModalTwo('${url}');">  
            <label for="" style="margin-top: 0.5em;">Are you sure you want to open this link?</label><br>

            <div class="urlWrapper unselectable">
                ${url}
            </div>
            
            <button id="go-button" type="submit" href="/">Go</button>
        </form>
    `;

    toggleModal(html);
}

function openURLModalTwo(url) {
    window.open(url, '_blank')
    toggleModal(false)
}

function deleteWish(wishID, wishlistID, groupID, userID) {
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
                placeWishes(result.wishes, wishlistID, userID);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/wishes/" + wishID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function editWish(wishID, wishlistID, groupID, userID) {
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
                editWishTwo(wishID, wishlistID, groupID, userID, toBASE64(JSON.stringify(result.wish)));
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishes/" + wishID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function editWishTwo(wishID, wishlistID, groupID, userID, wishObjectBase64) {
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))
    
    var html = '';
    
    html += `
        <form action="" onsubmit="event.preventDefault(); editWishThree('${wishID}', '${userID}', '${groupID}', '${userID}', '${wishObjectBase64}');">        
            <label for="wish_name">Edit wish:</label><br>
            <input type="text" name="wish_name" id="wish_name" placeholder="Wish name" value="${wishObject.name}" autocomplete="off" required />
            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function editWishThree(wishID, wishlistID, groupID, userID, wishObjectBase64) {
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))

    try {
        var wishName = document.getElementById("wish_name").value

        if(wishName == "" || wishName.length < 5) {
            alert("Wish name too short");
            return
        }

        wishObject.name = wishName
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }

    wishObjectBase64 = toBASE64(JSON.stringify(wishObject))
    var html = '';
    
    html += `
        <div class="profile-icon clickable top-left-button" onclick="editWishTwo('${wishID}', '${wishlistID}', '${groupID}', '${userID}', '${wishObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); editWishFour('${wishID}', '${userID}', '${wishlistID}', '${groupID}', '${wishObjectBase64}');">
            <label for="wish_note" style="">Optional details:</label><br>

            <textarea name="wish_note" id="wish_note" placeholder="Wish note" value="" autocomplete="off" rows="3" />${wishObject.note}</textarea>

            <input type="text" name="wish_url" id="wish_url" placeholder="Wish URL" value="${wishObject.url}" autocomplete="off" />

            <input type="number" name="wish_price" id="wish_price" step="0.01" min="0" placeholder="Wish price in ${wishObject.currency}" value="${wishObject.price}" autocomplete="off" />

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function editWishFour(wishID, wishlistID, groupID, userID, wishObjectBase64) {
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))

    try {
        var wishNote = document.getElementById("wish_note").value
        var wishURL = document.getElementById("wish_url").value
        var wishPrice = parseFloat(document.getElementById("wish_price").value)
        wishObject.note = wishNote
        wishObject.url = wishURL
        wishObject.price = wishPrice
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }

    wishObjectBase64 = toBASE64(JSON.stringify(wishObject))
    var html = '';
    
    html += `
        <div class="profile-icon clickable top-left-button" onclick="editWishThree('${wishID}', '${wishlistID}', '${groupID}', '${userID}', '${wishObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); editWishFive('${wishID}', '${userID}', '${wishlistID}', '${groupID}', '${wishObjectBase64}');">
            <label id="form-input-icon" for="wish_image" style="">Replace optional image:</label>
            <input type="file" name="wish_image" id="wish_image" placeholder="" value="" accept="image/png, image/jpeg" />
            
            <button id="register-button" type="submit" href="/">Save wish</button>
        </form>
    `;

    toggleModal(html);
}

function editWishFive(wishID, userID, wishlistID, groupID, wishObjectBase64) {
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))

    if(!confirm("Are you sure you want to update this wish?")) {
        return;
    }

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
                "name" : wishObject.name,
                "note" : wishObject.note,
                "url": wishObject.url,
                "price": wishObject.price,
                "image_data": result
            };

            var form_data = JSON.stringify(form_obj);

            editWishSix(form_data, wishID, userID, wishlistID, groupID);
        });
    } else {
        var form_obj = { 
            "name" : wishObject.name,
            "note" : wishObject.note,
            "url": wishObject.url,
            "price": wishObject.price,
            "image_data": ""
        };

        var form_data = JSON.stringify(form_obj);
        editWishSix(form_data, wishID, userID, wishlistID, groupID)
    }

}

function editWishSix(form_data, wishID, userID, wishlistID, groupID) {
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
                placeWish(result.wish, wishlistID, groupID, userID)
                toggleModal(false);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishes/" + wishID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function createWish(wishlistID, userID, currency, wishObjectBase64) {
    try {
        var wishObject = JSON.parse(fromBASE64(wishObjectBase64))
    } catch (error) {
        var wishObject = {
            "name": "",
            "note" : "",
            "image": "",
            "url": "",
            "price": null
        }
        wishObjectBase64 = toBASE64(JSON.stringify(wishObject))
        console.log("Remade object. Error: " + error)
    }

    var html = '';
    
    html += `
        <form action="" onsubmit="event.preventDefault(); createWishTwo('${wishlistID}', '${userID}', '${currency}', '${wishObjectBase64}');">        
            <label for="wish_name">Create wish:</label><br>
            <input type="text" name="wish_name" id="wish_name" placeholder="Wish name" autocomplete="off" value="${wishObject.name}" required />

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createWishTwo(wishlistID, userID, currency, wishObjectBase64) {
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))

    try {
        var wishName = document.getElementById("wish_name").value
        wishObject.name = wishName

        if(wishName == "" || wishName.length < 5) {
            alert("Wish name too short");
            return
        }
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }

    wishObjectBase64 = toBASE64(JSON.stringify(wishObject))
    var html = '';
    
    html += `
        <div class="profile-icon clickable top-left-button" onclick="createWish('${wishlistID}', '${userID}', '${currency}', '${wishObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); createWishThree('${wishlistID}', '${userID}', '${wishObjectBase64}');">
            <label for="wish_note" style="">Optional details:</label><br>

            <textarea name="wish_note" id="wish_note" placeholder="Wish note" autocomplete="off" rows="3" />${wishObject.note}</textarea>

            <input type="text" name="wish_url" id="wish_url" placeholder="Wish URL" autocomplete="off" value="${wishObject.url}" />

            <input type="number" name="wish_price" id="wish_price" step="0.01" min="0" placeholder="Wish price in ${currency}" value="${wishObject.price}" autocomplete="off" />

            <button id="register-button" type="submit" href="/">Next</button>
        </form>
    `;

    toggleModal(html);
}

function createWishThree(wishlistID, userID, wishObjectBase64) {
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))

    try {
        var wishNote = document.getElementById("wish_note").value
        var wishURL = document.getElementById("wish_url").value
        var wishPrice = parseFloat(document.getElementById("wish_price").value)
        wishObject.note = wishNote
        wishObject.url = wishURL
        wishObject.price = wishPrice
    } catch (error) {
        console.log("Failed to get values. Error: " + error)
    }

    wishObjectBase64 = toBASE64(JSON.stringify(wishObject))
    var html = '';
    
    html += `
        <div class="profile-icon clickable top-left-button" onclick="createWishTwo('${wishlistID}', '${userID}', '${currency}', '${wishObjectBase64}');" title="Go back" style="">
            <img class="icon-img" src="/assets/arrow-left.svg">
        </div>

        <form action="" onsubmit="event.preventDefault(); createWishFour('${wishlistID}', '${userID}', '${wishObjectBase64}');">
            <label id="form-input-icon" for="wish_image" style="">Replace optional image:</label>
            <input type="file" name="wish_image" id="wish_image" placeholder="" value="${wishObject.image}" accept="image/png, image/jpeg" />
            
            <button id="register-button" type="submit" href="/">Create wish</button>
        </form>
    `;

    toggleModal(html);
}

function createWishFour(wishlistID, userID, wishObjectBase64){
    var wishObject = JSON.parse(fromBASE64(wishObjectBase64))
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
                "name" : wishObject.name,
                "note" : wishObject.note,
                "url": wishObject.url,
                "price": wishObject.price,
                "image_data": result
            };

            var form_data = JSON.stringify(form_obj);

            createWishFive(form_data, wishlistID, userID);
        });

    } else {
        var form_obj = { 
                "name" : wishObject.name,
                "note" : wishObject.note,
                "url": wishObject.url,
                "price": wishObject.price,
                "image_data": ""
            };

        var form_data = JSON.stringify(form_obj);

        createWishFive(form_data, wishlistID, userID);
    }
}

function createWishFive(form_data, wishlistID, userID) {
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
                placeWishes(result.wishes, wishlistID, userID);
                toggleModal(false);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/wishes?wishlist=" + wishlistID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}