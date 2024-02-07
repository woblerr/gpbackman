#!/bin/sh

# Local image for e2e tests should be built before running tests.
# See make file for details.
# This test works with files from src_data directory.
# If new file with backup info is added to src_data, it's nessary to update test cases in this script.

GPBACKMAN_TEST_COMMAND="backup-info"

SRC_DIR="/home/gpbackman/src_data"

# backup-info commnad for yaml backup history format.
GPBACKMAN_RESULT_YAML=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-file ${SRC_DIR}/gpbackup_history_dataonly_nodata_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_dataonly_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_failure_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_full_local.yaml \
--history-file ${SRC_DIR}/gpbackup_history_full_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_incremental_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_metadata_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_incremental_include_schemas_plugin.yaml \
--history-file ${SRC_DIR}/gpbackup_history_incremental_include_tables_plugin.yaml \
--deleted \
--failed)

# backup-info commnad for sqlite backup history format.
GPBACKMAN_RESULT_SQLITE=$(gpbackman ${GPBACKMAN_TEST_COMMAND} \
--history-db ${SRC_DIR}/gpbackup_history.db \
--deleted \
--failed)

IFS=$'\n'
################################################################
# Test 1.
# Simple test to check the number of provided backups.
# Format:
#   status | type | object filtering| plugin | date deleted | repetitions.
# For backup without plugin info - blank line, so them skips in this test.
REGEX_LIST='''Success|data-only|gpbackup_s3_plugin|1
Success|metadata-only|gpbackup_s3_plugin|2
Success|full|gpbackup_s3_plugin|4
Failure|full|gpbackup_s3_plugin|3
Success|incremental|gpbackup_s3_plugin|9'''

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1."
for i in ${REGEX_LIST}
do
    bckp_status=$(echo "${i}" | cut -f1 -d'|')
    bckp_type=$(echo "${i}" | cut -f2 -d'|')
    bckp_plugin=$(echo "${i}" | cut -f3 -d'|')
    cnt=$(echo "${i}" | cut -f4 -d'|')
    result_cnt_yaml=$(echo "${GPBACKMAN_RESULT_YAML}" | grep -w "${bckp_status}" | grep -w "${bckp_type}" | grep -w "${bckp_plugin}" | wc -l | tr -d ' ')
    result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | grep -w "${bckp_status}" | grep -w "${bckp_type}" | grep -w "${bckp_plugin}" | wc -l | tr -d ' ')
    if [ "${result_cnt_yaml}" != "${cnt}" ] || [ "${result_cnt_sqlite}" != "${cnt}" ]; then\
        echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test 1 failed.\n'${i}': get_yaml=${result_cnt_yaml}, get_sqlite=${result_cnt_sqlite}, want=${cnt}"
        exit 1
    fi
done
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 1 passed."

