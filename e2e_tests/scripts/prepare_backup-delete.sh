#!/bin/sh

set -e

mc config host add ${S3_MINIO_HOSTNAME} http://minio:9000 ${MINIO_ROOT_USER} ${MINIO_ROOT_PASSWORD};
mc mb ${S3_MINIO_HOSTNAME}/${S3_MINIO_BUCKET};
mc admin user add ${S3_MINIO_HOSTNAME} ${S3_MINIO_KEY} ${S3_MINIO_KEY_SECRET};
mc admin policy attach ${S3_MINIO_HOSTNAME} readwrite --user ${S3_MINIO_KEY}

TIMESTAMP="20230724090000"
touch /tmp/test.txt
mc cp /tmp/test.txt ${S3_MINIO_HOSTNAME}/${S3_MINIO_BUCKET}/test/backups/${TIMESTAMP:0:8}/${TIMESTAMP}/test.txt

TIMESTAMPS="20230725101959 20230725102950 20230725102831"
for i in ${TIMESTAMPS}; do
    mc cp /tmp/test.txt ${S3_MINIO_HOSTNAME}/${S3_MINIO_BUCKET}/test/backups/${i:0:8}/${i}/test.txt
done

