#!/usr/bin/env bash

BIN_DIR="/home/gpadmin/gpbackman"
DATA_DIR="/data/master/gpseg-1"
PLUGIN_CFG="/home/gpadmin/gpbackup_s3_plugin.yaml"

TIMESTAMP_GREP_PATTERN='^[[:space:]][0-9]{14}'

log_test_start() {
    local command="${1}"
    local test_id="${2}"
    echo "[INFO] ${command} TEST ${test_id}"
}

log_test_success() {
    local command="${1}"
    local test_id="${2}"
    echo "[INFO] ${command} TEST ${test_id} is successful"
}

log_all_tests_passed() {
    local command="${1}"
    echo "[INFO] ${command} all tests passed"
}

get_backup_info() {
    local label="${1}"; shift
    ${BIN_DIR}/gpbackman backup-info --history-db ${DATA_DIR}/gpbackup_history.db --deleted --failed "$@" || { 
        echo "[ERROR] backup-info ${label} failed"; exit 1; 
    }
}

count_deleted_backups() {
    get_backup_info "count_deleted" | grep -E "${TIMESTAMP_GREP_PATTERN}" | awk -F'|' 'NF >= 9 && $NF !~ /^[[:space:]]*$/' | wc -l
}

get_cutoff_timestamp() {
    local line_no="$1"
    get_backup_info "get_line_${line_no}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | sed -n "${line_no}p" | awk '{print $1}'
}

assert_equals() {
    local expected="${1}"
    local actual="${2}"
    local message="${3:-}"
    
    [ "${actual}" -eq "${expected}" ] || { 
        echo "[ERROR] Expected ${expected}, got ${actual}${message:+ - ${message}}"; exit 1; 
    }
}

assert_equals_both() {
    local expected="${1}"
    local actual1="${2}"  
    local actual2="${3}"
    local message="${4:-}"
    
    [ "${actual1}" -eq "${expected}" ] && [ "${actual2}" -eq "${expected}" ] || { 
        echo "[ERROR] Expected ${expected}, got1=${actual1}, got2=${actual2}${message:+ - ${message}}"; exit 1; 
    }
}

run_test() {
    local command="${1}"
    local test_id="${2}"
    local test_function="${3}"
    
    log_test_start "${command}" "${test_id}"
    ${test_function}
    log_test_success "${command}" "${test_id}"
}
