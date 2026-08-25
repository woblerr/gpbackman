#!/usr/bin/env bash
set -Eeuo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="backup-info"

get_backup_info_timestamp() {
    local label="${1}"; shift
    run_gpbackman "backup-info" "${label}"  "$@"
}

BACKUP_INFO_FILTER_DATABASE=""
BACKUP_INFO_FILTER_ADDITIONAL_TIMESTAMP=""

assert_backup_info_database_rows() {
    local output="${1}"
    local expected_database="${2}"
    local expected_count="${3}"
    local rows
    local actual_count
    local matching_count

    rows="$(printf '%s\n' "${output}" | backup_info_rows | awk -F'|' -v database="${expected_database}" '
        function trim(value) {
            sub(/^[[:space:]]+/, "", value)
            sub(/[[:space:]]+$/, "", value)
            return value
        }
        { print trim($4) }
    ')"
    if [ -n "${rows}" ]; then
        actual_count="$(printf '%s\n' "${rows}" | wc -l)"
    else
        actual_count=0
    fi
    if [ "${actual_count}" -ne "${expected_count}" ]; then
        echo "[ERROR] Expected ${expected_count} backup-info rows for database ${expected_database}, got ${actual_count}"
        echo "${output}"
        exit 1
    fi

    matching_count="$(printf '%s\n' "${output}" | backup_info_rows | awk -F'|' -v database="${expected_database}" '
        function trim(value) {
            sub(/^[[:space:]]+/, "", value)
            sub(/[[:space:]]+$/, "", value)
            return value
        }
        trim($4) == database { count++ }
        END { print count + 0 }
    ')"
    if [ "${matching_count}" -ne "${actual_count}" ]; then
        echo "[ERROR] backup-info returned rows for a different database"
        echo "${output}"
        exit 1
    fi
}

