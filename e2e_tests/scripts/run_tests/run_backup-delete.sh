#!/usr/bin/env bash
set -Eeuo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="backup-delete"

run_command(){
    local label="${1}"; shift
    echo "[INFO] Running ${COMMAND}: ${label}"
    ${BIN_DIR}/gpbackman backup-delete --history-db ${DATA_DIR}/gpbackup_history.db "$@" || { echo "[ERROR] ${COMMAND} ${label} failed"; exit 1; }
}

get_backup_info_for_timestamp(){
    local timestamp="${1}"
    ${BIN_DIR}/gpbackman backup-info --history-db ${DATA_DIR}/gpbackup_history.db --deleted --failed | grep "${timestamp}" || echo "No info found for timestamp ${timestamp}"
}

# Test 1: Delete local full backup
test_delete_local_full() {
    local timestamp=$(get_backup_info "get_local_full" --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep -v plugin | head -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find full local backup timestamp"
        exit 1
    fi
    
    run_command "delete_local_full" --timestamp "${timestamp}"
    
    local deleted_backup=$(get_backup_info_for_timestamp "${timestamp}")
    local date_deleted=$(echo "${deleted_backup}" | grep "${timestamp}" | awk -F'|' '{print $NF}' | xargs)
    
    if [ -n "${date_deleted}" ]; then
        echo "[INFO] Backup ${timestamp} successfully marked as deleted"
    else
        echo "[ERROR] Backup should be marked as deleted"
        exit 1
    fi
}

# Test 2: Delete S3 incremental backup
test_delete_s3_incremental() {
    local timestamp=$(get_backup_info "get_s3_incremental" --type incremental | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep plugin | head -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find S3 incremental backup"
        exit 1
    fi
    
    run_command "delete_s3_incremental" --timestamp "${timestamp}" --plugin-config /home/gpadmin/gpbackup_s3_plugin.yaml
    
    local deleted_backup=$(get_backup_info_for_timestamp "${timestamp}")
    local date_deleted=$(echo "${deleted_backup}" | grep "${timestamp}" | awk -F'|' '{print $NF}' | xargs)
    
    if [ -n "${date_deleted}" ]; then
        echo "[INFO] S3 backup ${timestamp} successfully marked as deleted"
    else
        echo "[ERROR] S3 backup should be marked as deleted"
        exit 1
    fi
}

# Test 3: Delete S3 full backup with cascade
test_delete_s3_full_cascade() {
    local timestamp=$(get_backup_info "get_s3_full" --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep plugin | tail -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find S3 full backup"
        exit 1
    fi
    # Expected: 1 backup from test 1 + 1 from test 2 + 2 backups (incr + full) from this test = 4 total
    local want=4
    run_command "delete_s3_full_cascade" --timestamp "${timestamp}" --plugin-config /home/gpadmin/gpbackup_s3_plugin.yaml --cascade
    local got=$(count_deleted_backups)
    assert_equals "${want}" "${got}"
}

# Test 4: Try to delete non-existent backup (should fail)
test_delete_nonexistent_backup() {
    local fake_timestamp="19990101000000"
    
    echo "[INFO] Attempting to delete non-existent backup: ${fake_timestamp}"
    
    if ${BIN_DIR}/gpbackman backup-delete --history-db ${DATA_DIR}/gpbackup_history.db --timestamp "${fake_timestamp}" --force 2>/dev/null; then
        echo "[ERROR] Expected deletion of non-existent backup to fail, but it succeeded"
        exit 1
    else
        echo "[INFO] Deletion of non-existent backup correctly failed as expected"
    fi
}

run_test "${COMMAND}" 1 test_delete_local_full
run_test "${COMMAND}" 2 test_delete_s3_incremental  
run_test "${COMMAND}" 3 test_delete_s3_full_cascade
run_test "${COMMAND}" 4 test_delete_nonexistent_backup

log_all_tests_passed "${COMMAND}"
