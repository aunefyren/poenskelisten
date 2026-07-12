// Wish categories: shared logic for grouping wishes under collapsible headers and
// for the category picker in the create/edit wish modal. Loaded by both the
// authenticated wishlist page and the read-only public page.

// Categories for the current wishlist, used to populate the picker. Refreshed
// after wishes/categories change. Unused on the public (read-only) page.
var wishlistCategories = [];

// --- Wish sorting -----------------------------------------------------------
// Sorting is done entirely client-side, in the shared grouping layer, so both
// the authenticated wishlist page and the read-only public page behave
// identically without any API changes. The user's choice is remembered across
// reloads and across wishlists via localStorage.

var WISH_SORT_STORAGE_KEY = "poenskelistenWishSort";

// currentWishSort: "date_asc" | "date_desc" | "price_asc" | "price_desc".
// Date sorting uses updated_at (falling back to creation date for un-edited
// wishes). Default "date_asc" keeps oldest-first, matching the API's baseline
// order for a freshly created list, so nothing shifts for users who never touch
// the control.
var currentWishSort = "date_asc";

// currentWishLayout: "grouped" keeps category headers and sorts within them;
// "flat" ignores categories and shows one globally sorted list. Until the user
// explicitly picks a layout (wishLayoutExplicit stays false), the layout is
// decided per wishlist by effectiveWishLayout(): grouped when the list has
// categories, flat when it has none.
var currentWishLayout = "grouped";
var wishLayoutExplicit = false;

loadWishSortPrefs();

function loadWishSortPrefs() {
    try {
        var raw = localStorage.getItem(WISH_SORT_STORAGE_KEY);
        if(!raw) {
            return;
        }
        var prefs = JSON.parse(raw);
        if(prefs.sort) {
            currentWishSort = prefs.sort;
        }
        // Only an explicit past choice is stored; its presence marks it explicit.
        if(prefs.layout) {
            currentWishLayout = prefs.layout;
            wishLayoutExplicit = true;
        }
    } catch(e) {
        console.log("Failed to load wish sort preferences. Error: " + e);
    }
}

function saveWishSortPrefs() {
    try {
        // Persist the layout only once the user has explicitly chosen one, so
        // the per-wishlist auto default keeps applying until then.
        var prefs = { sort: currentWishSort };
        if(wishLayoutExplicit) {
            prefs.layout = currentWishLayout;
        }
        localStorage.setItem(WISH_SORT_STORAGE_KEY, JSON.stringify(prefs));
    } catch(e) {
        console.log("Failed to save wish sort preferences. Error: " + e);
    }
}

// wishesHaveCategory reports whether any wish in the list carries a category.
function wishesHaveCategory(list) {
    for(var i = 0; i < list.length; i++) {
        if(list[i].category) {
            return true;
        }
    }
    return false;
}

// effectiveWishLayout resolves the layout to use for a given wish list. Grouping
// is meaningless without categories, so a list with none is always flat,
// regardless of any saved preference (which re-applies once categories exist).
// Otherwise the user's explicit choice wins, falling back to grouped.
function effectiveWishLayout(list) {
    if(!wishesHaveCategory(list)) {
        return "flat";
    }
    if(wishLayoutExplicit) {
        return currentWishLayout;
    }
    return "grouped";
}

// wishDateValue returns the wish's last-updated time in ms, for date sorting.
// updated_at is set to created_at on insert, so un-edited wishes still sort by
// their creation date, and it matches the date shown on the wish card.
function wishDateValue(wish) {
    var value = Date.parse(wish.updated_at);
    return isNaN(value) ? 0 : value;
}

// wishPriceValue returns a numeric price, or null when the wish has no usable
// price. A missing price and a zero price are both treated as "no price" to
// match how the card rendering hides them, and such wishes always sink to the
// bottom regardless of sort direction.
function wishPriceValue(wish) {
    if(wish.price === null || wish.price === undefined || wish.price === 0) {
        return null;
    }
    return Number(wish.price);
}

function compareWishPrice(a, b, direction) {
    var pa = wishPriceValue(a);
    var pb = wishPriceValue(b);

    if(pa === null && pb === null) return 0;
    if(pa === null) return 1;   // price-less wishes always last
    if(pb === null) return -1;

    return (pa - pb) * direction;
}

