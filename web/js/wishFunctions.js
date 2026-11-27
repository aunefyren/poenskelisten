function openURLModal(url) {
    var html = '';

    html += `
        <form action="" class="" ${onSubmit('openURLModalTwo', url)}>
            <label for="" style="margin-top: 0.5em;">Are you sure you want to open this link?</label><br>

            <div class="urlWrapper unselectable">
                ${escapeHTML(url)}
            </div>
            
            <button id="go-button" type="submit" href="/">Go</button>
        </form>
    `;

    toggleModal(html);
}

function openURLModalTwo(url) {
    openExternalURL(url)
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
                refreshCategories(wishlistID);
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
                showEditWishForm(wishID, wishlistID, userID, result.wish);
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

// Creating and editing a wish share one single-screen form: every field is on
// it at once, so nothing has to be carried between steps, and a save the server
// refuses leaves the form open with what was typed.
function wishFormFieldsHTML(wishObject, currency, selectedCategoryID) {
    var price = (typeof wishObject.price === "number" && !isNaN(wishObject.price)) ? wishObject.price : "";

    return `
        <label class="form-label" for="wish_name">Name</label>
        <input type="text" name="wish_name" id="wish_name" placeholder="Wish name" autocomplete="off" value="${escapeHTML(wishObject.name)}" required />

        <label class="form-label" for="wish_url">Link <span class="form-label-optional">(optional)</span></label>
        <input type="text" name="wish_url" id="wish_url" placeholder="Wish URL" inputmode="url" autocomplete="off" value="${escapeHTML(wishObject.url)}" />

        <label class="form-label" for="wish_price">Price <span class="form-label-optional">(optional)</span></label>
        <input type="number" name="wish_price" id="wish_price" step="0.01" min="0" inputmode="decimal" placeholder="Wish price in ${escapeHTML(currency)}" value="${price}" autocomplete="off" />

        <label class="form-label" for="wish_note">Note <span class="form-label-optional">(optional)</span></label>
        <textarea name="wish_note" id="wish_note" placeholder="Wish note" autocomplete="off" rows="3">${escapeHTML(wishObject.note)}</textarea>

        ${categoryPickerHTML(selectedCategoryID)}
    `;
}

function wishFormButtonsHTML(submitText) {
    return `
        <div class="alert alert-danger form-error" id="wish-form-error" role="alert" style="display: none;"></div>

        <button id="wish-form-submit" type="submit">${submitText}</button>
        <button type="button" class="button-secondary" data-on-click="toggleModal">Cancel</button>
    `;
}

function createWish(wishlistID, userID, currency) {
    var wishObject = { "name": "", "note": "", "url": "", "price": null };

    var html = `
        <form action="" ${onSubmit('submitCreateWish', String(wishlistID), String(userID))}>
            <div class="title-two wish-form-title">Create wish</div>

            ${wishFormFieldsHTML(wishObject, currency, "")}

            <label class="form-label" for="wish_image">Image <span class="form-label-optional">(optional)</span></label>
            <input type="file" name="wish_image" id="wish_image" accept="image/png, image/jpeg" />

            ${wishFormButtonsHTML("Create wish")}
        </form>
    `;

    toggleModal(html);
    document.getElementById("wish_name").focus();
}

function showEditWishForm(wishID, wishlistID, userID, wishObject) {
    var selectedCategoryID = wishObject.category_id ? wishObject.category_id : (wishObject.category ? wishObject.category.id : "");

    // A wish without an image just offers the upload; one with an image picks
    // between keeping, replacing and removing it.
    var imageHTML = '';
    if(wishObject.image) {
        imageHTML = `
            <div class="form-label" id="wish-image-label">Image</div>
            <div class="wish-image-edit">
                <img id="modal-img" class="wish-image-edit-img" src="/assets/loading.gif" alt="Current image of the wish">
            </div>

            <div class="button-grouping" id="image-button-grouping" role="group" aria-labelledby="wish-image-label">
                <button type="button" id="button-grouping-part-leave" class="button-grouping-part button-grouping-part-border button-grouping-part-selected" aria-pressed="true" data-on-click="toggleEditWishImage" data-click-args="[&quot;leave&quot;]">Keep</button>
                <button type="button" id="button-grouping-part-replace" class="button-grouping-part button-grouping-part-border" aria-pressed="false" data-on-click="toggleEditWishImage" data-click-args="[&quot;replace&quot;]">Replace</button>
                <button type="button" id="button-grouping-part-delete" class="button-grouping-part" aria-pressed="false" data-on-click="toggleEditWishImage" data-click-args="[&quot;delete&quot;]">Remove</button>
            </div>

            <div class="full-width" id="edit-wish-replace-image" style="display: none;">
                <label class="form-label" for="wish_image">New image</label>
                <input type="file" name="wish_image" id="wish_image" accept="image/png, image/jpeg" />
            </div>
        `;
    } else {
        imageHTML = `
            <label class="form-label" for="wish_image">Image <span class="form-label-optional">(optional)</span></label>
            <input type="file" name="wish_image" id="wish_image" accept="image/png, image/jpeg" />
        `;
    }

    var html = `
        <form action="" ${onSubmit('submitEditWish', String(wishID), String(wishlistID), String(userID))}>
            <div class="title-two wish-form-title">Edit wish</div>

            ${wishFormFieldsHTML(wishObject, wishObject.currency, selectedCategoryID)}

            ${imageHTML}

            <input type="hidden" id="image_delete" value="false">

            ${wishFormButtonsHTML("Save wish")}
        </form>
    `;

    toggleModal(html);
    document.getElementById("wish_name").focus();

    if(wishObject.image) {
        GetWishImage(wishID);
    }
}

function toggleEditWishImage(buttonName) {
    var choices = ['leave', 'replace', 'delete'];
    if(choices.indexOf(buttonName) === -1) {
        console.log('No button case hit. Input: ' + buttonName)
        return;
    }

    var replacing = buttonName == 'replace';
    document.getElementById('edit-wish-replace-image').style.display = replacing ? 'block' : 'none';
    document.getElementById('image_delete').value = buttonName == 'delete' ? 'true' : 'false';
    document.getElementById('wish_image').required = replacing;
    if(!replacing) {
        document.getElementById('wish_image').value = "";
    }

    choices.forEach(function(choice) {
        var button = document.getElementById('button-grouping-part-' + choice);
        button.classList.toggle('button-grouping-part-selected', choice == buttonName);
        button.setAttribute('aria-pressed', choice == buttonName ? 'true' : 'false');
    });
}

function showWishFormError(message) {
    var errorBox = document.getElementById("wish-form-error");
    if(!errorBox) {
        error(message);
        return;
    }
    errorBox.textContent = message;
    errorBox.style.display = "block";
    errorBox.scrollIntoView({ block: "nearest" });
}

// The submit button stays disabled while a save is in flight, so a double tap
// can't create the same wish twice.
function setWishFormBusy(busy) {
    var button = document.getElementById("wish-form-submit");
    if(button) {
        button.disabled = busy;
    }
}

// Reads the shared fields, or returns null when they can't be saved (showing
// why) or a save is already under way.
function readWishForm() {
    if(document.getElementById("wish-form-submit").disabled) {
        return null;
    }

    var errorBox = document.getElementById("wish-form-error");
    errorBox.style.display = "none";

    var wishObject = {
        "name": document.getElementById("wish_name").value,
        "note": document.getElementById("wish_note").value,
        "url": document.getElementById("wish_url").value,
        "price": parseFloat(document.getElementById("wish_price").value),
        "category_id": null,
        "category_name": ""
    };
    readCategorySelection(wishObject);

    if(wishObject.name.trim().length < 5) {
        showWishFormError("The name of the wish must be five or more letters.");
        document.getElementById("wish_name").focus();
        return null;
    }

    return wishObject;
}

// Calls back with the chosen image as a data URL, or "" when none was picked.
function readWishImage(callback) {
    var wishImage = document.getElementById('wish_image').files[0];

    if(!wishImage) {
        callback("");
        return;
    }

    if(wishImage.size > 10000000) {
        showWishFormError("Image exceeds 10MB size limit.");
        setWishFormBusy(false);
        return;
    }

    get_base64(wishImage).then(callback, function() {
        showWishFormError("Could not read the image.");
        setWishFormBusy(false);
    });
}

function submitCreateWish(wishlistID, userID) {
    var wishObject = readWishForm();
    if(!wishObject) {
        return;
    }

    setWishFormBusy(true);
    readWishImage(function(imageData) {
        var form_obj = {
            "name" : wishObject.name,
            "note" : wishObject.note,
            "url": wishObject.url,
            "price": wishObject.price,
            "image_data": imageData,
            "category_id": wishObject.category_id ? wishObject.category_id : null,
            "category_name": wishObject.category_name ? wishObject.category_name : ""
        };

        sendWishForm(api_url + "auth/wishes?wishlist=" + wishlistID, JSON.stringify(form_obj), wishlistID, userID);
    });
}

function submitEditWish(wishID, wishlistID, userID) {
    var wishObject = readWishForm();
    if(!wishObject) {
        return;
    }

    var imageDelete = document.getElementById('image_delete').value == 'true';
    if(imageDelete && !confirm("Remove the image from this wish?")) {
        return;
    }

    setWishFormBusy(true);
    readWishImage(function(imageData) {
        var form_obj = {
            "name" : wishObject.name,
            "note" : wishObject.note,
            "url": wishObject.url,
            "price": wishObject.price,
            "image_data": imageData,
            "image_delete": imageDelete,
            "category_id": wishObject.category_id ? wishObject.category_id : null,
            "category_name": wishObject.category_name ? wishObject.category_name : ""
        };

        sendWishForm(api_url + "auth/wishes/" + wishID, JSON.stringify(form_obj), wishlistID, userID);
    });
}

function sendWishForm(url, form_data, wishlistID, userID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                showWishFormError("Could not reach API.");
                setWishFormBusy(false);
                return;
            }

            if(result.error) {
                showWishFormError(result.error);
                setWishFormBusy(false);
            } else {
                // Re-render the full list so the wish lands under its (possibly
                // changed) category, and refresh the picker's category cache.
                placeWishes(result.wishes, wishlistID, userID);
                refreshCategories(wishlistID);
                toggleModal(false);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", url);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
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

// Functions this file's markup may invoke (see registerActions in actions.js).
registerActions({
    openURLModal,
    openURLModalTwo,
    deleteWish,
    editWish,
    toggleEditWishImage,
    submitEditWish,
    createWish,
    submitCreateWish,
});
