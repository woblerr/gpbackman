#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="backup-delete"

HOME_DIR="/home/gpbackman"
SRC_DIR="${HOME_DIR}/src_data"
WORK_DIR="${HOME_DIR}/test_data"

DATE_REGEX="(Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s(0[1-9]|[12][0-9]|3[01])\s[0-9]{4}\s(0[0-9]|1[0-9]|2[0-3]):(0[0-9]|[1-5][0-9]):(0[0-9]|[1-5][0-9])"
TIMESTAMP=""

# Prepare data.
rm -rf "${WORK_DIR}/"
mkdir -p "${WORK_DIR}"
cp ${SRC_DIR}/gpbackup_history_metadata_plugin.yaml \
${SRC_DIR}/gpbackup_history.db \
${WORK_DIR}

################################################################
# Test 1.
# All ther calls are executed for the same timestamp.
# At the first call, the backup is deleted from the s3.
# The yaml history file format is used.
# History file yaml format is used, there are a real s3 call and a real backup deletion.

# At secod call, there are a real s3 call and no real backup deletion.
# The sqlite history file format is used.
# Because this backup was deleted in first call, there are no files in the s3.
# But the info about deletion attempt is written to log file and DATE DELETED is updated in history file.
TEST_ID="1"

TIMESTAMP="20230724090000"

# Execute backup-delete commnad.
gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-file ${WORK_DIR}/gpbackup_history_metadata_plugin.yaml \
--timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml

gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml

GPBACKMAN_RESULT_YAML=$(gpbackman backup-info \
--history-file ${WORK_DIR}/gpbackup_history_metadata_plugin.yaml \
--deleted | grep -w ${TIMESTAMP})

GPBACKMAN_RESULT_SQLITE=$(gpbackman backup-info \
--history-db ${WORK_DIR}/gpbackup_history.db \
--deleted | grep -w ${TIMESTAMP})

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
bckp_date_deleted=$(echo "${GPBACKMAN_RESULT_YAML}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX})
if [ $? != 0 ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nget_yaml:\n${bckp_date_deleted}"
    exit 1
fi
bckp_date_deleted=$(echo "${GPBACKMAN_RESULT_SQLITE}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX})
if [ $? != 0 ]; then
    echo -e "[ERROR] r${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nget_sqlite:\n${bckp_date_deleted}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."

################################################################
# Test 2.
# Test cascade delete option
TEST_ID="2"

TIMESTAMP="20230725101959"
# After successful delete, in history there should be 5 backup with dete deleted info.
# 2 from source + 1 from test 1 + 3 from this test.
TEST_CNT=6

# Execute backup-delete commnad.
gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml \
--cascade

GPBACKMAN_RESULT_SQLITE=$(gpbackman backup-info \
--history-db ${WORK_DIR}/gpbackup_history.db \
--deleted)

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX} | wc -l)
if [ "${result_cnt_sqlite}" != "${TEST_CNT}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nget_sqlite=${result_cnt_sqlite}, want=${TEST_CNT}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."

################################################################
# Test 3.
# Test errors in logs.
TEST_ID="3"

TIMESTAMP="20230725101959"
TEST_CNT=5

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
logs_errors=$(grep -r ERROR ${HOME_DIR}/gpAdminLogs/)
if [ $? == 0 ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nget_logs:\n${logs_errors}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} all tests passed"
exit 0
