#!/bin/sh

# Initialize the command with the binary
CMD="/app/poenskelisten"

# Add the --port flag if the PORT environment variable is set
if [ -n "$port" ]; then
  CMD="$CMD --port $port"
fi

# Add the --timezone flag if the TIMEZONE environment variable is set
if [ -n "$timezone" ]; then
  CMD="$CMD --timezone $timezone"
fi

# Add database-related flags if the corresponding environment variables are set
if [ -n "$dbip" ]; then
  CMD="$CMD --dbip $dbip"
fi

if [ -n "$dbport" ]; then
  CMD="$CMD --dbport $dbport"
fi

if [ -n "$dbname" ]; then
  CMD="$CMD --dbname $dbname"
fi

if [ -n "$dbusername" ]; then
  CMD="$CMD --dbusername $dbusername"
fi

if [ -n "$dbpassword" ]; then
  CMD="$CMD --dbpassword $dbpassword"
fi

# Add flags for invite generation and SMTP settings if those environment variables are set
if [ -n "$generateinvite" ]; then
  CMD="$CMD --generateinvite $generateinvite"
fi

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

# Execute the final command
exec $CMD