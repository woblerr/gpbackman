package cmd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/greenplum-db/gp-common-go-libs/gplog"
	"github.com/greenplum-db/gp-common-go-libs/operating"
	_ "github.com/mattn/go-sqlite3"
	"github.com/woblerr/gpbackman/gpbckpconfig"
	"github.com/woblerr/gpbackman/textmsg"
)

const (
	// Leave enough time for the 30-second SSH connection timeout and remote removal
	// while keeping failure cleanup bounded.
	historyStandbySyncCleanupTimeout = 120 * time.Second
	// Cap the timeout at one day to catch accidentally oversized CLI values.
	// A longer transport deadline is not meaningful for standby history synchronization.
	historySyncStandbyTimeoutMax = int(24 * time.Hour / time.Second)
)

type historyStandbySyncTarget struct {
	sourceDBPath         string
	standbyHost          string
	standbyDataDir       string
	standbyHistoryDBPath string
}

var (
	historyStandbySyncNewClusterConn     = gpbckpconfig.NewClusterLocalClusterConnWithDefault
	historyStandbySyncTempDir            = os.TempDir
	historyStandbySyncNow                = time.Now
	historyStandbySyncPID                = os.Getpid
	historyStandbySyncTimeoutSeconds     = historySyncStandbyTimeoutDefault
	historyStandbySyncContextWithTimeout = context.WithTimeout
	historyStandbySyncExecCommand        = func(ctx context.Context, name string, args ...string) ([]byte, error) {
		// #nosec G204 -- command names and arguments come from internal standby sync call sites.
		return exec.CommandContext(ctx, name, args...).CombinedOutput()
	}
	historyStandbySyncCurrentUser = func() (string, error) {
		currentUser, err := operating.System.CurrentUser()
		if err != nil {
			return "", err
		}
		return currentUser.Username, nil
	}
)

func syncHistoryStandbyBestEffort(clusterDBName string, noHistorySyncStandby bool) {
	if noHistorySyncStandby {
		gplog.Info("%s", textmsg.InfoTextHistoryStandbySyncSkipped("disabled by --"+noHistorySyncStandbyFlagName))
		return
	}
	skipReason, err := syncHistoryStandby(clusterDBName)
	if err != nil {
		gplog.Warn("%s", textmsg.WarnTextHistoryStandbySyncFailed(err))
		return
	}
	if skipReason != "" {
		gplog.Debug("%s", textmsg.InfoTextHistoryStandbySyncSkipped(skipReason))
	}
}

func runHistoryMutationWithStandbySync(work func() (string, error), noHistorySyncStandby bool) {
	clusterDBName, err := work()
	if err != nil {
		execOSExit(exitErrorCode)
		return
	}
	syncHistoryStandbyBestEffort(clusterDBName, noHistorySyncStandby)
}

func syncHistoryStandby(clusterDBName string) (string, error) {
	sourceDBPath, shouldDiscover, skipReason := getHistoryStandbySyncSourceDBPath()
	if !shouldDiscover {
		return skipReason, nil
	}

	target, skipReason, err := discoverHistoryStandbySyncTarget(sourceDBPath, clusterDBName)
	if err != nil {
		return "", err
	}
	if skipReason != "" {
		return skipReason, nil
	}
	gplog.Info("%s", textmsg.InfoTextHistoryStandbySyncStart(target.sourceDBPath))

	if err := withHistoryStandbySyncSnapshot(target.sourceDBPath, func(snapshotPath string) error {
		userName, err := historyStandbySyncCurrentUser()
		if err != nil {
			return fmt.Errorf("resolve current OS user for standby history sync: %w", err)
		}

		ctx, cancel := historyStandbySyncContextWithTimeout(
			context.Background(),
			time.Duration(historyStandbySyncTimeoutSeconds)*time.Second,
		)
		defer cancel()

		transportErr := syncHistoryStandbySnapshotToStandby(ctx, target, userName, snapshotPath)
		if transportErr != nil && errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf(
				"standby history sync transport timed out after %d seconds: %w",
				historyStandbySyncTimeoutSeconds,
				transportErr,
			)
		}
		return transportErr
	}); err != nil {
		return "", err
	}

	gplog.Info("%s", textmsg.InfoTextHistoryStandbySyncSuccess(target.standbyHost, target.standbyHistoryDBPath))
	return "", nil
}

