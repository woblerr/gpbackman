#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="report-info"

HOME_DIR="/home/gpbackman"
SRC_DIR="${HOME_DIR}/src_data"
WORK_DIR="${HOME_DIR}/test_data"

# Prepare  general data.
rm -rf "${WORK_DIR}/"
mkdir -p "${WORK_DIR}"
cp ${SRC_DIR}/gpbackup_history_metadata_plugin.yaml \
${SRC_DIR}/gpbackup_history_full_local.yaml \
${SRC_DIR}/gpbackup_history.db \
${WORK_DIR}

################################################################
# Test 1.
# Get report info for specified backup with gpbackup_s3_plugin.
TEST_ID="1"

TIMESTAMP="20230724090000"

# Execute report-info commnad.
GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP} \
--plugin-config ${HOME_DIR}/gpbackup_s3_plugin.yaml | grep -v 'Reading Plugin Config')

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
bckp_report=$(cat ${SRC_DIR}/gpbackup_${TIMESTAMP}_report)
if [ "${bckp_report}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nbckp_report:\n${bckp_report}\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqlite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."

################################################################
# Test 2.
# Get report info for specified local backup with specifying backup directory without single-backup-dir format.
# Set backup directory from console.
TEST_ID="2"

TIMESTAMP="20240505201504"
BACKUP_DIR="/tmp/testWithPrefix"
REPORT_DIR="${BACKUP_DIR}/segment-1/backups/${TIMESTAMP:0:8}/${TIMESTAMP}"
# Prepare data.
mkdir -p ${REPORT_DIR}

cp ${SRC_DIR}/gpbackup_${TIMESTAMP}_report ${REPORT_DIR}

# Execute report-info commnad.
GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP} \
--backup-dir ${BACKUP_DIR})

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
bckp_report=$(cat ${SRC_DIR}/gpbackup_${TIMESTAMP}_report)
if [ "${bckp_report}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nbckp_report:\n${bckp_report}\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqlite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."


################################################################
# Test 3.
# Get report info for specified local backup with specifying backup directory without single-backup-dir format.
# Set backup directory from history database.
TEST_ID="3"

TIMESTAMP="20240505201504"
BACKUP_DIR="/tmp/testWithPrefix"
REPORT_DIR="${BACKUP_DIR}/segment-1/backups/${TIMESTAMP:0:8}/${TIMESTAMP}"
# Prepare data.
mkdir -p ${REPORT_DIR}

cp ${SRC_DIR}/gpbackup_${TIMESTAMP}_report ${REPORT_DIR}

# Execute report-info commnad.
GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP})

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
bckp_report=$(cat ${SRC_DIR}/gpbackup_${TIMESTAMP}_report)
if [ "${bckp_report}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nbckp_report:\n${bckp_report}\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqlite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."


################################################################
# Test 4s.
# Get report info for specified local backup with specifying backup directory with single-backup-dir format.
# Set backup directory from console.
TEST_ID="4"

TIMESTAMP="20240506201504"
BACKUP_DIR="/tmp/testNoPrefix"
REPORT_DIR="${BACKUP_DIR}/backups/${TIMESTAMP:0:8}/${TIMESTAMP}"
# Prepare data.
mkdir -p ${REPORT_DIR}

cp ${SRC_DIR}/gpbackup_${TIMESTAMP}_report ${REPORT_DIR}

# Execute report-info commnad.
GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP} \
--backup-dir ${BACKUP_DIR})

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
bckp_report=$(cat ${SRC_DIR}/gpbackup_${TIMESTAMP}_report)
if [ "${bckp_report}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nbckp_report:\n${bckp_report}\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqlite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."


################################################################
# Test 4.
# Get report info for specified local backup with specifying backup directory with single-backup-dir format.
# Set backup directory from history database.
TEST_ID="5"

TIMESTAMP="20240506201504"
BACKUP_DIR="/tmp/testNoPrefix"
REPORT_DIR="${BACKUP_DIR}/backups/${TIMESTAMP:0:8}/${TIMESTAMP}"
# Prepare data.
mkdir -p ${REPORT_DIR}

cp ${SRC_DIR}/gpbackup_${TIMESTAMP}_report ${REPORT_DIR}

# Execute report-info commnad.
GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${WORK_DIR}/gpbackup_history.db \
--timestamp ${TIMESTAMP})

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID}."
bckp_report=$(cat ${SRC_DIR}/gpbackup_${TIMESTAMP}_report)
if [ "${bckp_report}" != "${GPBACKMAN_RESULT_SQLITE}" ]; then
    echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} failed.\nbckp_report:\n${bckp_report}\nget_yaml:\n${GPBACKMAN_RESULT_YAML}\nget_sqlite:\n${GPBACKMAN_RESULT_SQLITE}"
    exit 1
fi
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test ${TEST_ID} passed."

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} all tests passed"
exit 0
