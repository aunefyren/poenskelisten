#!/bin/sh

# A first argument that isn't a flag is a command to run instead of the app
# (e.g. `docker run -it <image> sh` for debugging).
if [ -n "$1" ] && [ "${1#-}" = "$1" ]; then
    exec "$@"
fi

# Any arguments passed to the container are Pønskelisten flags. Remember how
# many there are so they can be moved after the environment-derived flags
# below: Go's flag package keeps the last value given, so explicit arguments
# win over environment variables.
argCount=$#

# Add the Pønskelisten environment variables if set
[ -n "$port" ] && set -- "$@" --port "$port"
[ -n "$externalurl" ] && set -- "$@" --externalurl "$externalurl"
[ -n "$additionalurls" ] && set -- "$@" --additionalurls "$additionalurls"
[ -n "$timezone" ] && set -- "$@" --timezone "$timezone"
[ -n "$environment" ] && set -- "$@" --environment "$environment"
[ -n "$testemail" ] && set -- "$@" --testemail "$testemail"
[ -n "$name" ] && set -- "$@" --name "$name"
[ -n "$description" ] && set -- "$@" --description "$description"
[ -n "$loglevel" ] && set -- "$@" --loglevel "$loglevel"

# Add database-related flags if the corresponding environment variables are set
[ -n "$dbport" ] && set -- "$@" --dbport "$dbport"
[ -n "$dbtype" ] && set -- "$@" --dbtype "$dbtype"
[ -n "$dbusername" ] && set -- "$@" --dbusername "$dbusername"
[ -n "$dbpassword" ] && set -- "$@" --dbpassword "$dbpassword"
[ -n "$dbname" ] && set -- "$@" --dbname "$dbname"
[ -n "$dbip" ] && set -- "$@" --dbip "$dbip"
[ -n "$dbssl" ] && set -- "$@" --dbssl "$dbssl"
[ -n "$dblocation" ] && set -- "$@" --dblocation "$dblocation"

# Add security-related flags if those environment variables are set
[ -n "$mfaenforced" ] && set -- "$@" --mfaenforced "$mfaenforced"
[ -n "$mfarecoverycodes" ] && set -- "$@" --mfarecoverycodes "$mfarecoverycodes"

# Add OIDC single sign-on flags if those environment variables are set
[ -n "$oidcenabled" ] && set -- "$@" --oidcenabled "$oidcenabled"
[ -n "$oidcprovidername" ] && set -- "$@" --oidcprovidername "$oidcprovidername"
[ -n "$oidcissuerurl" ] && set -- "$@" --oidcissuerurl "$oidcissuerurl"
[ -n "$oidcclientid" ] && set -- "$@" --oidcclientid "$oidcclientid"
[ -n "$oidcclientsecret" ] && set -- "$@" --oidcclientsecret "$oidcclientsecret"
[ -n "$oidcredirecturl" ] && set -- "$@" --oidcredirecturl "$oidcredirecturl"
[ -n "$oidcautocreateusers" ] && set -- "$@" --oidcautocreateusers "$oidcautocreateusers"
[ -n "$disablelocallogin" ] && set -- "$@" --disablelocallogin "$disablelocallogin"

# Enable the MCP resource server if the environment variable is set. The OAuth
# issuer/algorithm and API/MCP resource identifiers auto-derive from the external
# URL, so they need no environment variables.
[ -n "$mcpenabled" ] && set -- "$@" --mcpenabled "$mcpenabled"

# Add flags for invite generation if those environment variables are set
[ -n "$generateinvite" ] && set -- "$@" --generateinvite "$generateinvite"

# Account recovery; see "Recovering an account from the server" in README.md
[ -n "$resetpassword" ] && set -- "$@" --resetpassword "$resetpassword"
[ -n "$resetmfa" ] && set -- "$@" --resetmfa "$resetmfa"

# Add flags for SMTP settings if those environment variables are set
[ -n "$disablesmtp" ] && set -- "$@" --disablesmtp "$disablesmtp"
[ -n "$smtphost" ] && set -- "$@" --smtphost "$smtphost"
[ -n "$smtpport" ] && set -- "$@" --smtpport "$smtpport"
[ -n "$smtpusername" ] && set -- "$@" --smtpusername "$smtpusername"
[ -n "$smtppassword" ] && set -- "$@" --smtppassword "$smtppassword"
[ -n "$smtpfrom" ] && set -- "$@" --smtpfrom "$smtpfrom"

# Rotate the container's own arguments from the front to the end, then prefix
# the binary.
i=0
while [ "$i" -lt "$argCount" ]; do
    set -- "$@" "$1"
    shift
    i=$((i + 1))
done
set -- /app/poenskelisten "$@"

# Started as root (the default): make the writable data directories belong to
# PUID/PGID and drop to that uid/gid. Only entries with the wrong owner are
# changed, so restarts with a large image library stay fast. A bind-mounted
# directory Docker created on the host is root-owned, so without this the app
# couldn't write its config, database, log or uploaded images.
# Started as non-root (e.g. `user: "1000:1000"` in compose): run as-is.
if [ "$(id -u)" = "0" ]; then
    PUID=${PUID:-1000}
    PGID=${PGID:-1000}
    case "$PUID$PGID" in
        *[!0-9]*)
            echo "PUID and PGID must be numeric (got PUID='$PUID' PGID='$PGID')" >&2
            exit 1
            ;;
    esac
    mkdir -p /app/files /app/images
    find /app/files /app/images \( ! -user "$PUID" -o ! -group "$PGID" \) -exec chown "$PUID:$PGID" {} +
    exec su-exec "$PUID:$PGID" "$@"
fi

exec "$@"
