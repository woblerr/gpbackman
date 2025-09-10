#!/usr/bin/env bash
set -Eeuo pipefail

# Tests for history-migrate (current):
# 1) Migrate gpbackup_history_full_local.yaml into an empty DB in /tmp -> expect 2 backups.
# 2) Migrate the same file into an existing DB prepared by setup -> expect 12 base + 2 = 14 total.
# 3) Duplicate migration into the same DB -> must fail with UNIQUE constraint.
# 4) Migrate all YAML files into a fresh empty DB in /tmp -> expect 14 backups total.
# 5) Migrate all YAML files into an existing DB (excluding already migrated full_local) -> expect 12 base + 14 = 26 total; duplicates skipped.

source "$(dirname "${BASH_SOURCE[0]}")/common_functions.sh"

COMMAND="history-migrate"
SRC_DIR="/home/gpadmin/src_data"
WORK_BASE="/tmp/history_migrate_tests"

TEST_FILE_FULL_LOCAL="gpbackup_history_full_local.yaml"

run_command(){
  local label="${1}"; shift
  echo "[INFO] Running ${COMMAND}: ${label}"
  ${BIN_DIR}/gpbackman history-migrate "$@" || { echo "[ERROR] ${COMMAND} ${label} failed"; exit 1; }
}

prepare_workdir(){
  local name="${1}"
  local dir="${WORK_BASE}/${name}"
  rm -rf "${dir}" && mkdir -p "${dir}"
  echo "${dir}"
}

# Test 1: Single file into empty DB in /tmp
# Expect 2 backups from file
test_migrate_single_into_empty_db(){
  local workdir=$(prepare_workdir test1)
  cp "${SRC_DIR}/${TEST_FILE_FULL_LOCAL}" "${workdir}/"
  local db="${workdir}/gpbackup_history.db"
  run_command "single_into_empty_db" --history-file "${workdir}/${TEST_FILE_FULL_LOCAL}" --history-db "${db}"
  local want=2
  local got=$(get_backup_info total_full_backups --history-db ${db} | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
  assert_equals "${want}" "${got}"
}

# Test 2: Single file into existing DB (prepared by setup)
# 12 backups from initial setup + 2 from file
test_migrate_single_into_existing_db(){
  local workdir=$(prepare_workdir test2)
  cp "${SRC_DIR}/${TEST_FILE_FULL_LOCAL}" "${workdir}/"
  local db="${DATA_DIR}/gpbackup_history.db"
  run_command "single_into_existing_db" --history-file "${workdir}/${TEST_FILE_FULL_LOCAL}" --history-db "${db}"
  local want=14
  local got=$(get_backup_info total_full_backups --history-db ${db} | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
  assert_equals "${want}" "${got}"
}

# Test 3: Duplicate migration into the same DB must fail with UNIQUE constraint
test_migrate_duplicate_into_existing_db_fail(){
  local workdir=$(prepare_workdir test3)
  cp "${SRC_DIR}/${TEST_FILE_FULL_LOCAL}" "${workdir}/"
  local db="${DATA_DIR}/gpbackup_history.db"
  set +e
  set -x
  ${BIN_DIR}/gpbackman history-migrate --history-file "${workdir}/${TEST_FILE_FULL_LOCAL}" --history-db "${db}"
  if [ $? -eq 0 ]; then
    echo "[ERROR] Expected failure, but command succeeded"
    exit 1
  fi
  set +x
  set -e
}

# Test 4: All files into fresh empty DB in /tmp
# 14 backups from all files
test_migrate_all_into_empty_db(){
  local workdir=$(prepare_workdir test3)
  cp "${SRC_DIR}"/*.yaml "${workdir}/"
  local db="${workdir}/gpbackup_history.db"
  local args=()
  for f in "${workdir}"/*.yaml; do
    args+=(--history-file "${f}")
  done
  run_command "all_into_empty_db" "${args[@]}" --history-db "${db}"
  local want=14
  local got=$(get_backup_info total_full_backups --history-db "${db}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
  assert_equals "${want}" "${got}"
}

# Test 5: All files into existing DB
# 12 backups from initial setup + 12 from files
# The duplicates, already loaded in test2, should be skipped
test_migrate_all_into_existing_db(){
local workdir=$(prepare_workdir test4)
cp "${SRC_DIR}"/*.yaml "${workdir}/"
rm -f "${workdir}/${TEST_FILE_FULL_LOCAL}"
local db="${DATA_DIR}/gpbackup_history.db"
local args=()
for f in "${workdir}"/*.yaml; do
    args+=(--history-file "${f}")
done
run_command "all_into_existing_db" "${args[@]}" --history-db "${db}"
local want=26
local got=$(get_backup_info total_full_backups --history-db "${db}" | grep -E "${TIMESTAMP_GREP_PATTERN}" | wc -l)
assert_equals "${want}" "${got}"
}

run_test "${COMMAND}" 1 test_migrate_single_into_empty_db
run_test "${COMMAND}" 2 test_migrate_single_into_existing_db
run_test "${COMMAND}" 3 test_migrate_duplicate_into_existing_db_fail
run_test "${COMMAND}" 4 test_migrate_all_into_empty_db
run_test "${COMMAND}" 5 test_migrate_all_into_existing_db

log_all_tests_passed "${COMMAND}"