// Standby sync is allowed only for the cluster history db.
// Explicit --history-db must still point to gpbackup_history.db under the primary data dir.
// --auto-load-history-db is allowed only when it resolves from the coordinator data dir env.
// The default working-directory gpbackup_history.db is treated as local/custom and is not synced.
func getHistoryStandbySyncSourceDBPath() (string, bool, string) {
	path := getHistoryDBPath(rootHistoryDB, rootAutoLoadHistoryDB)
	switch {
	case rootHistoryDB != "":
		return getExplicitHistoryStandbySyncSourceDBPath(path)
	case rootAutoLoadHistoryDB:
		if path != historyDBNameConst {
			return path, true, ""
		}
		return path, false, "--auto-load-history-db did not find MASTER_DATA_DIRECTORY or COORDINATOR_DATA_DIRECTORY"
	default:
		return path, false, "using default working-directory history db"
	}
}

func getExplicitHistoryStandbySyncSourceDBPath(path string) (string, bool, string) {
	if dataDir := getHistoryStandbySyncDataDirFromEnv(); dataDir != "" {
		eligible, normalizedSourcePath, normalizedPrimaryHistoryDBPath, err := checkHistoryStandbySyncSourcePathEligible(path, dataDir)
		if err != nil {
			return path, false, fmt.Sprintf("unable to check explicit history db path eligibility: %v", err)
		}
		if !eligible {
			return path, false, fmt.Sprintf("source history db %s is not cluster history db %s", normalizedSourcePath, normalizedPrimaryHistoryDBPath)
		}
		return path, true, ""
	}
	if filepath.Base(filepath.Clean(path)) != historyDBNameConst {
		return path, false, fmt.Sprintf("source history db %s is not named %s", path, historyDBNameConst)
	}
	return path, true, ""
}

func getHistoryStandbySyncDataDirFromEnv() string {
	for _, envVar := range historyDBEnvVars {
		if dataDir := os.Getenv(envVar); dataDir != "" {
			return dataDir
		}
	}
	return ""
}

func discoverHistoryStandbySyncTarget(sourceDBPath, clusterDBName string) (*historyStandbySyncTarget, string, error) {
	db, err := historyStandbySyncNewClusterConn(clusterDBName)
	if err != nil {
		return nil, "", fmt.Errorf("connect to local cluster for standby history sync discovery: %w", err)
	}
	defer db.Close()

	primaryDataDir, err := gpbckpconfig.GetPrimaryCoordinatorDataDirLocalClusterConn(db)
	if err != nil {
		return nil, "", fmt.Errorf("query primary coordinator datadir for standby history sync discovery: %w", err)
	}

	eligible, normalizedSourcePath, normalizedPrimaryHistoryDBPath, err := checkHistoryStandbySyncSourcePathEligible(sourceDBPath, primaryDataDir)
	if err != nil {
		return nil, "", fmt.Errorf("check standby history sync source path eligibility: %w", err)
	}
	if !eligible {
		skipReason := fmt.Sprintf("source history db %s is not cluster history db %s", normalizedSourcePath, normalizedPrimaryHistoryDBPath)
		return nil, skipReason, nil
	}

	standbyConfig, err := gpbckpconfig.GetUpStandbyCoordinatorLocalClusterConn(db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "no up standby coordinator found", nil
		}
		return nil, "", fmt.Errorf("query up standby coordinator for standby history sync discovery: %w", err)
	}

	target := &historyStandbySyncTarget{
		sourceDBPath:         normalizedSourcePath,
		standbyHost:          standbyConfig.Hostname,
		standbyDataDir:       standbyConfig.DataDir,
		standbyHistoryDBPath: filepath.Join(standbyConfig.DataDir, historyDBNameConst),
	}
	gplog.Debug(
		"Discovered standby history sync target: source=%s standby=%s:%s",
		target.sourceDBPath,
		target.standbyHost,
		target.standbyHistoryDBPath,
	)
	return target, "", nil
}

