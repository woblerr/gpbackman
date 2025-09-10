#!/usr/bin/env bash
set -Eeuo pipefail

# In the test, we consistently perform cleanup for backups created within the script prepare/prepare_gpdb_backups.sh
# If the backup creation logic changes in the script, this test may start to fail and corrections also need to be made here.
# 
# First, we delete all local backups older than the 9th timestamp from backup-info command,
# there should be 3 deleted backups. 
# 
# Then we delete all local backups younger than the 3th timestamp,
# there should be a total of 5 deleted backups.
# 
# Then we delete all S3 backups younger than the 5th timestamp,
# there should be a total of 7 deleted backups.
# 
# Then we delete all S3 backups older than the 5th timestamp, 
# there should be a total of 12 deleted backups.

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="backup-clean"

run_command() {
    local label="${1}"; shift
    echo "[INFO] Running ${COMMAND}: ${label}"
    ${BIN_DIR}/gpbackman backup-clean --history-db ${DATA_DIR}/gpbackup_history.db "$@" || { 
        echo "[ERROR] ${COMMAND} ${label} failed"; exit 1; 
    }
}

# Test 1: Clean local backups older than timestamp (--before-timestamp)
#  Without --cascade, no dependent backups
test_clean_local_backups_before_timestamp() {
    local want=3
    local cutoff_timestamp=$(get_cutoff_timestamp 9)
    run_command "clean_local_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

# Test 2: Clean local backups newer than timestamp (--after-timestamp)
# Without --cascade, no dependent backups
test_clean_local_backups_after_timestamp() {
    local want=5
    local cutoff_timestamp=$(get_cutoff_timestamp 3)
    run_command "clean_local_after_${cutoff_timestamp}" --after-timestamp "${cutoff_timestamp}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

# Test 3: Clean S3 backups newer than timestamp (--after-timestamp)
# Without --cascade, no dependent backups
test_clean_s3_backups_after_timestamp() {
    local want=7
    local cutoff_timestamp=$(get_cutoff_timestamp 5)
    run_command "clean_s3_after_${cutoff_timestamp}" --after-timestamp "${cutoff_timestamp}" --plugin-config "${PLUGIN_CFG}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

# Test 4: Clean S3 backups older than timestamp (--before-timestamp)
# With --cascade
test_clean_s3_backups_before_timestamp() {
    local want=12
    local cutoff_timestamp=$(get_cutoff_timestamp 5)
    run_command "clean_s3_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}" --plugin-config "${PLUGIN_CFG}" --cascade
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

run_test "${COMMAND}" 1 test_clean_local_backups_before_timestamp
run_test "${COMMAND}" 2 test_clean_local_backups_after_timestamp  
run_test "${COMMAND}" 3 test_clean_s3_backups_after_timestamp
run_test "${COMMAND}" 4 test_clean_s3_backups_before_timestamp

log_all_tests_passed "${COMMAND}"
