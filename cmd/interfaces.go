package cmd

import (
	"database/sql"

	"github.com/greenplum-db/gpbackup/utils"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

type backupDeleteInterface interface {
	backupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History, ignoreErrors bool) error
	backupDeleteDB(backupName string, hDB *sql.DB, ignoreErrors bool) error
}

type backupPluginDeleter struct {
	pluginConfigPath string
	pluginConfig     *utils.PluginConfig
}

func (bpd *backupPluginDeleter) backupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History, ignoreErrors bool) error {
	return backupDeleteFilePluginFunc(backupData, parseHData, bpd.pluginConfigPath, bpd.pluginConfig, ignoreErrors)
}

func (bpd *backupPluginDeleter) backupDeleteDB(backupName string, hDB *sql.DB, ignoreErrors bool) error {
	return backupDeleteDBPluginFunc(backupName, bpd.pluginConfigPath, bpd.pluginConfig, hDB, ignoreErrors)
}

type backupLocalDeleter struct {
	backupDir            string
	maxParallelProcesses int
}

func (bld *backupLocalDeleter) backupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History, ignoreErrors bool) error {
	return backupDeleteFileLocalFunc(backupData, parseHData, bld.backupDir, bld.maxParallelProcesses, ignoreErrors)
}

func (bld *backupLocalDeleter) backupDeleteDB(backupName string, hDB *sql.DB, ignoreErrors bool) error {
	return backupDeleteDBLocalFunc(backupName, bld.backupDir, bld.maxParallelProcesses, hDB, ignoreErrors)
}
