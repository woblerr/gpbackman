package cmd

import (
	"database/sql"

	"github.com/greenplum-db/gpbackup/utils"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

type backupDeleteInterface interface {
	backupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History) error
	backupDeleteDB(backupName string, hDB *sql.DB) error
}

type backupPluginDeleter struct {
	pluginConfigPath string
	pluginConfig     *utils.PluginConfig
}

func (bpd *backupPluginDeleter) backupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History) error {
	return backupDeleteFilePluginFunc(backupData, parseHData, bpd.pluginConfigPath, bpd.pluginConfig)
}

func (bpd *backupPluginDeleter) backupDeleteDB(backupName string, hDB *sql.DB) error {
	return backupDeleteDBPluginFunc(backupName, bpd.pluginConfigPath, bpd.pluginConfig, hDB)
}

type backupLocalDeleter struct {
	backupDir            string
	maxParallelProcesses int
}

func (bld *backupLocalDeleter) backupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History) error {
	return backupDeleteFileLocalFunc(backupData, parseHData, bld.backupDir, bld.maxParallelProcesses)
}

func (bld *backupLocalDeleter) backupDeleteDB(backupName string, hDB *sql.DB) error {
	return backupDeleteDBLocalFunc(backupName, bld.backupDir, bld.maxParallelProcesses, hDB)
}
