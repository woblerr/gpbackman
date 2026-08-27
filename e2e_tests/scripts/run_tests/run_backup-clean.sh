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
#
# Then we run a no-op cleanup with standby history sync disabled,
# there should still be a total of 12 deleted backups.
#
# Then we create local backups for both standard databases, clean only the
# filter database, and verify the selective result on primary and standby.

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="backup-clean"

# Test 1: Clean local backups older than timestamp (--before-timestamp)
#  Without --cascade, no dependent backups
test_clean_local_backups_before_timestamp() {
    local want=3
    local cutoff_timestamp=$(get_cutoff_timestamp 9)
    local output
    output=$(run_command_capture "clean_local_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}")
    assert_history_sync_success_output "${output}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
    assert_primary_standby_deleted_backup_rows_match
}

# Test 2: Clean local backups newer than timestamp with standby history sync disabled
test_clean_local_backups_after_timestamp_no_history_sync_standby() {
    local want=5
    local cutoff_timestamp=$(get_cutoff_timestamp 3)
    local standby_rows_before
    local standby_rows_after
    local output

    standby_rows_before="$(standby_deleted_backup_rows)"
    output=$(run_command_capture "clean_local_after_${cutoff_timestamp}_no_history_sync_standby" --after-timestamp "${cutoff_timestamp}" --no-history-sync-standby)
    assert_history_sync_disabled_output "${output}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"

    standby_rows_after="$(standby_deleted_backup_rows)"
    assert_string_equals "${standby_rows_before}" "${standby_rows_after}" "standby rows should remain unchanged when history sync is disabled"
}

# Test 3: Clean S3 backups newer than timestamp (--after-timestamp)
# Without --cascade, no dependent backups
test_clean_s3_backups_after_timestamp() {
    local want=7
    local cutoff_timestamp=$(get_cutoff_timestamp 5)
    local output
    output=$(run_command_capture "clean_s3_after_${cutoff_timestamp}" --after-timestamp "${cutoff_timestamp}" --plugin-config "${PLUGIN_CFG}")
    assert_history_sync_success_output "${output}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
    assert_primary_standby_deleted_backup_rows_match
}

# Test 4: Clean S3 backups older than timestamp (--before-timestamp)
# With --cascade
test_clean_s3_backups_before_timestamp() {
    local want=12
    local cutoff_timestamp=$(get_cutoff_timestamp 5)
    local output
    output=$(run_command_capture "clean_s3_before_${cutoff_timestamp}" --before-timestamp "${cutoff_timestamp}" --plugin-config "${PLUGIN_CFG}" --cascade)
    assert_history_sync_success_output "${output}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
    assert_primary_standby_deleted_backup_rows_match
}

# Test 5: Run a no-op cleanup with standby history sync disabled
test_clean_no_history_sync_standby_noop() {
    local want=12
    local output
    output=$(run_command_capture "clean_no_history_sync_standby_noop" --before-timestamp 19000101000000 --no-history-sync-standby)
    assert_history_sync_disabled_output "${output}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

# Test 6: Clean only local backups for the filter database
test_clean_filter_database_backups() {
    local primary_timestamp
    local filter_timestamp
    local output

    primary_timestamp="$(create_local_backup_for_database "${E2E_PRIMARY_DB}")"
    filter_timestamp="$(create_local_backup_for_database "${E2E_FILTER_DB}")"
    output=$(run_command_capture "clean_${E2E_FILTER_DB}" --before-timestamp 99999999999999 --database "${E2E_FILTER_DB}")
    assert_history_sync_success_output "${output}"

    assert_primary_backup_active "${primary_timestamp}"
    assert_primary_backup_deleted "${filter_timestamp}"
    assert_standby_backup_active "${primary_timestamp}"
    assert_standby_backup_deleted "${filter_timestamp}"
    assert_primary_standby_backup_row_match "${primary_timestamp}"
    assert_primary_standby_backup_row_match "${filter_timestamp}"
}

run_test "${COMMAND}" 1 test_clean_local_backups_before_timestamp
run_test "${COMMAND}" 2 test_clean_local_backups_after_timestamp_no_history_sync_standby
run_test "${COMMAND}" 3 test_clean_s3_backups_after_timestamp
run_test "${COMMAND}" 4 test_clean_s3_backups_before_timestamp
run_test "${COMMAND}" 5 test_clean_no_history_sync_standby_noop
run_test "${COMMAND}" 6 test_clean_filter_database_backups

log_all_tests_passed "${COMMAND}"