// sortWishes returns a new array sorted by the given sort value. It never
// mutates its input.
function sortWishes(list, sortValue) {
    var sorted = list.slice();

    sorted.sort(function(a, b) {
        switch(sortValue) {
            case "date_desc": return wishDateValue(b) - wishDateValue(a);
            case "date_asc":  return wishDateValue(a) - wishDateValue(b);
            case "price_asc":  return compareWishPrice(a, b, 1);
            case "price_desc": return compareWishPrice(a, b, -1);
            default:           return 0;
        }
    });

    return sorted;
}

// sortControlsHTML builds the layout + sort control cluster shown above the
// wish list. `selected` markers reflect the persisted state.
function sortControlsHTML(effectiveLayout, showLayoutToggle) {
    function opt(value, label, current) {
        return '<option value="' + value + '"' + (value == current ? ' selected' : '') + '>' + label + '</option>';
    }

    // The layout toggle is only offered when the list has categories to group by.
    var layout = '';
    if(showLayoutToggle) {
        layout += '<select class="wish-sort-select" id="wish-layout-select" onchange="onWishLayoutChange(this.value)" title="Layout">';
        layout += opt("grouped", "Grouped by category", effectiveLayout);
        layout += opt("flat", "Flat list", effectiveLayout);
        layout += '</select>';
    }

    var sort = '';
    sort += '<select class="wish-sort-select" id="wish-sort-select" onchange="onWishSortChange(this.value)" title="Sort by">';
    sort += opt("date_desc", "Date updated (newest)", currentWishSort);
    sort += opt("date_asc", "Date updated (oldest)", currentWishSort);
    sort += opt("price_asc", "Price (low → high)", currentWishSort);
    sort += opt("price_desc", "Price (high → low)", currentWishSort);
    sort += '</select>';

    return '<div class="wish-sort-controls unselectable">' + layout + sort + '</div>';
}

// renderSortControls injects the control cluster into the page's container, if
// present. Hidden when there is nothing to sort. Takes the wish list so the
// layout dropdown can reflect the per-wishlist auto default.
function renderSortControls(wishesArray) {
    var container = document.getElementById("wish-sort-controls");
    if(!container) {
        return;
    }
    var list = wishesArray || [];
    if(list.length === 0) {
        container.innerHTML = '';
        return;
    }
    var hasCategories = wishesHaveCategory(list);
    container.innerHTML = sortControlsHTML(effectiveWishLayout(list), hasCategories);
}

function onWishSortChange(value) {
    currentWishSort = value;
    saveWishSortPrefs();
    triggerWishRerender();
}

function onWishLayoutChange(value) {
    currentWishLayout = value;
    wishLayoutExplicit = true;
    saveWishSortPrefs();
    triggerWishRerender();
}

// triggerWishRerender asks the current page to re-render its wishes from the
// cached array. Each page defines rerenderWishes() with its own arguments.
function triggerWishRerender() {
    if(typeof rerenderWishes === "function") {
        rerenderWishes();
    }
}

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

// Render a wish list into the wishes box, honouring the current sort and layout.
// `renderWishList` is supplied by the page so each page keeps its own
// generate_wish_html call signature.
//
// In "flat" layout categories are ignored and the whole list is sorted globally.
// In "grouped" layout category headers keep their manual sort_order and the sort
// is applied within each header (and the trailing "Other" bucket).
function placeWishesGrouped(wishes_array, renderWishList) {
    if(effectiveWishLayout(wishes_array) === "flat") {
        return renderWishList(sortWishes(wishes_array, currentWishSort));
    }

    var buckets = bucketWishesByCategory(wishes_array);
    var html = '';

    for(var c = 0; c < buckets.categoryOrder.length; c++) {
        var catID = buckets.categoryOrder[c];
        var cat = buckets.categories[catID];
        var catWishes = sortWishes(cat.wishes, currentWishSort);
        html += categorySectionHTML(catID, cat.name, renderWishList(catWishes), catWishes.length);
    }

    if(buckets.uncategorized.length > 0) {
        var otherWishes = sortWishes(buckets.uncategorized, currentWishSort);
        if(buckets.categoryOrder.length > 0) {
            html += categorySectionHTML('none', 'Other', renderWishList(otherWishes), otherWishes.length);
        } else {
            html += renderWishList(otherWishes);
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
