package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/bytebase/bytebase"
	"github.com/bytebase/bytebase/api"
	"go.uber.org/zap"
)

const (
	backupRunnerInterval = time.Duration(10) * time.Minute
)

// NewBackupRunner creates a new backup runner.
func NewBackupRunner(logger *zap.Logger, server *Server) *BackupRunner {
	return &BackupRunner{
		l:      logger,
		server: server,
	}
}

// BackupRunner is the backup runner scheduling automatic backups.
type BackupRunner struct {
	l      *zap.Logger
	server *Server
}

// Run is the runner for backup runner.
func (s *BackupRunner) Run() error {
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						err, ok := r.(error)
						if !ok {
							err = fmt.Errorf("%v", r)
						}
						s.l.Error("Backup runner PANIC RECOVER", zap.Error(err))
					}
				}()
				// Find all databases that need a backup in this hour.
				t := time.Now().UTC().Truncate(time.Hour)

				match := &api.BackupSettingsMatch{
					Hour:      t.Hour(),
					DayOfWeek: int(t.Weekday()),
				}
				uniqueKey := fmt.Sprintf("%v", t.Unix())
				list, err := s.server.BackupService.GetBackupSettingsMatch(context.Background(), match)
				if err != nil {
					s.l.Error("Failed to retrieve backup settings match", zap.Error(err))
				}

				for _, backupSetting := range list {
					databaseFind := &api.DatabaseFind{
						ID: &backupSetting.DatabaseId,
					}
					database, err := s.server.ComposeDatabaseByFind(context.Background(), databaseFind)
					if err != nil {
						s.l.Error("Failed to get database for backup setting",
							zap.Int("id", backupSetting.ID),
							zap.String("databaseID", fmt.Sprintf("%v", backupSetting.DatabaseId)),
							zap.String("error", err.Error()))
						continue
					}
					backupSetting.Database = database

					go func(backupSetting *api.BackupSetting, uniqueKey string) {
						if err := s.scheduleBackupTask(backupSetting, uniqueKey); err != nil {
							s.l.Error("Failed to create automatic backup for database",
								zap.Int("id", backupSetting.ID),
								zap.String("databaseID", fmt.Sprintf("%v", backupSetting.DatabaseId)),
								zap.String("error", err.Error()))
						}
					}(backupSetting, uniqueKey)
				}
			}()

			time.Sleep(backupRunnerInterval)
		}
	}()

	return nil
}

func (s *BackupRunner) scheduleBackupTask(backupSetting *api.BackupSetting, uniqueKey string) error {
	key := fmt.Sprintf("auto-backup-%s-%v", uniqueKey, backupSetting.DatabaseId)
	backupCreate := &api.BackupCreate{
		CreatorId:      api.SYSTEM_BOT_ID,
		DatabaseId:     backupSetting.DatabaseId,
		Name:           key,
		Status:         string(api.BackupStatusPendingCreate),
		Type:           string(api.BackupTypeAutomatic),
		StorageBackend: string(api.BackupStorageBackendLocal),
		Path:           fmt.Sprintf("%s.sql", key),
		Comment:        fmt.Sprintf("Automatic backup for database %s at %s", backupSetting.Database.Name, uniqueKey),
	}

	backup, err := s.server.BackupService.CreateBackup(context.Background(), backupCreate)
	if err != nil {
		if bytebase.ErrorCode(err) == bytebase.ECONFLICT {
			// Automatic backup already exists.
			return nil
		}
		return fmt.Errorf("failed to create backup: %v", err)
	}

	payload := api.TaskDatabaseBackupPayload{
		BackupID: backup.ID,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to create task payload: %v", err)
	}

	createdPipeline, err := s.server.PipelineService.CreatePipeline(context.Background(), &api.PipelineCreate{
		Name:      key,
		CreatorId: backupCreate.CreatorId,
	})
	if err != nil {
		return fmt.Errorf("failed to create pipeline: %v", err)
	}

	createdStage, err := s.server.StageService.CreateStage(context.Background(), &api.StageCreate{
		Name:          key,
		EnvironmentId: backupSetting.Database.Instance.EnvironmentId,
		PipelineId:    createdPipeline.ID,
		CreatorId:     backupCreate.CreatorId,
	})
	if err != nil {
		return fmt.Errorf("failed to create stage: %v", err)
	}

	_, err = s.server.TaskService.CreateTask(context.Background(), &api.TaskCreate{
		Name:       key,
		PipelineId: createdPipeline.ID,
		StageId:    createdStage.ID,
		InstanceId: backupSetting.Database.InstanceId,
		DatabaseId: &backupSetting.Database.ID,
		Status:     api.TaskPending,
		Type:       api.TaskDatabaseBackup,
		Payload:    string(bytes),
		CreatorId:  backupCreate.CreatorId,
	})
	if err != nil {
		return fmt.Errorf("failed to create task: %v", err)
	}
	return nil
}