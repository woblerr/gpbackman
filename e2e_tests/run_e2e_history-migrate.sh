#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="history-migrate"

SRC_DIR="/home/gpbackman/e2e_tests/src_data"
WORK_DIR="/home/gpbackman/test_data"

# Prepare data.
rm -rf "${WORK_DIR}/"
mkdir -p "${WORK_DIR}"
cp "${SRC_DIR}/gpbackup_history_dataonly_nodata_plugin.yaml" \
"${SRC_DIR}/gpbackup_history_metadata_plugin.yaml" \
"${WORK_DIR}"

# Execute history-migrate commnad.
gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-file ${WORK_DIR}/gpbackup_history_dataonly_nodata_plugin.yaml \
--history-file ${WORK_DIR}/gpbackup_history_metadata_plugin.yaml \
--history-db ${WORK_DIR}/gpbackup_history.db

################################################################
# Test 1.
# Check that in source data there are files with .migrated type after migration.
# Format:
#   source_file.megrated.
REGEX_LIST='''gpbackup_history_dataonly_nodata_plugin.yaml.migrated
gpbackup_history_metadata_plugin.yaml.migrated
'''

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1."
for i in ${REGEX_LIST}
do
    if [ ! -f "${WORK_DIR}/${i}" ]; then
        echo "[ERROR] file ${i} not found."
        exit 1
    fi
done
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1 passed."

################################################################
# Test 2.
# Compare results of backup-info command before and after migration.
GPBACKMAN_RESULT_YAML=$(gpbackman backup-info \
--history-file ${WORK_DIR}/gpbackup_history_dataonly_nodata_plugin.yaml.migrated \
--history-file ${WORK_DIR}/gpbackup_history_metadata_plugin.yaml.migrated \
--show-deleted \
--show-failed)

# backup-info commnad for sqllite backup history format.
# This result from migrated data.
GPBACKMAN_RESULT_SQLITE=$(gpbackman backup-info \
--history-db ${WORK_DIR}/gpbackup_history.db \
--show-deleted \
--show-failed)

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 2."
if [ "${GPBACKMAN_RESULT_YAML}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] results before and after migration do not match.\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqllite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 2 passed."

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} all tests passed"
exit 0
