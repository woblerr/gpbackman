#!/usr/bin/env bash
set -Eeuo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="backup-info"

get_backup_info_timestamp() {
    local label="${1}"; shift
    run_gpbackman "backup-info" "${label}"  "$@"
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

log_all_tests_passed "${COMMAND}"
