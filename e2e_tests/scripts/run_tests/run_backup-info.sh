#!/usr/bin/env bash
set -Eeuo pipefail

COMMAND="backup-info"
BIN_DIR="/home/gpadmin/gpbackman"
DATA_DIR="/data/master/gpseg-1"

run_command(){
  local label="$1"; shift
  echo "[INFO] Running ${COMMAND}: $label"
  ${BIN_DIR}/gpbackman backup-info --history-db ${DATA_DIR}/gpbackup_history.db --deleted --failed  "$@" || { echo "[ERROR] ${COMMAND} $label failed"; exit 1; }
}

################################################################
# Count of all backups in the history database
test_id=1

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=12
got=$(run_command total_backups | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got" -eq "$want" ] || { echo "[ERROR] Expected $want , got $got"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Count of all full backups in the history database
# Compare the number of backups from the output of all backups and 
# from the output with the --type full flag
test_id=2

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=7
got1=$(run_command total_full_backups | grep -E '^[[:space:]][0-9]{14} ' | grep full | wc -l)
got2=$(run_command filter_full_backups --type full | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got1" -eq "$want" ] && [ "$got2" -eq "$want" ] || { echo "[ERROR] Expected $want , got1=$got1, got2=$got2"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Count of all incremental backups in the history database
# Compare the number of backups from the output of all backups and 
# from the output with the --type incremental flag
test_id=3 

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=3
got1=$(run_command total_incremental_backups | grep -E '^[[:space:]][0-9]{14} ' | grep incremental | wc -l)
got2=$(run_command filter_incremental_backups --type incremental | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got1" -eq "$want" ] && [ "$got2" -eq "$want" ] || { echo "[ERROR] Expected $want , got1=$got1, got2=$got2"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Count of backups which include table sch2.tbl_c
test_id=4 

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=2
got=$(run_command total_include_table_backups --table sch2.tbl_c | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got" -eq "$want" ] || { echo "[ERROR] Expected $want , got $got"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Count of backups which exclude table sch2.tbl_d

test_id=5 

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=2
got=$(run_command total_exclude_table_backups --table sch2.tbl_d --exclude | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got" -eq "$want" ] || { echo "[ERROR] Expected $want , got $got"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Count of full backups which include table sch2.tbl_c
# Use --type full to filter only full backups
test_id=6

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=1
got=$(run_command total_include_table_full_backups --table sch2.tbl_c --type full | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got" -eq "$want" ] || { echo "[ERROR] Expected $want , got $got"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"


################################################################
# Count of  incremental backups which exclude table sch2.tbl_d
# Use --type incremental to filter only incremental backups
test_id=7

echo "[INFO] ${COMMAND} TEST ${test_id}"

want=1
got=$(run_command total_exclude_table_incremental_backups --table sch2.tbl_d --exclude --type incremental | grep -E '^[[:space:]][0-9]{14} ' | wc -l)

[ "$got" -eq "$want" ] || { echo "[ERROR] Expected $want , got $got"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"
