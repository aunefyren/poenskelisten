// Forced MFA-enrollment gate page. Reached when the administrator enforces MFA and
// the logged-in (SSO) user hasn't enrolled yet — before they hold an access token.
// All calls go to the SSO-authed /open MFA endpoints (cookie sent automatically).

function runEnroll() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if(result.error) {
                error(result.error);
                if(this.status === 401) {
                    setTimeout(function() { window.location.href = '/login'; }, 1500);
                }
                return;
            }
            renderActivate(result.secret, result.otpauth_url, result.qr_code);
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/mfa/enroll");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send();
}

function renderActivate(secret, otpauthURL, qrCode) {
    var qrHTML = qrCode
        ? `<img src="${qrCode}" alt="Scan this QR code with your authenticator app" style="width: 12em; height: 12em; max-width: 100%; margin: 0.5em auto; display: block; image-rendering: pixelated;">`
        : "";

    document.getElementById('content').innerHTML = `
        <div class="module">
            <p style="margin-bottom: 0.5em;">Your administrator requires two-factor authentication. Scan this with your authenticator app:</p>
            ${qrHTML}
            <p style="margin: 0.5em 0; font-size: 0.85em;">Can't scan it? Enter this key manually:</p>
            <p style="font-family: monospace; word-break: break-all; font-size: 1.1em;">${secret}</p>
            <p style="word-break: break-all; font-size: 0.8em;"><a href="${otpauthURL}">Open in authenticator app</a></p>

            <form action="" class="icon-border" style="margin-top: 1em;" onsubmit="event.preventDefault(); enrollActivate();">
                <p style="margin-bottom: 0.5em;">Enter the 6-digit code to confirm:</p>
                <input type="text" id="enroll_code" placeholder="6-digit code" autocomplete="one-time-code" inputmode="numeric" required/>
                <button type="submit" style="padding: 0.75em 1em;">Confirm and enable</button>
            </form>
        </div>
    `;
}

function enrollActivate() {
    var code = document.getElementById('enroll_code').value;

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if(result.error) {
                error(result.error);
                try { document.getElementById('enroll_code').value = ""; } catch(e) { console.log(e); }
                return;
            }
            success(result.message);
            if(result.recovery_codes && result.recovery_codes.length) {
                showRecoveryCodes(result.recovery_codes);
            } else {
                window.location.href = '/';
            }
        } else {
            info("Enabling two-factor authentication...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/mfa/activate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(JSON.stringify({ code: code }));
}

function showRecoveryCodes(codes) {
    var codesHTML = "";
    for(var i = 0; i < codes.length; i++) {
        codesHTML += `<div style="font-family: monospace; font-size: 1.05em;">${codes[i]}</div>`;
    }

    document.getElementById('content').innerHTML = `
        <div class="module">
            <p style="margin-bottom: 0.5em;"><b>Save these recovery codes.</b> Each can be used once if you lose access to your authenticator. They won't be shown again.</p>
            <div style="margin: 0.5em auto; padding: 0.5em; border: 1px solid; max-width: 14em; text-align: center;">
                ${codesHTML}
            </div>
            <br>
            <button type="button" style="padding: 0.75em 1em;" onclick="window.location.href='/';">I've saved my codes</button>
        </div>
    `;
}
