#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="report-info"

HOME_DIR="/home/gpbackman"
SRC_DIR="${HOME_DIR}/src_data"
WORK_DIR="${HOME_DIR}/test_data"

TIMESTAMP="20230724090000"

# Prepare data.
rm -rf "${WORK_DIR}/"
mkdir -p "${WORK_DIR}"
cp ${SRC_DIR}/gpbackup_history_metadata_plugin.yaml \
${SRC_DIR}/gpbackup_history.db \
${WORK_DIR}

################################################################
# Test 1.
# Get report info for specified backup.

# Execute report-info commnad.
GPBACKMAN_RESULT_YAML=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-file ${WORK_DIR}/gpbackup_history_metadata_plugin.yaml \
--timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml | grep -v 'Reading Plugin Config')

GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml | grep -v 'Reading Plugin Config')

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1."
bckp_report=$(cat ${SRC_DIR}/gpbackup_${TIMESTAMP}_report)
if [ "${bckp_report}" != "${GPBACKMAN_RESULT_YAML}" ] ||  [ "${bckp_report}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] results do not match.\nbckp_report:\n${bckp_report}\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqlite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1 passed."

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} all tests passed"
exit 0
