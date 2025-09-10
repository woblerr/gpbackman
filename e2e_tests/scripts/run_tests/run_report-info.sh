#!/usr/bin/env bash
set -Eeuo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="report-info"
BACKUP_DIR_PREFIX="/tmp/testWithPrefix"
BACKUP_DIR_SINGLE="/tmp/testNoPrefix"

run_command(){
  local label="${1}"; shift
  echo "[INFO] Running ${COMMAND}: ${label}"
  ${BIN_DIR}/gpbackman report-info --history-db ${DATA_DIR}/gpbackup_history.db "$@" || { echo "[ERROR] ${COMMAND} ${label} failed"; exit 1; }
}

# Test 1: Get report info for full local backup (without backup-dir)
test_report_full_local_no_dir() {
    local timestamp=$(get_backup_info "get_full_local" --history-db ${DATA_DIR}/gpbackup_history.db --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep -v plugin | head -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find full local backup timestamp"
        exit 1
    fi
    
    local report_output=$(run_command "full_local_no_dir" --timestamp "${timestamp}")
    
    echo "${report_output}" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
    echo "${report_output}" | grep -q "timestamp key:.*${timestamp}" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
    echo "${report_output}" | grep -q "plugin executable:.*None" || { echo "[ERROR] Expected 'plugin executable: None' for local backup"; exit 1; }
}

# Test 2: Get report info for full local backup (with backup-dir)
test_report_full_local_with_dir() {
    local timestamp=$(get_backup_info "get_full_local_with_dir" --history-db ${DATA_DIR}/gpbackup_history.db --history-db ${DATA_DIR}/gpbackup_history.db --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep -v plugin | head -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find full local backup timestamp for backup-dir test"
        exit 1
    fi
    
    local report_dir="/data/master/gpseg-1"
    local report_output=$(run_command "local_with_backup_dir_console" --timestamp "${timestamp}" --backup-dir "${report_dir}")
    
    echo "${report_output}" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
    echo "${report_output}" | grep -q "timestamp key:.*${timestamp}" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
    echo "${report_output}" | grep -q "plugin executable:.*None" || { echo "[ERROR] Expected 'plugin executable: None' for local backup"; exit 1; }
}

# Test 3: Get report info for full S3 backup (without plugin-report-file-path)
test_report_s3_no_plugin_path() {
    local timestamp=$(get_backup_info "get_full_s3" --history-db ${DATA_DIR}/gpbackup_history.db --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep plugin | head -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find full s3 backup timestamp"
        exit 1
    fi
    
    local report_output=$(run_command "s3_without_plugin_report_file_path" --timestamp "${timestamp}" --plugin-config "${PLUGIN_CFG}")
    
    echo "${report_output}" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
    echo "${report_output}" | grep -q "timestamp key:.*${timestamp}" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
    echo "${report_output}" | grep -q "plugin executable:.*gpbackup_s3_plugin" || { echo "[ERROR] Expected 'plugin executable: gpbackup_s3_plugin' for s3 backup"; exit 1; }
}

# Test 4: Get report info for full S3 backup (with plugin-report-file-path)
test_report_s3_with_plugin_path() {
    local timestamp=$(get_backup_info "get_full_s3" --history-db ${DATA_DIR}/gpbackup_history.db --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep plugin | head -1 | awk '{print $1}')
    
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find full s3 backup timestamp for plugin-report-file-path test"
        exit 1
    fi
    
    local report_dir="/backup/test/backups/${timestamp:0:8}/${timestamp}"
    local report_output=$(run_command "s3_with_plugin_report_file_path" --timestamp "${timestamp}" --plugin-config "${PLUGIN_CFG}" --plugin-report-file-path "${report_dir}")

    echo "${report_output}" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
    echo "${report_output}" | grep -q "timestamp key:.*${timestamp}" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
    echo "${report_output}" | grep -q "plugin executable:.*gpbackup_s3_plugin" || { echo "[ERROR] Expected 'plugin executable: gpbackup_s3_plugin' for s3 backup"; exit 1; }
}

run_test "${COMMAND}" 1 test_report_full_local_no_dir
run_test "${COMMAND}" 2 test_report_full_local_with_dir  
run_test "${COMMAND}" 3 test_report_s3_no_plugin_path
run_test "${COMMAND}" 4 test_report_s3_with_plugin_path

log_all_tests_passed "${COMMAND}"
