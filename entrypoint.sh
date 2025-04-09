#!/bin/sh

# Initialize the command with the binary
CMD="/app/poenskelisten"

# Add the Pønskelisten environment variables if set
if [ -n "$port" ]; then
  CMD="$CMD --port $port"
fi

if [ -n "$externalurl" ]; then
  CMD="$CMD --externalurl $externalurl"
fi

if [ -n "$timezone" ]; then
  CMD="$CMD --timezone $timezone"
fi

if [ -n "$environment" ]; then
  CMD="$CMD --environment $environment"
fi

if [ -n "$testemail" ]; then
  CMD="$CMD --testemail $testemail"
fi

if [ -n "$name" ]; then
  CMD="$CMD --name $name"
fi

if [ -n "$loglevel" ]; then
  CMD="$CMD --loglevel $loglevel"
fi

# Add database-related flags if the corresponding environment variables are set
if [ -n "$dbport" ]; then
  CMD="$CMD --dbport $dbport"
fi

if [ -n "$dbtype" ]; then
  CMD="$CMD --dbtype $dbtype"
fi

if [ -n "$dbusername" ]; then
  CMD="$CMD --dbusername $dbusername"
fi

if [ -n "$dbpassword" ]; then
  CMD="$CMD --dbpassword $dbpassword"
fi

if [ -n "$dbname" ]; then
  CMD="$CMD --dbname $dbname"
fi

if [ -n "$dbip" ]; then
  CMD="$CMD --dbip $dbip"
fi

if [ -n "$dbssl" ]; then
  CMD="$CMD --dbssl $dbssl"
fi

if [ -n "$dblocation" ]; then
  CMD="$CMD --dblocation $dblocation"
fi

# Add flags for invite generation if those environment variables are set
if [ -n "$generateinvite" ]; then
  CMD="$CMD --generateinvite $generateinvite"
fi

# Add flags for SMTP settings if those environment variables are set
if [ -n "$disablesmtp" ]; then
  CMD="$CMD --disablesmtp $disablesmtp"
fi

if [ -n "$smtphost" ]; then
  CMD="$CMD --smtphost $smtphost"
fi

if [ -n "$smtpport" ]; then
  CMD="$CMD --smtpport $smtpport"
fi

if [ -n "$smtpusername" ]; then
  CMD="$CMD --smtpusername $smtpusername"
fi

if [ -n "$smtppassword" ]; then
  CMD="$CMD --smtppassword $smtppassword"
fi

if [ -n "$smtpfrom" ]; then
  CMD="$CMD --smtpfrom $smtpfrom"
fi

# Execute the final command
exec $CMD