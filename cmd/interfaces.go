package cmd

import (
	"github.com/greenplum-db/gpbackup/utils"
	"github.com/woblerr/gpbackman/gpbckpconfig"
)

type BackupDeleteFileInterface interface {
	BackupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History) error
}

type BackupPluginDeleter struct {
	pluginConfigPath string
	pluginConfig     *utils.PluginConfig
}

func (bpd *BackupPluginDeleter) BackupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History) error {
	return backupDeleteFilePluginFunc(backupData, parseHData, bpd.pluginConfigPath, bpd.pluginConfig)
}

type BackupLocalDeleter struct {
	backupDir            string
	maxParallelProcesses int
}

func (bld *BackupLocalDeleter) BackupDeleteFile(backupData gpbckpconfig.BackupConfig, parseHData *gpbckpconfig.History) error {
	return backupDeleteFileLocalFunc(backupData, parseHData, bld.backupDir, bld.maxParallelProcesses)
}