func checkHistoryStandbySyncSourcePathEligible(sourceDBPath, primaryDataDir string) (bool, string, string, error) {
	normalizedSourcePath, err := normalizeHistoryStandbySyncPath(sourceDBPath)
	if err != nil {
		return false, "", "", err
	}
	normalizedPrimaryHistoryDBPath, err := normalizeHistoryStandbySyncPath(filepath.Join(primaryDataDir, historyDBNameConst))
	if err != nil {
		return false, "", "", err
	}
	return normalizedSourcePath == normalizedPrimaryHistoryDBPath, normalizedSourcePath, normalizedPrimaryHistoryDBPath, nil
}

func normalizeHistoryStandbySyncPath(path string) (string, error) {
	absPath, err := filepath.Abs(filepath.Clean(path))
	if err != nil {
		return "", err
	}
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		return normalizeHistoryStandbySyncMissingPath(absPath)
	}
	return filepath.Clean(resolvedPath), nil
}

func normalizeHistoryStandbySyncMissingPath(absPath string) (string, error) {
	resolvedDir, err := filepath.EvalSymlinks(filepath.Dir(absPath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return absPath, nil
		}
		return "", err
	}
	return filepath.Join(resolvedDir, filepath.Base(absPath)), nil
}

func withHistoryStandbySyncSnapshot(sourceDBPath string, syncFn func(string) error) error {
	snapshotPath, err := createHistoryStandbySyncSnapshot(sourceDBPath)
	if err != nil {
		return err
	}
	defer cleanupHistoryStandbySyncSnapshot(snapshotPath)
	return syncFn(snapshotPath)
}

func createHistoryStandbySyncSnapshot(sourceDBPath string) (string, error) {
	snapshotPath, err := newHistoryStandbySyncSnapshotPath()
	if err != nil {
		return "", err
	}
	if err := ensureHistoryStandbySyncSnapshotDoesNotExist(snapshotPath); err != nil {
		return "", err
	}
	if err := createHistoryStandbySyncSnapshotFile(sourceDBPath, snapshotPath); err != nil {
		cleanupHistoryStandbySyncSnapshot(snapshotPath)
		return "", err
	}
	if err := validateHistoryStandbySyncSnapshot(snapshotPath); err != nil {
		cleanupHistoryStandbySyncSnapshot(snapshotPath)
		return "", err
	}
	return snapshotPath, nil
}

func newHistoryStandbySyncSnapshotPath() (string, error) {
	tempDir, err := filepath.Abs(historyStandbySyncTempDir())
	if err != nil {
		return "", err
	}
	now := historyStandbySyncNow().UTC()
	timestamp := fmt.Sprintf("%s_%09d", now.Format("20060102150405"), now.Nanosecond())
	filename := fmt.Sprintf("%s_%s_%d%s.snap", historyFileNameBaseConst, timestamp, historyStandbySyncPID(), historyFileDBSuffixConst)
	return filepath.Join(tempDir, filename), nil
}

