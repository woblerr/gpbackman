#!/usr/bin/env bash
set -Eeuo pipefail

# Backup sequence overview:
# 1.  full_local               : Full LOCAL backup (all tables)
# 2.  full_local_include_table : Full LOCAL backup including only sch1.tbl_a
# 3.  full_local_exclude_table : Full LOCAL backup excluding sch1.tbl_b
# 4.  metadata_only_s3         : Metadata-only S3 backup (no data)
# 5.  full_s3                  : Full S3 backup (all tables, leaf partition data)
# 6.  full_s3_include_tables   : Full S3 backup including sch2.tbl_c, sch2.tbl_d
# 7.  full_s3_exclude_schema   : Full S3 backup excluding schema sch1
# 8.  (data change)            : Insert into sch2.tbl_c and sch2.tbl_d
# 9.  incr_s3                  : Incremental S3 backup
# 10. incr_s3_include_tables   : Incremental S3 backup including sch2.tbl_c, sch2.tbl_d
# 11. (data change)            : Insert more rows into sch2.tbl_c
# 12. incr_s3_exclude_schema   : Incremental S3 backup excluding schema sch1
# 13. data_only_local          : Data-only LOCAL backup (no metadata)
# 14. full_local               : Final full LOCAL backup (all tables)

DB_NAME="demo"
PLUGIN_CFG=/home/gpadmin/gpbackup_s3_plugin.yaml
COMMON_PLUGIN_FLAGS=(--plugin-config "$PLUGIN_CFG")

run_backup(){
  local label="$1"; shift
  echo "[INFO] Running backup: $label"
  gpbackup --dbname ${DB_NAME} "$@" || { echo "[ERROR] Backup $label failed"; exit 1; }
  sleep 10
}

# Full LOCAL no filters
run_backup full_local

# Full LOCAL include-table sch1.tbl_a
run_backup full_local_include_table --include-table sch1.tbl_a

# Full LOCAL exclude-table sch1.tbl_b
run_backup full_local_exclude_table --exclude-table sch1.tbl_b

# Metadata-only s3
run_backup metadata_only_s3 "${COMMON_PLUGIN_FLAGS[@]}" --metadata-only

# Full S3 no filters
run_backup full_s3 "${COMMON_PLUGIN_FLAGS[@]}" --leaf-partition-data

# Full S3 include-table sch2.tbl_c, sch2.tbl_d
run_backup full_s3_include_table "${COMMON_PLUGIN_FLAGS[@]}" --include-table sch2.tbl_c --include-table sch2.tbl_d --leaf-partition-data

# Full S3 exclude-schema sch1
run_backup full_s3_exclude_schema "${COMMON_PLUGIN_FLAGS[@]}" --exclude-schema sch1 --leaf-partition-data

# Insert data
psql -d demo -c "INSERT INTO sch2.tbl_c SELECT i, i FROM generate_series(1,100000) i;"
psql -d demo -c "INSERT INTO sch2.tbl_d SELECT i, i FROM generate_series(1,100000) AS i;"

# Incremental S3 no filters
run_backup incr_s3 "${COMMON_PLUGIN_FLAGS[@]}" --incremental --leaf-partition-data

# Incremental S3 include-tables sch2.tbl_c, sch2.tbl_d
run_backup incr_s3_include_table "${COMMON_PLUGIN_FLAGS[@]}" --incremental --include-table sch2.tbl_c --include-table sch2.tbl_d --leaf-partition-data

# Insert data
psql -d demo -c "INSERT INTO sch2.tbl_c SELECT i, i FROM generate_series(1,100000) i;"

# Incremental S3 exclude-schema sch1
run_backup incr_s3_exclude_schema "${COMMON_PLUGIN_FLAGS[@]}" --incremental --exclude-schema sch1 --leaf-partition-data

# Data-only LOCAL no filters
run_backup data_only_local --data-only

# Full LOCAL no filters
run_backup full_local

echo "[INFO] Backups prepared successfully"
exit 0
