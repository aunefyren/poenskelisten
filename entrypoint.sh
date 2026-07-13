#!/bin/sh

# Start with the binary
set -- /app/poenskelisten

# Add the Pønskelisten environment variables if set
[ -n "$port" ] && set -- "$@" --port "$port"
[ -n "$externalurl" ] && set -- "$@" --externalurl "$externalurl"
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

# Enable the MCP resource server if the environment variable is set. The OAuth
# issuer/algorithm and API/MCP resource identifiers auto-derive from the external
# URL, so they need no environment variables.
[ -n "$mcpenabled" ] && set -- "$@" --mcpenabled "$mcpenabled"

# Add flags for invite generation if those environment variables are set
[ -n "$generateinvite" ] && set -- "$@" --generateinvite "$generateinvite"

# Add flags for SMTP settings if those environment variables are set
[ -n "$disablesmtp" ] && set -- "$@" --disablesmtp "$disablesmtp"
[ -n "$smtphost" ] && set -- "$@" --smtphost "$smtphost"
[ -n "$smtpport" ] && set -- "$@" --smtpport "$smtpport"
[ -n "$smtpusername" ] && set -- "$@" --smtpusername "$smtpusername"
[ -n "$smtppassword" ] && set -- "$@" --smtppassword "$smtppassword"
[ -n "$smtpfrom" ] && set -- "$@" --smtpfrom "$smtpfrom"

# Execute safely
exec "$@"
