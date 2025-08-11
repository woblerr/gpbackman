#!/usr/bin/env bash

uid=$(id -u)

if [ "${uid}" = "0" ]; then
    # Custom time zone.
    if [ "${TZ}" != "Etc/UTC" ]; then
        cp /usr/share/zoneinfo/${TZ} /etc/localtime
        echo "${TZ}" > /etc/timezone
    fi
    # Custom user group.
    if [ "${GPBACKMAN_GROUP}" != "gpbackman" ] || [ "${GPBACKMAN_GID}" != "1001" ]; then
        groupmod -g ${GPBACKMAN_GID} -n ${GPBACKMAN_GROUP} gpbackman
    fi
    # Custom user.
    if [ "${GPBACKMAN_USER}" != "gpbackman" ] || [ "${GPBACKMAN_UID}" != "1001" ]; then
        usermod -g ${GPBACKMAN_GID} -l ${GPBACKMAN_USER} -u ${GPBACKMAN_UID} -m -d /home/${GPBACKMAN_USER} gpbackman
    fi
    chown -R ${GPBACKMAN_USER}:${GPBACKMAN_GROUP} /home/${GPBACKMAN_USER} 
fi

if [ "${uid}" = "0" ]; then
    exec gosu ${GPBACKMAN_USER} "$@"
else
    exec "$@"
fi
