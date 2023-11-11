#!/bin/sh

uid=$(id -u)

if [ "${uid}" = "0" ]; then
    # Custom time zone.
    if [ "${TZ}" != "Etc/UTC" ]; then
        cp /usr/share/zoneinfo/${TZ} /etc/localtime
        echo "${TZ}" > /etc/timezone
    fi

    # Set custom user UID or GID.
    if  [ "${GPBACKMAN_UID}" != "1001" ] || [ "${GPBACKMAN_GID}" != "1001" ] ; then
        sed -i "s/:1001:1001:/:${GPBACKMAN_UID}:${GPBACKMAN_GID}:/g" /etc/passwd
    fi
    chown -R ${GPBACKMAN_USER}:${GPBACKMAN_USER} /home/${GPBACKMAN_USER} 
fi

if [ "${uid}" = "0" ]; then
    exec su-exec ${GPBACKMAN_USER} "$@"
else
    exec "$@"
fi
