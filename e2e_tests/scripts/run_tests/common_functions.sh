#!/usr/bin/env bash

BIN_DIR="/home/gpadmin/gpbackman"
DATA_DIR="/data/master/gpseg-1"
HISTORY_DB="${DATA_DIR}/gpbackup_history.db"
PLUGIN_CFG="/home/gpadmin/gpbackup_s3_plugin.yaml"

TIMESTAMP_GREP_PATTERN='^[[:space:]][0-9]{14}'

log_all_tests_passed() {
    local command="${1}"
    echo "[INFO] ${command} all tests passed"
}
run_gpbackman() {
    local subcmd="${1}"; shift
    local label="${1}"; shift
    echo "[INFO] Running ${subcmd}: ${label}"
    ${BIN_DIR}/gpbackman "${subcmd}" "$@" || { 
        echo "[ERROR] ${subcmd} ${label} failed"; exit 1; 
    }
}

run_gpbackman_capture() {
    local subcmd="${1}"; shift
    local label="${1}"; shift
    echo "[INFO] Running ${subcmd}: ${label}" >&2
    local output
    if ! output=$(${BIN_DIR}/gpbackman "${subcmd}" "$@" 2>&1); then
        echo "${output}" >&2
        echo "[ERROR] ${subcmd} ${label} failed" >&2
        exit 1
    fi
    echo "${output}" >&2
    echo "${output}"
}

run_command_capture() {
    local label="${1}"; shift
    run_gpbackman_capture "${COMMAND}" "${label}" --history-db "${HISTORY_DB}" "$@"
}

assert_output_contains() {
    local output="${1}"
    local expected="${2}"
    case "${output}" in
        *"${expected}"*) ;;
        *)
            echo "[ERROR] Expected output to contain: ${expected}"
            echo "${output}"
            exit 1
            ;;
    esac
}

assert_string_equals() {
    local expected="${1}"
    local actual="${2}"
    local message="${3:-values differ}"

    if [ "${actual}" != "${expected}" ]; then
        echo "[ERROR] ${message}"
        echo "[ERROR] Expected: ${expected}"
        echo "[ERROR] Actual: ${actual}"
        exit 1
    fi
}

assert_history_sync_success_output() {
    local output="${1}"
    assert_output_contains "${output}" "History db sync to standby coordinator completed:"
}

assert_history_sync_disabled_output() {
    local output="${1}"
    assert_output_contains "${output}" "Skipping history db sync to standby coordinator: disabled by --no-history-sync-standby"
}

get_backup_info() {
    local label="${1}"; shift
    run_gpbackman "backup-info" "${label}" --deleted --failed "$@"
}

count_deleted_backups() {
    get_backup_info "count_deleted" --history-db "${HISTORY_DB}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | awk -F'|' 'NF >= 9 && $NF !~ /^[[:space:]]*$/' | wc -l
}

get_cutoff_timestamp() {
    local line_no="$1"
    get_backup_info "get_line_${line_no}" --history-db "${HISTORY_DB}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | sed -n "${line_no}p" | awk '{print $1}'
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

get_primary_backup_info() {
    local label="${1}"; shift
    run_gpbackman "backup-info" "${label}" --history-db "${HISTORY_DB}" "$@"
}

get_standby_backup_info() {
    local label="${1}"; shift
    local output
    local arg

    for arg in "$@"; do
        case "${arg}" in
            --deleted|--failed) ;;
            *)
                echo "[ERROR] Unsupported standby backup-info argument: ${arg}"
                exit 1
                ;;
        esac
    done

    echo "[INFO] Running standby backup-info: ${label}"
    if ! output=$(ssh -o BatchMode=yes -o StrictHostKeyChecking=yes -o ConnectTimeout=10 standby "${BIN_DIR}/gpbackman" backup-info --history-db "${HISTORY_DB}" "$@" 2>&1); then
        echo "${output}"
        echo "[ERROR] standby backup-info ${label} failed"
        exit 1
    fi

    echo "${output}"
}

backup_info_rows() {
    grep -E "${TIMESTAMP_GREP_PATTERN}" || true
}

backup_info_row_for_timestamp() {
    local timestamp="${1}"
    grep -E "^[[:space:]]*${timestamp}[[:space:]]*\\|" | head -1 || true
}

primary_backup_row_for_timestamp() {
    local timestamp="${1}"
    get_primary_backup_info "primary_${timestamp}" --deleted --failed | backup_info_row_for_timestamp "${timestamp}"
}

