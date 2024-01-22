#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="backup-clean"

HOME_DIR="/home/gpbackman"
SRC_DIR="${HOME_DIR}/src_data"
WORK_DIR="${HOME_DIR}/test_data"

DATE_REGEX="(Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s(0[1-9]|[12][0-9]|3[01])\s[0-9]{4}\s(0[0-9]|1[0-9]|2[0-3]):(0[0-9]|[1-5][0-9]):(0[0-9]|[1-5][0-9])"
TIMESTAMP=""

# Prepare data.
rm -rf "${WORK_DIR}/"
mkdir -p "${WORK_DIR}"
cp ${SRC_DIR}/gpbackup_history_incremental_plugin.yaml \
${SRC_DIR}/gpbackup_history.db \
${WORK_DIR}

################################################################
# Test 1.
# Delete all backups older than timestamp.
# Because other backup are incermental and we don't use the option --cascade, no backup will be deleted.

TIMESTAMP="20230725101500"

# Execute backup-delete commnad.


gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--before-timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml

GPBACKMAN_RESULT_SQLITE=$(gpbackman backup-info \
--history-db ${WORK_DIR}/gpbackup_history.db \
--show-deleted)

TEST_CNT_SQL=1

# Check results.
# In sql db there is one predifined deleted backup - 20230725110310.
# So, it's ok that one deleted backup exists.
result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX} | wc -l)
if [ "${result_cnt_sqlite}" != "${TEST_CNT_SQL}" ]; then
    echo -e "[ERROR] on cascade delition test.\nget_sqlite=${result_cnt_sqlite}, want=${TEST_CNT_SQL}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1 passed."

################################################################
# Test 2.
# Test cascade delete option.
# All backupd older than timestamp should be deleted.

gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-file ${WORK_DIR}/gpbackup_history_incremental_plugin.yaml \
--before-timestamp  ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml \
--cascade

gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--before-timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml \
--cascade

GPBACKMAN_RESULT_YAML=$(gpbackman backup-info \
--history-file ${WORK_DIR}/gpbackup_history_incremental_plugin.yaml \
--show-deleted)

GPBACKMAN_RESULT_SQLITE=$(gpbackman backup-info \
--history-db ${WORK_DIR}/gpbackup_history.db \
--show-deleted)

# After successful delete, in history there should be 11 fo sql and 7 for yaml backup with date deleted info.
TEST_CNT_YAML=7
TEST_CNT_SQL=11


result_cnt_yaml=$(echo "${GPBACKMAN_RESULT_YAML}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX} | wc -l)
if [ "${result_cnt_yaml}" != "${TEST_CNT_YAML}" ]; then
    echo -e "[ERROR] on cascade delition test.\nget_yaml=${result_cnt_yaml}, want=${TEST_CNT_YAML}"
    exit 1
fi

result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX} | wc -l)
if [ "${result_cnt_sqlite}" != "${TEST_CNT_SQL}" ]; then
    echo -e "[ERROR] on cascade delition test.\nget_sqlite=${result_cnt_sqlite}, want=${TEST_CNT_SQL}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 2 passed."