################################################################
# Test 2.
# Simple test to check full info about backups.
# Format:
#  timestamp| date | status | database| type| plugin | duration | repetitions.
# The match of all fields in the backup information is checked.
# Don't test backup with empty object filtering, plugin info and non-empty dete deleted fields.
REGEX_LIST='''20230806230400|Sun Aug 06 2023 23:04:00|Failure|demo|full|gpbackup_s3_plugin|00:00:38|1
20230725102950|Tue Jul 25 2023 10:29:50|Success|demo|incremental|gpbackup_s3_plugin|00:00:19|1
20230725110051|Tue Jul 25 2023 11:00:51|Success|demo|incremental|gpbackup_s3_plugin|00:00:20|1
20230725102831|Tue Jul 25 2023 10:28:31|Success|demo|incremental|gpbackup_s3_plugin|00:00:18|1
20230725101959|Tue Jul 25 2023 10:19:59|Success|demo|incremental|gpbackup_s3_plugin|00:00:22|1
20230725101152|Tue Jul 25 2023 10:11:52|Success|demo|incremental|gpbackup_s3_plugin|00:00:18|1
20230725101115|Tue Jul 25 2023 10:11:15|Success|demo|full|gpbackup_s3_plugin|00:00:20|1
20230724090000|Mon Jul 24 2023 09:00:00|Success|demo|metadata-only|gpbackup_s3_plugin|00:05:17|1
20230723082000|Sun Jul 23 2023 08:20:00|Success|demo|data-only|gpbackup_s3_plugin|00:35:17|1
20230722100000|Sat Jul 22 2023 10:00:00|Success|demo|full|gpbackup_s3_plugin|00:25:17|1
20230721090000|Fri Jul 21 2023 09:00:00|Success|demo|metadata-only|gpbackup_s3_plugin|00:04:17|1'''

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 2."
for i in ${REGEX_LIST}
do
    bckp_timestamp=$(echo "${i}" | cut -f1 -d'|')
    bckp_date=$(echo "${i}" | cut -f2 -d'|')
    bckp_status=$(echo "${i}" | cut -f3 -d'|')
    bckp_database=$(echo "${i}" | cut -f4 -d'|')
    bckp_type=$(echo "${i}" | cut -f5 -d'|')
    bckp_plugin=$(echo "${i}" | cut -f6 -d'|')
    bckp_duration=$(echo "${i}" | cut -f7 -d'|')
    cnt=$(echo "${i}" | cut -f8 -d'|')
    result_cnt_yaml=$(echo "${GPBACKMAN_RESULT_YAML}" | \
        grep -w "${bckp_timestamp}" | \
        grep -w "${bckp_date}" | \
        grep -w "${bckp_status}" | \
        grep -w "${bckp_database}" | \
        grep -w "${bckp_type}" | \
        grep -w "${bckp_plugin}" | \
        grep -w "${bckp_duration}" | \
        wc -l | tr -d ' ')
    result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | \
        grep -w "${bckp_timestamp}" | \
        grep -w "${bckp_date}" | \
        grep -w "${bckp_status}" | \
        grep -w "${bckp_database}" | \
        grep -w "${bckp_type}" | \
        grep -w "${bckp_plugin}" | \
        grep -w "${bckp_duration}" | \
        wc -l | tr -d ' ')
    if [ "${result_cnt_yaml}" != "${cnt}" ] || [ "${result_cnt_sqlite}" != "${cnt}" ]; then
        echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test 2 failed.\n'${i}': get_yaml=${result_cnt_yaml}, get_sqlite=${result_cnt_sqlite}, want=${cnt}"
        exit 1
    fi
done
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 2 passed."

################################################################
# Test 3.
# Simple test to check full info about backups with deleted field.
# Format:
#  timestamp| date | status | database| type | plugin | duration | date deleted | repetitions.
# The match of all fields in the backup information is checked.
# Don't test backup with empty object filtering field.
REGEX_LIST="20230725110310|Tue Jul 25 2023 11:03:10|Success|demo|incremental|gpbackup_s3_plugin|00:00:18|Wed Jul 26 2023 11:03:28|1"

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 3."
for i in ${REGEX_LIST}
do
    bckp_timestamp=$(echo "${i}" | cut -f1 -d'|')
    bckp_date=$(echo "${i}" | cut -f2 -d'|')
    bckp_status=$(echo "${i}" | cut -f3 -d'|')
    bckp_database=$(echo "${i}" | cut -f4 -d'|')
    bckp_type=$(echo "${i}" | cut -f5 -d'|')
    bckp_plugin=$(echo "${i}" | cut -f6 -d'|')
    bckp_duration=$(echo "${i}" | cut -f7 -d'|')
    bckp_date_deleted=$(echo "${i}" | cut -f8 -d'|')
    cnt=$(echo "${i}" | cut -f9 -d'|')
    result_cnt_yaml=$(echo "${GPBACKMAN_RESULT_YAML}" | \
        grep -w "${bckp_timestamp}" | \
        grep -w "${bckp_date}" | \
        grep -w "${bckp_status}" | \
        grep -w "${bckp_database}" | \
        grep -w "${bckp_type}" | \
        grep -w "${bckp_plugin}" | \
        grep -w "${bckp_duration}" | \
        grep -w "${bckp_date_deleted}" | \
        wc -l | tr -d ' ')
    result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | \
        grep -w "${bckp_timestamp}" | \
        grep -w "${bckp_date}" | \
        grep -w "${bckp_status}" | \
        grep -w "${bckp_database}" | \
        grep -w "${bckp_type}" | \
        grep -w "${bckp_plugin}" | \
        grep -w "${bckp_duration}" | \
        grep -w "${bckp_date_deleted}" | \
        wc -l | tr -d ' ')
    if [ "${result_cnt_yaml}" != "${cnt}" ] || [ "${result_cnt_sqlite}" != "${cnt}" ]; then
        echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test 3 failed.\n'${i}': get_yaml=${result_cnt_yaml}, get_sqlite=${result_cnt_sqlite}, want=${cnt}"
        exit 1
    fi