func ensureHistoryStandbySyncSnapshotDoesNotExist(snapshotPath string) error {
	if _, err := os.Stat(snapshotPath); err == nil {
		return fmt.Errorf("create standby history sync snapshot: target already exists: %s", snapshotPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat standby history sync snapshot target %s: %w", snapshotPath, err)
	}
	return nil
}

func createHistoryStandbySyncSnapshotFile(sourceDBPath, snapshotPath string) error {
	sourceInfo, err := os.Stat(sourceDBPath)
	if err != nil {
		return fmt.Errorf("stat source history db for standby sync snapshot attrs: %w", err)
	}
	if err := vacuumHistoryStandbySyncSnapshot(sourceDBPath, snapshotPath); err != nil {
		return err
	}
	if err := os.Chmod(snapshotPath, sourceInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("set standby history sync snapshot permissions from source history db: %w", err)
	}
	return nil
}

func vacuumHistoryStandbySyncSnapshot(sourceDBPath, snapshotPath string) error {
	sourceDB, err := sql.Open("sqlite3", historyStandbySyncSQLiteURI(sourceDBPath, "ro"))
	if err != nil {
		return fmt.Errorf("open source history db for standby sync snapshot: %w", err)
	}
	defer sourceDB.Close()

	if _, err := sourceDB.Exec("VACUUM main INTO ?", snapshotPath); err != nil {
		return fmt.Errorf("create standby history sync snapshot with VACUUM INTO: %w", err)
	}
	return nil
}

func validateHistoryStandbySyncSnapshot(snapshotPath string) error {
	results, err := runHistoryStandbySyncQuickCheck(snapshotPath)
	if err != nil {
		return err
	}
	if err := validateHistoryStandbySyncQuickCheckResults(results); err != nil {
		return fmt.Errorf("validate standby history sync snapshot quick_check: %w", err)
	}
	return nil
}

func runHistoryStandbySyncQuickCheck(snapshotPath string) ([]string, error) {
	snapshotDB, err := sql.Open("sqlite3", historyStandbySyncSQLiteURI(snapshotPath, "ro"))
	if err != nil {
		return nil, fmt.Errorf("open standby history sync snapshot read-only: %w", err)
	}
	defer snapshotDB.Close()

	rows, err := snapshotDB.Query("PRAGMA quick_check")
	if err != nil {
		return nil, fmt.Errorf("run PRAGMA quick_check on standby history sync snapshot: %w", err)
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var result string
		if err := rows.Scan(&result); err != nil {
			return nil, fmt.Errorf("scan PRAGMA quick_check result for standby history sync snapshot: %w", err)
		}
		results = append(results, result)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read PRAGMA quick_check results for standby history sync snapshot: %w", err)
	}
	return results, nil
}

func validateHistoryStandbySyncQuickCheckResults(results []string) error {
	if len(results) != 1 || results[0] != "ok" {
		return fmt.Errorf("expected single ok result, got %v", results)
	}
	return nil
}

func cleanupHistoryStandbySyncSnapshot(snapshotPath string) {
	if err := os.Remove(snapshotPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		gplog.Debug("Unable to remove local standby history sync snapshot %s: %v", snapshotPath, err)
	}
}

func syncHistoryStandbySnapshotToStandby(
	ctx context.Context,
	target *historyStandbySyncTarget,
	userName, snapshotPath string,
) error {
	remoteTempPath := newHistoryStandbySyncRemoteTempPath(target.standbyDataDir, snapshotPath)
	if err := rsyncHistoryStandbySyncSnapshot(ctx, snapshotPath, target.standbyHost, userName, remoteTempPath); err != nil {
		return cleanupHistoryStandbySyncRemoteTempAfterError(err, target.standbyHost, userName, remoteTempPath)
	}
	if err := installHistoryStandbySyncSnapshotOnStandby(ctx, target, userName, remoteTempPath); err != nil {
		return cleanupHistoryStandbySyncRemoteTempAfterError(err, target.standbyHost, userName, remoteTempPath)
	}
	return nil
}

func newHistoryStandbySyncRemoteTempPath(standbyDataDir, snapshotPath string) string {
	return filepath.Join(standbyDataDir, filepath.Base(snapshotPath))
}

func rsyncHistoryStandbySyncSnapshot(
	ctx context.Context,
	snapshotPath, standbyHost, userName, remoteTempPath string,
) error {
	args := buildHistoryStandbySyncRsyncArgs(snapshotPath, standbyHost, userName, remoteTempPath)
	gplog.Debug(
		"Transfer history db snapshot to standby coordinator: %s -> %s:%s",
		snapshotPath,
		standbyHost,
		remoteTempPath,
	)
	output, err := historyStandbySyncExecCommand(ctx, "rsync", args...)
	if ctxErr := ctx.Err(); ctxErr != nil {
		err = ctxErr
	}
	if err != nil {
		return fmt.Errorf(
			"rsync standby history snapshot to %s:%s failed: %w%s",
			standbyHost,
			remoteTempPath,
			err,
			formatHistoryStandbySyncCommandOutput(output),
		)
	}
	return nil
}

func buildHistoryStandbySyncRsyncArgs(snapshotPath, standbyHost, userName, remoteTempPath string) []string {
	return []string{
		"-e",
		"ssh -o BatchMode=yes -o StrictHostKeyChecking=no -o ConnectTimeout=30",
		"--",
		snapshotPath,
		fmt.Sprintf("%s@%s:%s", userName, standbyHost, remoteTempPath),
	}
}

func installHistoryStandbySyncSnapshotOnStandby(
	ctx context.Context,
	target *historyStandbySyncTarget,
	userName, remoteTempPath string,
) error {
	command := buildHistoryStandbySyncRemoteInstallCommand(remoteTempPath, target.standbyHistoryDBPath)
	gplog.Debug(
		"Install history db snapshot on standby coordinator: %s:%s",
		target.standbyHost,
		target.standbyHistoryDBPath,
	)
	output, err := runHistoryStandbySyncSSHCommand(ctx, command, target.standbyHost, userName)
	if err != nil {
		return fmt.Errorf(
			"install standby history snapshot on %s:%s failed: %w%s",
			target.standbyHost,
			target.standbyHistoryDBPath,
			err,
			formatHistoryStandbySyncCommandOutput(output),
		)
	}
	return nil
}

func buildHistoryStandbySyncRemoteInstallCommand(remoteTempPath, standbyHistoryDBPath string) string {
	quotedTempPath := shellQuote(remoteTempPath)
	quotedHistoryDBPath := shellQuote(standbyHistoryDBPath)
	return fmt.Sprintf(
		"if [ -e %s ]; then chown --reference=%s -- %s 2>/dev/null || true; chmod --reference=%s -- %s 2>/dev/null || true; fi; mv -f -- %s %s",
		quotedHistoryDBPath,
		quotedHistoryDBPath,
		quotedTempPath,
		quotedHistoryDBPath,
		quotedTempPath,
		quotedTempPath,
		quotedHistoryDBPath,
	)
}

func cleanupHistoryStandbySyncRemoteTempAfterError(primaryErr error, standbyHost, userName, remoteTempPath string) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), historyStandbySyncCleanupTimeout)
	defer cancel()

	if cleanupErr := cleanupHistoryStandbySyncRemoteTemp(cleanupCtx, standbyHost, userName, remoteTempPath); cleanupErr != nil {
		return fmt.Errorf("%w; additionally failed to clean up remote temp file: %w", primaryErr, cleanupErr)
	}
	return primaryErr
}

