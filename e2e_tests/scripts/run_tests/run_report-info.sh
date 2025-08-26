#!/usr/bin/env bash
set -Eeuo pipefail

COMMAND="report-info"
BIN_DIR="/home/gpadmin/gpbackman"
DATA_DIR="/data/master/gpseg-1"
BACKUP_DIR_PREFIX="/tmp/testWithPrefix"
BACKUP_DIR_SINGLE="/tmp/testNoPrefix"

run_command(){
  local label="$1"; shift
  echo "[INFO] Running ${COMMAND}: $label"
  ${BIN_DIR}/gpbackman report-info --history-db ${DATA_DIR}/gpbackup_history.db "$@" || { echo "[ERROR] ${COMMAND} $label failed"; exit 1; }
}

get_backup_info(){
  local label="$1"; shift
  ${BIN_DIR}/gpbackman backup-info --history-db ${DATA_DIR}/gpbackup_history.db --deleted --failed "$@" || { echo "[ERROR] backup-info $label failed"; exit 1; }
}

################################################################
# Test 1: Get report info for full local backup (without using backup-dir)

test_id=1

echo "[INFO] ${COMMAND} TEST ${test_id}"

# Get timestamp for first full local backup
timestamp=$(get_backup_info "get_full_local" --type full | grep -E '^[[:space:]][0-9]{14} ' | grep -v plugin | head -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find full local backup timestamp"
    exit 1
fi

report_output=$(run_command "full_local_no_dir" --timestamp "$timestamp")

echo "$report_output" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
echo "$report_output" | grep -q "timestamp key:.*$timestamp" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
echo "$report_output" | grep -q "plugin executable:.*None" || { echo "[ERROR] Expected 'plugin executable: None' for local backup"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Test 2: Get report info for full local backup (with using backup-dir)

test_id=2

echo "[INFO] ${COMMAND} TEST ${test_id}"

timestamp=$(get_backup_info "get_full_local_with_dir" --type full | grep -E '^[[:space:]][0-9]{14} ' | grep -v plugin | head -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find full local backup timestamp for backup-dir test"
    exit 1
fi

report_dir="/data/master/gpseg-1"

report_output=$(run_command "local_with_backup_dir_console" --timestamp "$timestamp" --backup-dir "${report_dir}")

echo "$report_output" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
echo "$report_output" | grep -q "timestamp key:.*$timestamp" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
echo "$report_output" | grep -q "plugin executable:.*None" || { echo "[ERROR] Expected 'plugin executable: None' for local backup"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Test 3: Get report info for full s3 backup (without using plugin-report-file-path)

test_id=3

echo "[INFO] ${COMMAND} TEST ${test_id}"

timestamp=$(get_backup_info "get_full_s3" --type full | grep -E '^[[:space:]][0-9]{14} ' | grep plugin | head -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find full s3 backup timestamp"
    exit 1
fi

report_output=$(run_command "s3_without_plugin_report_file_path" --timestamp "$timestamp" --plugin-config ~/gpbackup_s3_plugin.yaml)

echo "$report_output" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
echo "$report_output" | grep -q "timestamp key:.*$timestamp" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
echo "$report_output" | grep -q "plugin executable:.*gpbackup_s3_plugin" || { echo "[ERROR] Expected 'plugin executable: gpbackup_s3_plugin' for s3 backup"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

################################################################
# Test 4: Get report info for full s3 backup (with using plugin-report-file-path)

test_id=4

echo "[INFO] ${COMMAND} TEST ${test_id}"

timestamp=$(get_backup_info "get_full_s3" --type full | grep -E '^[[:space:]][0-9]{14} ' | grep plugin | head -1 | awk '{print $1}')

if [ -z "$timestamp" ]; then
    echo "[ERROR] Could not find full s3 backup timestamp for plugin-report-file-path test"
    exit 1
fi

report_dir="/backup/test/backups/${timestamp:0:8}/${timestamp}"

report_output=$(run_command "s3_with_plugin_report_file_path" --timestamp "$timestamp" --plugin-config ~/gpbackup_s3_plugin.yaml --plugin-report-file-path ${report_dir})

echo "$report_output" | grep -q "^Greenplum Database Backup Report" || { echo "[ERROR] Expected report header"; exit 1; }
echo "$report_output" | grep -q "timestamp key:.*$timestamp" || { echo "[ERROR] Expected timestamp key in report"; exit 1; }
echo "$report_output" | grep -q "plugin executable:.*gpbackup_s3_plugin" || { echo "[ERROR] Expected 'plugin executable: gpbackup_s3_plugin' for s3 backup"; exit 1; }

echo "[INFO] ${COMMAND} TEST ${test_id} is successful"

echo "[INFO] ${COMMAND} all tests passed"