# Test 1: Count all backups in history database
test_count_all_backups() {
    local want=12
    local got=$(get_backup_info total_backups  --history-db ${DATA_DIR}/gpbackup_history.db | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
}

# Test 2: Count all full backups
test_count_full_backups() {
    local want=7
    local got1=$(get_backup_info total_full_backups --history-db ${DATA_DIR}/gpbackup_history.db | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep full | wc -l)
    local got2=$(get_backup_info filter_full_backups --history-db ${DATA_DIR}/gpbackup_history.db --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals_both "${want}" "${got1}" "${got2}"
}

# Test 3: Count all incremental backups
# Compare the number of backups from the output of all backups and 
# from the output with the --type full flag
test_count_incremental_backups() {
    local want=3
    local got1=$(get_backup_info total_incremental_backups  --history-db ${DATA_DIR}/gpbackup_history.db | grep -E "${TIMESTAMP_GREP_PATTERN}" | grep incremental | wc -l)
    local got2=$(get_backup_info filter_incremental_backups --history-db ${DATA_DIR}/gpbackup_history.db --type incremental | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals_both "${want}" "${got1}" "${got2}"
}

# Test 4: Count backups that include table sch2.tbl_c
test_count_include_table_backups() {
    local want=2
    local got=$(get_backup_info total_include_table_backups --history-db ${DATA_DIR}/gpbackup_history.db --table sch2.tbl_c | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
}

# Test 5: Count backups that exclude schema sch1
test_count_exclude_schema_backups() {
    local want=2
    local got=$(get_backup_info total_exclude_schema_backups --history-db ${DATA_DIR}/gpbackup_history.db --schema sch1 --exclude | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
}

# Test 6: Count full backups that include table sch2.tbl_c
# Use --type full to filter only full backups
test_count_include_table_full_backups() {
    local want=1
    local got=$(get_backup_info total_include_table_full_backups --history-db ${DATA_DIR}/gpbackup_history.db --table sch2.tbl_c --type full | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
}

# Test 7: Count incremental backups that exclude schema sch1
test_count_exclude_schema_incremental_backups() {
    local want=1
    local got=$(get_backup_info total_exclude_schema_incremental_backups --history-db ${DATA_DIR}/gpbackup_history.db --schema sch1 --exclude --type incremental | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
}

# Test 8: Check backup chain and details for include tables sch2.tbl_c, sch2.tbl_d
test_backup_chain_include_tables() {
    local want=2
    local cutoff_timestamp=$(get_cutoff_timestamp 7)
    local got=$(get_backup_info_timestamp backup_chain_include_tables --history-db ${DATA_DIR}/gpbackup_history.db --timestamp "${cutoff_timestamp}" --detail| grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
    local got_details=$(get_backup_info_timestamp backup_chain_include_tables --history-db ${DATA_DIR}/gpbackup_history.db --timestamp "${cutoff_timestamp}" --detail| grep -E "${TIMESTAMP_GREP_PATTERN}" | awk -F'|' '{print $NF}')
    if [[ -z "${got_details//[[:space:]]/}" ]]; then
        echo "[ERROR] Expected details column to be non-empty"
        exit 1
    fi
}

# Test 9: Check backup chain for incremental backup that exclude schema sch1
# For incremental there is no backup chain, so only one backup should be returned
test_backup_chain_incremental_exclude() {
    local want=1
    local cutoff_timestamp=$(get_cutoff_timestamp 3)
    local got=$(get_backup_info_timestamp backup_chain_incremental_exclude --history-db ${DATA_DIR}/gpbackup_history.db --timestamp "${cutoff_timestamp}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
    local got_details=$(get_backup_info_timestamp backup_chain_incremental_exclude --history-db ${DATA_DIR}/gpbackup_history.db --timestamp "${cutoff_timestamp}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | awk -F'|' '{print $NF}')
    if [[ ! -z "${got_details//[[:space:]]/}" ]]; then
        echo "[ERROR] Expected details column to be empty"
        exit 1
    fi
}

# Test 10: Check full local backup with include table sch1.tbl_a and object filtering details
test_full_local_include_table_details() {
    local want=1
    local got=$(get_backup_info full_local_include_table_details --history-db ${DATA_DIR}/gpbackup_history.db --table sch1.tbl_a --type full --detail | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
    assert_equals "${want}" "${got}"
    local got_details=$(get_backup_info full_local_include_table_details --history-db ${DATA_DIR}/gpbackup_history.db --table sch1.tbl_a --type full --detail| grep -E "${TIMESTAMP_GREP_PATTERN}" | awk -F'|' '{print $NF}')
    if [[ -z "${got_details//[[:space:]]/}" ]]; then
        echo "[ERROR] Expected details column to be non-empty"
        exit 1
    fi
}

# Test 11: Count all backups using --auto-load-history-db and MASTER_DATA_DIRECTORY
test_count_all_backups_auto_load_master() {
    local want=12
    local got=$(
        (
            export MASTER_DATA_DIRECTORY="${DATA_DIR}"
            unset COORDINATOR_DATA_DIRECTORY
            get_backup_info total_backups_auto_load_master --auto-load-history-db
        ) | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l
    )
    assert_equals "${want}" "${got}"
}

# Test 12: Count all backups using --auto-load-history-db and COORDINATOR_DATA_DIRECTORY
test_count_all_backups_auto_load_coordinator() {
    local want=12
    local got=$(
        (
            unset MASTER_DATA_DIRECTORY
            export COORDINATOR_DATA_DIRECTORY="${DATA_DIR}"
            get_backup_info total_backups_auto_load_coordinator --auto-load-history-db
        ) | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l
    )
    assert_equals "${want}" "${got}"
}

# Test 13: Create distinguishable local backups for two databases after timestamp-index-sensitive cases
test_prepare_database_filter_backups() {
    BACKUP_INFO_FILTER_DATABASE="backup_info_filter"
    create_local_backup_for_database demo >/dev/null
    BACKUP_INFO_FILTER_ADDITIONAL_TIMESTAMP="$(create_additional_database_local_backup "${BACKUP_INFO_FILTER_DATABASE}")"
}

# Test 14: Filter backup-info rows by an exact database name
test_database_filter() {
    local output

    output="$(get_backup_info database_filter --history-db "${DATA_DIR}/gpbackup_history.db" --database "${BACKUP_INFO_FILTER_DATABASE}")"
    assert_backup_info_database_rows "${output}" "${BACKUP_INFO_FILTER_DATABASE}" 1
}

# Test 15: Combine the database filter with --type full
test_database_filter_with_type() {
    local output

    output="$(get_backup_info database_filter_with_type --history-db "${DATA_DIR}/gpbackup_history.db" --database "${BACKUP_INFO_FILTER_DATABASE}" --type full)"
    assert_backup_info_database_rows "${output}" "${BACKUP_INFO_FILTER_DATABASE}" 1
}

# Test 16: Filter timestamp output and details by database
test_database_filter_timestamp_detail() {
    local output

    output="$(get_backup_info_timestamp database_filter_timestamp_detail --history-db "${DATA_DIR}/gpbackup_history.db" --database "${BACKUP_INFO_FILTER_DATABASE}" --timestamp "${BACKUP_INFO_FILTER_ADDITIONAL_TIMESTAMP}" --detail)"
    assert_backup_info_database_rows "${output}" "${BACKUP_INFO_FILTER_DATABASE}" 1

    output="$(get_backup_info_timestamp database_filter_timestamp_mismatch --history-db "${DATA_DIR}/gpbackup_history.db" --database demo --timestamp "${BACKUP_INFO_FILTER_ADDITIONAL_TIMESTAMP}" --detail)"
    assert_backup_info_database_rows "${output}" demo 0
}

# Test 17: An unknown database produces a successful empty result
test_database_filter_unknown_database() {
    local output
    local unknown_database="backup_info_unknown"

    output="$(get_backup_info unknown_database_filter --history-db "${DATA_DIR}/gpbackup_history.db" --database "${unknown_database}")"
    assert_backup_info_database_rows "${output}" "${unknown_database}" 0
}

run_test "${COMMAND}" 1 test_count_all_backups
run_test "${COMMAND}" 2 test_count_full_backups
run_test "${COMMAND}" 3 test_count_incremental_backups
run_test "${COMMAND}" 4 test_count_include_table_backups
run_test "${COMMAND}" 5 test_count_exclude_schema_backups
run_test "${COMMAND}" 6 test_count_include_table_full_backups
run_test "${COMMAND}" 7 test_count_exclude_schema_incremental_backups
run_test "${COMMAND}" 8 test_backup_chain_include_tables
run_test "${COMMAND}" 9 test_backup_chain_incremental_exclude
run_test "${COMMAND}" 10 test_full_local_include_table_details
run_test "${COMMAND}" 11 test_count_all_backups_auto_load_master
run_test "${COMMAND}" 12 test_count_all_backups_auto_load_coordinator
run_test "${COMMAND}" 13 test_prepare_database_filter_backups
run_test "${COMMAND}" 14 test_database_filter
run_test "${COMMAND}" 15 test_database_filter_with_type
run_test "${COMMAND}" 16 test_database_filter_timestamp_detail
run_test "${COMMAND}" 17 test_database_filter_unknown_database

log_all_tests_passed "${COMMAND}"
