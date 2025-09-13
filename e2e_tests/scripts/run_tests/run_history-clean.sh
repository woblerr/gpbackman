#!/usr/bin/env bash
set -Eeuo pipefail

# During the test, we consistently clean up the backups created within the script prepare/prepare_gpdb_backups.sh
# We clean up the history db from deleted backups and make sure that they are successfully deleted.
# It is checked that the number of deleted backups is 0.

# If the backup logic in the script changes, this test may fail, and corrections will also need to be made here.

# First, we delete all local backups older than the 9th timestamp using the backup-info command.

# Then we delete all S3 backups older than the 2th timestamp using the backup-info command.

# After each deletion we cleanup history db.

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="history-clean"

run_command(){
    local label="${1}"; shift
    run_gpbackman "${COMMAND}" "${label}" --history-db ${DATA_DIR}/gpbackup_history.db "$@"
}

run_backup_clean() {
    local label="${1}"; shift
    run_gpbackman "backup-clean" "${label}" --history-db ${DATA_DIR}/gpbackup_history.db "$@"
}

# Test 1: Clean from history db local backups older than timestamp (--before-timestamp)
test_history_clean_local_before_timestamp(){
    # Delete local backups
    local cutoff_timestamp=$(get_cutoff_timestamp 9)
    run_backup_clean "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}"
    run_command "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}"
    local want=0
    # Count deleted backups
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

# Test 2: Clean from history db S3 backups older than timestamp (--before-timestamp)
test_history_clean_s3_before_timestamp(){
    # Delete S3 backups
    local cutoff_timestamp=$(get_cutoff_timestamp 2)
    run_backup_clean "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}" --plugin-config "${PLUGIN_CFG}" --cascade
    run_command "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}"
    local want=0
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

run_test "${COMMAND}" 1 test_history_clean_local_before_timestamp
run_test "${COMMAND}" 2 test_history_clean_s3_before_timestamp

log_all_tests_passed "${COMMAND}"
