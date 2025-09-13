#!/bin/sh
set -eu

mc config host add ${S3_MINIO_HOSTNAME} http://minio:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD};
mc mb ${S3_MINIO_HOSTNAME}/${S3_MINIO_BUCKET};
mc admin user add ${S3_MINIO_HOSTNAME} ${S3_MINIO_KEY} ${S3_MINIO_KEY_SECRET};
mc admin policy attach ${S3_MINIO_HOSTNAME} readwrite --user ${S3_MINIO_KEY}
