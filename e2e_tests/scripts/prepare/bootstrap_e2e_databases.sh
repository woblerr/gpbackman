#!/usr/bin/env bash
set -Eeuo pipefail

HOME_DIR="/home/gpadmin"
DATABASE_CONFIG="${HOME_DIR}/e2e_databases.sh"
FILTER_FIXTURE="${HOME_DIR}/e2e_filter_init.sql"

# shellcheck source=/dev/null
source "${DATABASE_CONFIG}"

database_exists() {
    local database="$1"

    psql \
        --dbname postgres \
        --no-psqlrc \
        --tuples-only \
        --no-align \
        --set ON_ERROR_STOP=1 \
        --set database_name="${database}" \
        --command "SELECT 1 FROM pg_database WHERE datname = :'database_name';"
}

assert_primary_table_nonempty() {
    local schema="$1"
    local table="$2"
    local result

    if ! result=$(psql \
        --dbname "${E2E_PRIMARY_DB}" \
        --no-psqlrc \
        --tuples-only \
        --no-align \
        --set ON_ERROR_STOP=1 \
        --set schema_name="${schema}" \
        --set table_name="${table}" \
        --command 'SELECT 1 FROM :"schema_name".:"table_name" LIMIT 1;' 2>&1); then
        echo "[ERROR] Required table ${schema}.${table} is missing from ${E2E_PRIMARY_DB} or cannot be queried"
        echo "${result}"
        return 1
    fi

    if [ "${result}" != "1" ]; then
        echo "[ERROR] Required table ${schema}.${table} in ${E2E_PRIMARY_DB} is empty"
        return 1
    fi
}

assert_filter_fixture() {
    local row_count

    if ! row_count=$(psql \
        --dbname "${E2E_FILTER_DB}" \
        --no-psqlrc \
        --tuples-only \
        --no-align \
        --set ON_ERROR_STOP=1 \
        --command "SELECT count(*) FROM public.e2e_data;" 2>&1); then
        echo "[ERROR] Failed to query public.e2e_data in ${E2E_FILTER_DB}"
        echo "${row_count}"
        return 1
    fi

    if [ "${row_count}" != "100" ]; then
        echo "[ERROR] Expected 100 rows in ${E2E_FILTER_DB}.public.e2e_data, got ${row_count}"
        return 1
    fi
}

if [ "$(database_exists "${E2E_PRIMARY_DB}")" != "1" ]; then
    echo "[ERROR] Required primary database ${E2E_PRIMARY_DB} is missing"
    exit 1
fi

assert_primary_table_nonempty sch1 tbl_a
assert_primary_table_nonempty sch1 tbl_b
assert_primary_table_nonempty sch2 tbl_c
assert_primary_table_nonempty sch2 tbl_d

if [ "$(database_exists "${E2E_FILTER_DB}")" != "1" ]; then
    echo "[INFO] Creating filter database ${E2E_FILTER_DB}"
    if ! createdb "${E2E_FILTER_DB}"; then
        echo "[ERROR] Failed to create filter database ${E2E_FILTER_DB}"
        exit 1
    fi
fi

if ! psql \
    --dbname "${E2E_FILTER_DB}" \
    --no-psqlrc \
    --set ON_ERROR_STOP=1 \
    --file "${FILTER_FIXTURE}"; then
    echo "[ERROR] Failed to apply fixture to filter database ${E2E_FILTER_DB}"
    exit 1
fi

assert_filter_fixture
echo "[INFO] E2E databases bootstrapped successfully"
