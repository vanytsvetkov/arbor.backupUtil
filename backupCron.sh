#!/bin/ksh

# Define constants for directory paths
CONF_BACKUP_DIR="/base/data/files/config_backups"
# CONFDIR="/base/store/conf"
# not define because that already exists in env
GIT_SCRIPT_DIR="/base/data/files/scripts"
BACKUP_UTIL_SCRIPT="$GIT_SCRIPT_DIR/backupUtil"

# Create the backup directory if it doesn't exist
if [ ! -d "$CONF_BACKUP_DIR" ]; then
  mkdir -p "$CONF_BACKUP_DIR" || exit 0
fi

# Combine all *.conf files, sort and remove the prefix of the form "XXX.YYYYYY "
cat "$CONFDIR"/*.conf | sort -n | sed 's;^[0-9]\{3\}\.[0-9]\{6\} ;;' > "$CONF_BACKUP_DIR/$(hostname).cfg"

# Echo the commit log to the working directory
echo "*** empty log message ***" > "$CONF_BACKUP_DIR/COMMIT_EDITMSG"

# Check if backupUtil script exists before starting it
if [ -x "$BACKUP_UTIL_SCRIPT" ]; then
  "$BACKUP_UTIL_SCRIPT" > /dev/null 2>&1
fi
