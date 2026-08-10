#!/usr/bin/env bash
set -Eeuo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="backup-delete"

# Test 1: Delete local full backup
test_delete_local_full() {
    local timestamp=$(get_backup_info "get_local_full" --history-db "${HISTORY_DB}" --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep -v plugin | head -1 | awk '{print $1}')
    local output
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find full local backup timestamp"
        exit 1
    fi
    
    output=$(run_command_capture "delete_local_full" --timestamp "${timestamp}")
    assert_history_sync_success_output "${output}"
    
    assert_primary_backup_deleted "${timestamp}"
    assert_primary_standby_backup_row_match "${timestamp}"
    echo "[INFO] Backup ${timestamp} successfully marked as deleted and synced to standby"
}

# Test 2: Delete S3 incremental backup
test_delete_s3_incremental() {
    local timestamp=$(get_backup_info "get_s3_incremental" --history-db "${HISTORY_DB}" --type incremental | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep plugin | head -1 | awk '{print $1}')
    local output
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find S3 incremental backup"
        exit 1
    fi

    output=$(run_command_capture "delete_s3_incremental" --timestamp "${timestamp}" --plugin-config "${PLUGIN_CFG}")
    assert_history_sync_success_output "${output}"

    assert_primary_backup_deleted "${timestamp}"
    assert_primary_standby_backup_row_match "${timestamp}"
    echo "[INFO] S3 backup ${timestamp} successfully marked as deleted and synced to standby"
}

# Test 3: Delete S3 full backup with cascade
test_delete_s3_full_cascade() {
    local timestamp=$(get_backup_info "get_s3_full" --history-db "${HISTORY_DB}" --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep plugin | tail -1 | awk '{print $1}')
    local output
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find S3 full backup"
        exit 1
    fi
    # Expected: 1 backup from test 1 + 1 from test 2 + 2 backups (incr + full) from this test = 4 total
    local want=4
    output=$(run_command_capture "delete_s3_full_cascade" --timestamp "${timestamp}" --plugin-config "${PLUGIN_CFG}" --cascade)
    assert_history_sync_success_output "${output}"
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
    assert_primary_standby_deleted_backup_rows_match
}

# Test 4: Try to delete non-existent backup (should fail)
test_delete_nonexistent_backup() {
    local fake_timestamp="19990101000000"
    if ${BIN_DIR}/gpbackman backup-delete --history-db "${HISTORY_DB}" --timestamp "${fake_timestamp}" --force; then
        echo "[ERROR] Expected failure, but command succeeded"
        exit 1
    else
        echo "[INFO] Expected failure occurred"
    fi
}

# Test 5: Delete local data-only backup with standby history sync disabled
test_delete_local_data_only_no_history_sync_standby() {
    local timestamp
    timestamp="$(get_active_local_backup_timestamp "data-only")"
    local standby_row_before
    local standby_row_after

    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find data-only local backup timestamp"
        exit 1
    fi

    standby_row_before="$(standby_backup_row_for_timestamp "${timestamp}")"

    local output
    output=$(run_command_capture "delete_local_data_only_no_history_sync_standby" --timestamp "${timestamp}" --no-history-sync-standby)
    assert_history_sync_disabled_output "${output}"

    assert_primary_backup_deleted "${timestamp}"
    standby_row_after="$(standby_backup_row_for_timestamp "${timestamp}")"
    assert_string_equals "${standby_row_before}" "${standby_row_after}" "standby row should remain unchanged when history sync is disabled"

    echo "[INFO] Backup ${timestamp} successfully marked as deleted with standby history sync disabled"
}

# Test 6: Delete local full backup when standby rsync times out best-effort
test_delete_local_full_rsync_timeout_best_effort() {
    local timestamp
    timestamp="$(get_active_local_backup_timestamp "full")"
    local standby_row_before
    local standby_row_after
    local fake_bin_dir
    local output
    local elapsed

    standby_row_before="$(standby_backup_row_for_timestamp "${timestamp}")"
    fake_bin_dir="$(mktemp -d)"
    cat > "${fake_bin_dir}/rsync" <<'FAKE_RSYNC'
#!/usr/bin/env bash
exec sleep 30
FAKE_RSYNC
    chmod 755 "${fake_bin_dir}/rsync"

    SECONDS=0
    if ! output=$(
        export PATH="${fake_bin_dir}:${PATH}"
        run_command_capture "delete_local_full_rsync_timeout_best_effort" --timestamp "${timestamp}" --history-sync-standby-timeout 1
    ); then
        rm -rf "${fake_bin_dir}"
        exit 1
    fi
    elapsed="${SECONDS}"
    rm -rf "${fake_bin_dir}"

    assert_output_contains "${output}" "History db sync to standby coordinator failed; standby history may be stale:"
    assert_output_contains "${output}" "timed out after 1 seconds"
    assert_primary_backup_deleted "${timestamp}"
    standby_row_after="$(standby_backup_row_for_timestamp "${timestamp}")"
    assert_string_equals "${standby_row_before}" "${standby_row_after}" "standby row should remain unchanged when rsync times out"
    if [ "${elapsed}" -ge 15 ]; then
        echo "[ERROR] backup-delete timeout took ${elapsed} seconds; expected less than 15"
        exit 1
    fi
}

run_test "${COMMAND}" 1 test_delete_local_full
run_test "${COMMAND}" 2 test_delete_s3_incremental
run_test "${COMMAND}" 3 test_delete_s3_full_cascade
run_test "${COMMAND}" 4 test_delete_nonexistent_backup
run_test "${COMMAND}" 5 test_delete_local_data_only_no_history_sync_standby
run_test "${COMMAND}" 6 test_delete_local_full_rsync_timeout_best_effort

log_all_tests_passed "${COMMAND}"
