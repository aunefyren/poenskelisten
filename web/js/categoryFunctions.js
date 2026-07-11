// Wish categories: shared logic for grouping wishes under collapsible headers and
// for the category picker in the create/edit wish modal. Loaded by both the
// authenticated wishlist page and the read-only public page.

// Categories for the current wishlist, used to populate the picker. Refreshed
// after wishes/categories change. Unused on the public (read-only) page.
var wishlistCategories = [];

function get_categories(wishlist_id) {

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {

            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                return;
            }

            if(result.error) {
                // Non-fatal: members without picker access just get no categories.
                console.log("Could not load categories: " + result.error);
            } else {
                wishlistCategories = result.categories || [];
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/wishlists/" + wishlist_id + "/categories");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// Refresh the cached category list after a change, when running on a page that
// loads categories (the authenticated wishlist page).
function refreshCategories(wishlistID) {
    if(typeof get_categories === "function") {
        get_categories(wishlistID);
    }
}

// Group wishes into their categories before rendering, so similar wishes (e.g.
// four vinyl records) collapse under one header. Returns ordered category
// buckets plus a trailing list of uncategorized wishes.
function bucketWishesByCategory(wishes_array) {
    var categories = {};    // id -> { name, sort_order, wishes: [] }
    var categoryOrder = [];
    var uncategorized = [];

    for(var i = 0; i < wishes_array.length; i++) {
        var wish = wishes_array[i];
        if(wish.category) {
            var catID = wish.category.id;
            if(!categories[catID]) {
                categories[catID] = { name: wish.category.name, sort_order: wish.category.sort_order, wishes: [] };
                categoryOrder.push(catID);
            }
            categories[catID].wishes.push(wish);
        } else {
            uncategorized.push(wish);
        }
    }

    categoryOrder.sort(function(a, b) {
        var diff = categories[a].sort_order - categories[b].sort_order;
        if(diff !== 0) return diff;
        return categories[a].name.localeCompare(categories[b].name);
    });

    return { categories: categories, categoryOrder: categoryOrder, uncategorized: uncategorized };
}

function categorySectionHTML(catID, name, innerHTML, count) {
    var html = '';
    html += '<div class="wish-category" id="wish_category_' + catID + '">';
    html += '<div class="wish-category-header clickable unselectable" onclick="toggle_category(\'' + catID + '\')" title="Collapse/expand">';
    html += '<img id="wish_category_' + catID + '_arrow" class="icon-img wish-category-arrow" src="/assets/chevron-down.svg">';
    html += '<span class="wish-category-name">' + name + '</span>';
    html += '<span class="wish-category-count">(' + count + ')</span>';
    html += '</div>';
    html += '<div class="wish-category-wishes expanded" id="wish_category_' + catID + '_wishes">';
    html += innerHTML;
    html += '</div>';
    html += '</div>';
    return html;
}

// Render a bucketed wish list into the wishes box. `renderWishList` is supplied
// by the page so each page keeps its own generate_wish_html call signature.
function placeWishesGrouped(wishes_array, renderWishList) {
    var buckets = bucketWishesByCategory(wishes_array);
    var html = '';

    for(var c = 0; c < buckets.categoryOrder.length; c++) {
        var catID = buckets.categoryOrder[c];
        var cat = buckets.categories[catID];
        html += categorySectionHTML(catID, cat.name, renderWishList(cat.wishes), cat.wishes.length);
    }

    if(buckets.uncategorized.length > 0) {
        if(buckets.categoryOrder.length > 0) {
            html += categorySectionHTML('none', 'Other', renderWishList(buckets.uncategorized), buckets.uncategorized.length);
        } else {
            html += renderWishList(buckets.uncategorized);
        }
    }

    return html;
}

function toggle_category(catID) {
    var wishesEl = document.getElementById("wish_category_" + catID + "_wishes");
    var arrowEl = document.getElementById("wish_category_" + catID + "_arrow");

    if(wishesEl.classList.contains("collapsed")) {
        wishesEl.classList.remove("collapsed")
        wishesEl.classList.add("expanded")
        wishesEl.style.display = ""
        arrowEl.src = "/assets/chevron-down.svg"
    } else {
        wishesEl.classList.remove("expanded")
        wishesEl.classList.add("collapsed")
        wishesEl.style.display = "none"
        arrowEl.src = "/assets/chevron-right.svg"
    }
}

// Build the category picker used in the create/edit wish modal. Lists the
// wishlist's existing categories plus a "+ New category" option that reveals a
// free-text input. `selectedID` preselects the wish's current category on edit.
function categoryPickerHTML(selectedID) {
    var options = '<option value="">No category</option>';

    for(var i = 0; i < wishlistCategories.length; i++) {
        var category = wishlistCategories[i];
        var selected = (selectedID && selectedID == category.id) ? ' selected' : '';
        options += '<option value="' + category.id + '"' + selected + '>' + category.name + '</option>';
    }

    options += '<option value="__new__">+ New category…</option>';

    var html = '';
    html += '<label for="wish_category" style="margin-top: 0.5em;">Category (optional):</label><br>';
    html += '<select name="wish_category" id="wish_category" onchange="toggleNewCategoryInput()">' + options + '</select>';
    html += '<input type="text" name="wish_category_new" id="wish_category_new" placeholder="New category name" autocomplete="off" style="display: none;" />';
    return html;
}

function toggleNewCategoryInput() {
    var select = document.getElementById("wish_category");
    var input = document.getElementById("wish_category_new");

    if(select.value == "__new__") {
        input.style.display = "";
    } else {
        input.style.display = "none";
    }
}

// Read the picker into the wish object as category_id / category_name so the
// values survive the base64 round-trip between modal steps.
function readCategorySelection(wishObject) {
    try {
        var select = document.getElementById("wish_category");
        if(!select) {
            return;
        }

        if(select.value == "__new__") {
            wishObject.category_name = document.getElementById("wish_category_new").value.trim();
            wishObject.category_id = null;
        } else if(select.value == "") {
            wishObject.category_id = null;
            wishObject.category_name = "";
        } else {
            wishObject.category_id = select.value;
            wishObject.category_name = "";
        }
    } catch(e) {
        console.log("Failed to read category selection. Error: " + e)
    }
}
