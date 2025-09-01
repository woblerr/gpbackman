#!/usr/bin/env bash
set -Eeuo pipefail

COMMAND="backup-delete"
BIN_DIR="/home/gpadmin/gpbackman"
DATA_DIR="/data/master/gpseg-1"

run_command(){
  local label="$1"; shift
  echo "[INFO] Running ${COMMAND}: $label"
  ${BIN_DIR}/gpbackman backup-delete --history-db ${DATA_DIR}/gpbackup_history.db "$@" || { echo "[ERROR] ${COMMAND} $label failed"; exit 1; }
}

get_backup_info(){
  local label="$1"; shift
  ${BIN_DIR}/gpbackman backup-info --history-db ${DATA_DIR}/gpbackup_history.db --deleted --failed "$@" || { echo "[ERROR] backup-info $label failed"; exit 1; }
}

get_backup_info_for_timestamp(){
  local timestamp="$1"
  ${BIN_DIR}/gpbackman backup-info --history-db ${DATA_DIR}/gpbackup_history.db --deleted --failed | grep "$timestamp" || echo "No info found for timestamp $timestamp"
}

################################################################
# Test 1: Delete local full backup

test_id=1

echo "[INFO] ${COMMAND} TEST ${test_id}"

timestamp=$(get_backup_info "get_local_full" --type full | grep -E '^[[:space:]][0-9]{14} ' | grep -v plugin | head -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find full local backup timestamp"
    exit 1
fi

run_command "delete_local_full" --timestamp "$timestamp"

deleted_backup=$(get_backup_info_for_timestamp "$timestamp")

date_deleted=$(echo "$deleted_backup" | grep "$timestamp" | awk -F'|' '{print $NF}' | xargs)

if [ -n "$date_deleted" ]; then
    echo "[INFO] Backup $timestamp successfully marked as deleted"
else
    echo "[ERROR] Backup should be marked as deleted"
    exit 1
fi

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"


################################################################
# Test 2: Delete S3 incremental backup

test_id=2

echo "[INFO] ${COMMAND} TEST ${test_id}"

timestamp=$(get_backup_info "get_s3_incremental" --type incremental | grep -E '^[[:space:]][0-9]{14} ' | grep plugin | head -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find S3 incremental backup"
    exit 1
fi

run_command "delete_s3_incremental" --timestamp "$timestamp" --plugin-config /home/gpadmin/gpbackup_s3_plugin.yaml

deleted_backup=$(get_backup_info_for_timestamp "$timestamp")

date_deleted=$(echo "$deleted_backup" | grep "$timestamp" | awk -F'|' '{print $NF}' | xargs)
if [ -n "$date_deleted" ]; then
    echo "[INFO] S3 backup $timestamp successfully marked as deleted"
else
    echo "[ERROR] S3 backup should be marked as deleted"
    exit 1
fi

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Test 3: Delete S3 full backup with cascade

test_id=3

echo "[INFO] ${COMMAND} TEST ${test_id}"

timestamp=$(get_backup_info "get_s3_full" --type full | grep -E '^[[:space:]][0-9]{14} ' | grep plugin | tail -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find S3 full backup"
    exit 1
fi

run_command "delete_s3_full_cascade" --timestamp "$timestamp" --plugin-config /home/gpadmin/gpbackup_s3_plugin.yaml --cascade

deleted_count=$(get_backup_info "count_deleted" --deleted | grep -E '^[[:space:]][0-9]{14} ' | awk -F'|' 'NF >= 9 && $NF !~ /^[[:space:]]*$/' | wc -l)

# Delete one backup from test 1 and one from test 2
# Plus 2 backups (incr + full) from this test
[ "$deleted_count" -eq 4 ] || { echo "[ERROR] Expected 4 backups to be deleted, but found $deleted_count"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Test 4: Try to delete non-existent backup (should fail)

test_id=4

echo "[INFO] ${COMMAND} TEST ${test_id}"

fake_timestamp="19990101000000"

echo "[INFO] Attempting to delete non-existent backup: $fake_timestamp"

if ${BIN_DIR}/gpbackman backup-delete --history-db ${DATA_DIR}/gpbackup_history.db --timestamp "$fake_timestamp" --force 2>/dev/null; then
    echo "[ERROR] Expected deletion of non-existent backup to fail, but it succeeded"
    exit 1
else
    echo "[INFO] Deletion of non-existent backup correctly failed as expected"
fi

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
echo "[INFO] ${COMMAND} all tests passed"