standby_backup_row_for_timestamp() {
    local timestamp="${1}"
    get_standby_backup_info "standby_${timestamp}" --deleted --failed | backup_info_row_for_timestamp "${timestamp}"
}

assert_primary_backup_deleted() {
    local timestamp="${1}"
    local date_deleted

    date_deleted="$(primary_backup_row_for_timestamp "${timestamp}" | awk -F'|' '{print $NF}' | xargs)"
    if [ -z "${date_deleted}" ]; then
        echo "[ERROR] Backup should be marked as deleted: ${timestamp}"
        exit 1
    fi
}

assert_primary_standby_backup_row_match() {
    local timestamp="${1}"
    local primary_row
    local standby_row

    primary_row="$(primary_backup_row_for_timestamp "${timestamp}")"
    standby_row="$(standby_backup_row_for_timestamp "${timestamp}")"
    assert_string_equals "${primary_row}" "${standby_row}" "primary/standby backup row mismatch for ${timestamp}"
}

deleted_backup_rows() {
    backup_info_rows | awk -F'|' 'NF >= 9 && $NF !~ /^[[:space:]]*$/'
}

standby_deleted_backup_rows() {
    get_standby_backup_info "standby_deleted_rows" --deleted --failed | deleted_backup_rows
}

assert_primary_standby_deleted_backup_rows_match() {
    local primary_rows
    local standby_rows

    primary_rows="$(get_primary_backup_info "primary_deleted_rows" --deleted --failed | deleted_backup_rows)"
    standby_rows="$(standby_deleted_backup_rows)"
    assert_string_equals "${primary_rows}" "${standby_rows}" "primary/standby deleted backup rows mismatch"
}

get_active_local_backup_timestamp() {
    local backup_type="${1}"
    local timestamp

    timestamp="$(get_primary_backup_info "active_local_${backup_type}" --type "${backup_type}" | backup_info_rows | grep -v plugin | head -1 | awk '{print $1}')"
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find active local ${backup_type} backup timestamp"
        exit 1
    fi
    echo "${timestamp}"
}

get_backup_timestamp_for_database() {
    local database="${1}"
    local timestamp

    timestamp="$(get_primary_backup_info "latest_${database}_backup" --deleted --failed | backup_info_rows | awk -F'|' -v database="${database}" '
        function trim(value) {
            sub(/^[[:space:]]+/, "", value)
            sub(/[[:space:]]+$/, "", value)
            return value
        }
        trim($4) == database {
            print trim($2)
            exit
        }
    ')"
    if [ -z "${timestamp}" ]; then
        echo "[ERROR] Could not find backup timestamp for database: ${database}" >&2
        exit 1
    fi
    echo "${timestamp}"
}

create_local_backup_for_database() {
    local database="${1}"

    echo "[INFO] Creating local backup for database: ${database}" >&2
    gpbackup --dbname "${database}" >&2 || {
        echo "[ERROR] Could not create local backup for database: ${database}" >&2
        exit 1
    }
    sleep 1
    get_backup_timestamp_for_database "${database}"
}

create_additional_database_local_backup() {
    local database="${1}"

    createdb --maintenance-db demo "${database}" || {
        echo "[ERROR] Could not create additional database: ${database}" >&2
        exit 1
    }
    create_local_backup_for_database "${database}"
}

assert_primary_backup_active() {
    local timestamp="${1}"
    local row
    local date_deleted

    row="$(primary_backup_row_for_timestamp "${timestamp}")"
    if [ -z "${row}" ]; then
        echo "[ERROR] Active backup is missing from history: ${timestamp}"
        exit 1
    fi
    date_deleted="$(echo "${row}" | awk -F'|' '{print $NF}' | xargs)"
    if [ -n "${date_deleted}" ]; then
        echo "[ERROR] Backup should remain active: ${timestamp}"
        exit 1
    fi
}

assert_primary_backup_row_absent() {
    local timestamp="${1}"

    if [ -n "$(primary_backup_row_for_timestamp "${timestamp}")" ]; then
        echo "[ERROR] Backup should be absent from history: ${timestamp}"
        exit 1
    fi
}

run_test() {
    local command="${1}"
    local test_id="${2}"
    local test_function="${3}"
    
    echo "[INFO] ${command} TEST ${test_id}"
    ${test_function}
    echo "[INFO] ${command} TEST ${test_id} is successful"
}
