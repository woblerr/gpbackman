#!/usr/bin/env bash
set -Eeuo pipefail

# During the test, we consistently clean up the backups created within the script prepare/prepare_gpdb_backups.sh
# We clean up the history db from deleted backups and make sure that they are successfully deleted.
# It is checked that the number of deleted backups is 0.

# If the backup logic in the script changes, this test may fail, and corrections will also need to be made here.

# First, we delete all local backups older than the 9th timestamp using the backup-info command.

# Then we delete all S3 backups older than the 2th timestamp using the backup-info command.

# After each deletion we cleanup history db.

# Then we run history cleanup with standby history sync disabled for both mutation and no-op cases.

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="history-clean"

run_backup_clean() {
    local label="${1}"; shift
    run_gpbackman "backup-clean" "${label}" --history-db "${HISTORY_DB}" "$@"
}

# Test 1: Clean from history db local backups older than timestamp (--before-timestamp)
test_history_clean_local_before_timestamp(){
    # Delete local backups
    local cutoff_timestamp=$(get_cutoff_timestamp 9)
    local output
    run_backup_clean "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}"
    output=$(run_command_capture "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}")
    assert_history_sync_success_output "${output}"
    local want=0
    # Count deleted backups
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
    assert_primary_standby_deleted_backup_rows_match
}

# Test 2: Clean from history db S3 backups older than timestamp (--before-timestamp)
test_history_clean_s3_before_timestamp(){
    # Delete S3 backups
    local cutoff_timestamp=$(get_cutoff_timestamp 2)
    local output
    run_backup_clean "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}" --plugin-config "${PLUGIN_CFG}" --cascade
    output=$(run_command_capture "clean_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}")
    assert_history_sync_success_output "${output}"
    local want=0
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
    assert_primary_standby_deleted_backup_rows_match
}

# Test 3: Clean history records with standby history sync disabled
test_history_clean_no_history_sync_standby(){
    local timestamp
    timestamp="$(get_active_local_backup_timestamp "full")"
    local output
    local setup_output

    setup_output=$(run_gpbackman_capture "backup-delete" "delete_before_history_clean_no_history_sync_standby" --history-db "${HISTORY_DB}" --timestamp "${timestamp}")
    assert_history_sync_success_output "${setup_output}"
    if [ -z "$(standby_backup_row_for_timestamp "${timestamp}")" ]; then
        echo "[ERROR] standby should contain setup history record before disabled history clean"
        exit 1
    fi

    output=$(run_command_capture "clean_no_history_sync_standby" --before-timestamp 99999999999999 --no-history-sync-standby)
    assert_history_sync_disabled_output "${output}"
    assert_string_equals "" "$(primary_backup_row_for_timestamp "${timestamp}")" "primary history record should be cleaned"
    if [ -z "$(standby_backup_row_for_timestamp "${timestamp}")" ]; then
        echo "[ERROR] standby history record should remain when history sync is disabled"
        exit 1
    fi
}

# Test 4: Run a no-op history cleanup with standby history sync disabled
test_history_clean_no_history_sync_standby_noop(){
    local output
    output=$(run_command_capture "clean_no_history_sync_standby_noop" --before-timestamp 19000101000000 --no-history-sync-standby)
    assert_history_sync_disabled_output "${output}"
    local want=0
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

run_test "${COMMAND}" 1 test_history_clean_local_before_timestamp
run_test "${COMMAND}" 2 test_history_clean_s3_before_timestamp
run_test "${COMMAND}" 3 test_history_clean_no_history_sync_standby
run_test "${COMMAND}" 4 test_history_clean_no_history_sync_standby_noop

log_all_tests_passed "${COMMAND}"
