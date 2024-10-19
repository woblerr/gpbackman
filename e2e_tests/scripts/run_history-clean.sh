#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="history-clean"

SRC_DIR="/home/gpbackman/src_data"
WORK_DIR="/home/gpbackman/test_data"

DATE_REGEX="(Mon|Tue|Wed|Thu|Fri|Sat|Sun)\s(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s(0[1-9]|[12][0-9]|3[01])\s[0-9]{4}\s(0[0-9]|1[0-9]|2[0-3]):(0[0-9]|[1-5][0-9]):(0[0-9]|[1-5][0-9])"

# Prepare data.
rm -rf "${WORK_DIR}/"
mkdir -p "${WORK_DIR}"
cp ${SRC_DIR}/gpbackup_history_failure_plugin.yaml \
${SRC_DIR}/gpbackup_history_incremental_plugin.yaml \
${SRC_DIR}/gpbackup_history.db \
${WORK_DIR}

################################################################
# Test 1.
# Delete backups from history database older than timestamp.
# There are no failed or deleted backups after command execution.
TEST_ID="1"

TIMESTAMP="20231212101500"

# Execute history-clean commnad.

gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--before-timestamp ${TIMESTAMP} \

GPBACKMAN_RESULT_SQLITE=$(gpbackman backup-info \
--history-db ${WORK_DIR}/gpbackup_history.db \
--deleted --failed)

TEST_CNT=0

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | cut -f9 -d'|' | awk '{$1=$1};1' | grep -E ${DATE_REGEX} | wc -l)
if [ "${result_cnt_sqlite}" != "${TEST_CNT}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nget_yaml=${result_cnt_yaml}, get_sqlite=${result_cnt_sqlite}, want=${TEST_CNT}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} all tests passed"
exit 0