done
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 3 passed."

################################################################
# Test 4.
# Simple test to check full info about local backups.
# Format:
#  timestamp| date | status | database| type| duration | repetitions.
# The match of all fields in the backup information is checked.
# Don't test backup with empty object filtering and date deleted fields.
# For local backups plugin field is empty.
REGEX_LIST="20230809232817|Wed Aug 09 2023 23:28:17|Success|demo|full|04:00:03|1"

# Check results.
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 4."
for i in ${REGEX_LIST}
do
    bckp_timestamp=$(echo "${i}" | cut -f1 -d'|')
    bckp_date=$(echo "${i}" | cut -f2 -d'|')
    bckp_status=$(echo "${i}" | cut -f3 -d'|')
    bckp_database=$(echo "${i}" | cut -f4 -d'|')
    bckp_type=$(echo "${i}" | cut -f5 -d'|')
    bckp_duration=$(echo "${i}" | cut -f6 -d'|')
    cnt=$(echo "${i}" | cut -f7 -d'|')
    result_cnt_yaml=$(echo "${GPBACKMAN_RESULT_YAML}" | \
        grep -w "${bckp_timestamp}" | \
        grep -w "${bckp_date}" | \
        grep -w "${bckp_status}" | \
        grep -w "${bckp_database}" | \
        grep -w "${bckp_type}" | \
        grep -w "${bckp_duration}" | \
        wc -l | tr -d ' ')
    result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | \
        grep -w "${bckp_timestamp}" | \
        grep -w "${bckp_date}" | \
        grep -w "${bckp_status}" | \
        grep -w "${bckp_database}" | \
        grep -w "${bckp_type}" | \
        grep -w "${bckp_duration}" | \
        wc -l | tr -d ' ')
    if [ "${result_cnt_yaml}" != "${cnt}" ] || [ "${result_cnt_sqlite}" != "${cnt}" ]; then
        echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test 4 failed.\n'${i}': get_yaml=${result_cnt_yaml}, get_sqlite=${result_cnt_sqlite}, want=${cnt}"
        exit 1
    fi
done
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 4 passed."

################################################################
# Test 5.
# Simple test to check type iption
# Format:
#  status | type| repetitions.
#   status | type | object filtering| plugin | date deleted | repetitions.
# For backup without plugin info - blank line, so them skips in this test.
REGEX_LIST='''Success|full|5
Failure|full|3'''

echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 5."
for i in ${REGEX_LIST}
do
    bckp_status=$(echo "${i}" | cut -f1 -d'|')
    bckp_type=$(echo "${i}" | cut -f2 -d'|')
    cnt=$(echo "${i}" | cut -f3 -d'|')
    result_cnt_yaml=$(echo "${GPBACKMAN_RESULT_YAML}" | grep -w "${bckp_status}" | grep -w "${bckp_type}" | wc -l | tr -d ' ')
    result_cnt_sqlite=$(echo "${GPBACKMAN_RESULT_SQLITE}" | grep -w "${bckp_status}" | grep -w "${bckp_type}" | wc -l | tr -d ' ')
    if [ "${result_cnt_yaml}" != "${cnt}" ] || [ "${result_cnt_sqlite}" != "${cnt}" ]; then\
        echo -e "[ERROR] ${GPBACKMAN_TEST_COMMAND} test 5 failed.\n'${i}': get_yaml=${result_cnt_yaml}, get_sqlite=${result_cnt_sqlite}, want=${cnt}"
        exit 1
    fi
done
echo "[INFO] ${GPBACKMAN_TEST_COMMAND} test 5 passed."


echo "[INFO] ${GPBACKMAN_TEST_COMMAND} all tests passed"
exit 0