func cleanupHistoryStandbySyncRemoteTemp(
	ctx context.Context,
	standbyHost, userName, remoteTempPath string,
) error {
	command := buildHistoryStandbySyncRemoteCleanupCommand(remoteTempPath)
	gplog.Debug("Clean up remote standby history sync temp file: %s:%s", standbyHost, remoteTempPath)
	output, err := runHistoryStandbySyncSSHCommand(ctx, command, standbyHost, userName)
	if err != nil {
		return fmt.Errorf(
			"clean up remote standby history sync temp file %s:%s failed: %w%s",
			standbyHost,
			remoteTempPath,
			err,
			formatHistoryStandbySyncCommandOutput(output),
		)
	}
	return nil
}

func buildHistoryStandbySyncRemoteCleanupCommand(remoteTempPath string) string {
	return fmt.Sprintf("rm -f -- %s", shellQuote(remoteTempPath))
}

func runHistoryStandbySyncSSHCommand(
	ctx context.Context,
	remoteCommand, standbyHost, userName string,
) ([]byte, error) {
	output, err := historyStandbySyncExecCommand(
		ctx,
		"ssh",
		"-o",
		"BatchMode=yes",
		"-o",
		"StrictHostKeyChecking=no",
		"-o",
		"ConnectTimeout=30",
		fmt.Sprintf("%s@%s", userName, standbyHost),
		remoteCommand,
	)
	if ctxErr := ctx.Err(); ctxErr != nil {
		return output, ctxErr
	}
	return output, err
}

func historyStandbySyncSQLiteURI(dbPath, mode string) string {
	query := url.Values{}
	query.Set("mode", mode)
	dbURI := url.URL{Scheme: "file", Path: dbPath, RawQuery: query.Encode()}
	return dbURI.String()
}

func shellQuote(value string) string {
	if value == "" {
		return "''"
	}
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
}

func formatHistoryStandbySyncCommandOutput(output []byte) string {
	trimmedOutput := strings.TrimSpace(string(output))
	if trimmedOutput == "" {
		return ""
	}
	return ": " + trimmedOutput
}
