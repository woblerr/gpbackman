#!/usr/bin/env bash

readonly BIN_DIR="/home/gpadmin/gpbackman"
readonly DATA_DIR="/data/master/gpseg-1"

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
