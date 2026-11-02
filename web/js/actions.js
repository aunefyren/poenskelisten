// Declarative event handlers, loaded before every other script so each file
// can register its actions as it loads.
//
// The Content-Security-Policy allows no inline script, so markup can't carry
// on* attributes. An element instead names a registered action and its
// arguments as data:
//
//     `<button ${onClick("claim_wish", wishID, wishlistID)}>`
//
// renders data-on-click="claim_wish" plus the arguments as JSON in an
// HTML-escaped attribute. Values from the API therefore only ever travel as
// data; they are never spliced into code, which is how a quote in a wish URL
// or a user's name used to break out into script. Only functions passed to
// registerActions can be invoked, so injected markup can't call arbitrary
// globals either.
var registeredActions = {};
var DECLARATIVE_EVENTS = ["click", "submit", "change", "keydown"];

function registerActions(actions) {
    for(var name in actions) {
        if(Object.prototype.hasOwnProperty.call(actions, name)) {
            registeredActions[name] = actions[name];
        }
    }
}

function actionAttrs(eventType, name, args) {
    var html = 'data-on-' + eventType + '="' + escapeActionAttr(name) + '"';
    if(args.length > 0) {
        html += ' data-' + eventType + '-args="' + escapeActionAttr(JSON.stringify(args)) + '"';
    }
    return html;
}

function onClick(name, ...args) { return actionAttrs("click", name, args); }
function onSubmit(name, ...args) { return actionAttrs("submit", name, args); }
function onChange(name, ...args) { return actionAttrs("change", name, args); }

// Run every action on the path from the event target up to the document, the
// way inline handlers on nested elements all fired as the event bubbled.
// A form with a submit action is handled in-page (every form here used to call
// event.preventDefault() first), as is an href="#" link with a click action.
function dispatchDeclarativeEvent(event) {
    var nameAttr = "data-on-" + event.type;
    var argsAttr = "data-" + event.type + "-args";

    var element = event.target instanceof Element ? event.target.closest("[" + nameAttr + "]") : null;
    while(element) {
        if(event.type === "submit" || (element.tagName === "A" && element.getAttribute("href") === "#")) {
            event.preventDefault();
        }

        var name = element.getAttribute(nameAttr);
        var action = Object.prototype.hasOwnProperty.call(registeredActions, name) ? registeredActions[name] : null;
        if(typeof action !== "function") {
            console.error("No registered action named '" + name + "' for " + event.type + ".");
        } else {
            var args = [];
            var rawArgs = element.getAttribute(argsAttr);
            if(rawArgs) {
                try {
                    args = JSON.parse(rawArgs);
                } catch(e) {
                    console.error("Malformed " + argsAttr + " on action '" + name + "'.");
                    args = null;
                }
            }
            if(Array.isArray(args)) {
                action.apply(element, args);
            }
        }

        if(event.cancelBubble || !element.parentElement) {
            break;
        }
        element = element.parentElement.closest("[" + nameAttr + "]");
    }
}

DECLARATIVE_EVENTS.forEach(function(eventType) {
    document.addEventListener(eventType, dispatchDeclarativeEvent);
});

// Navigate to a path on this site. Only same-origin, absolute paths are
// accepted so a value from the API can't turn a button into an open redirect.
function navigateTo(path) {
    path = String(path);
    if(path.charAt(0) !== "/" || path.charAt(1) === "/" || path.charAt(1) === "\\") {
        console.error("Refusing to navigate to non-local path.");
        return;
    }
    window.location.href = path;
}

function showAlert(message) {
    alert(message);
}

function escapeActionAttr(value) {
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

registerActions({
    navigateTo,
    showAlert,
});
