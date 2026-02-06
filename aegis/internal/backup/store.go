package backup

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/bigdegenenergy/open-cloud-ops/aegis/pkg/models"
)

// BackupStore defines the persistence interface for backup jobs and records.
// Implementations must be safe for concurrent use.
type BackupStore interface {
	SaveJob(ctx context.Context, job *models.BackupJob) error
	GetJob(ctx context.Context, id string) (*models.BackupJob, error)
	ListJobs(ctx context.Context) ([]*models.BackupJob, error)

	SaveRecord(ctx context.Context, record *models.BackupRecord) error
	GetRecord(ctx context.Context, id string) (*models.BackupRecord, error)
	ListRecordsByJob(ctx context.Context, jobID string) ([]*models.BackupRecord, error)
	ListAllRecords(ctx context.Context) ([]*models.BackupRecord, error)
	DeleteRecord(ctx context.Context, id string) error
}

// scannable is satisfied by both pgx.Row and pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

// PgStore implements BackupStore using PostgreSQL via pgxpool.
type PgStore struct {
	pool *pgxpool.Pool
}

// NewPgStore creates a new PostgreSQL-backed backup store.
func NewPgStore(pool *pgxpool.Pool) *PgStore {
	return &PgStore{pool: pool}
}

const jobCols = `id, name, namespace, resource_types, schedule,
	retention_days, storage_location, status, last_run, next_run, created_at`

const recordCols = `id, job_id, status, size_bytes, duration_ms,
	resource_count, storage_path, error_message, started_at, completed_at`

// SaveJob inserts or updates a backup job.
func (s *PgStore) SaveJob(ctx context.Context, job *models.BackupJob) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO backup_jobs (`+jobCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (id) DO UPDATE SET
			name=$2, namespace=$3, resource_types=$4, schedule=$5,
			retention_days=$6, storage_location=$7, status=$8,
			last_run=$9, next_run=$10`,
		job.ID, job.Name, job.Namespace, job.ResourceTypes, job.Schedule,
		job.RetentionDays, job.StorageLocation, string(job.Status),
		job.LastRun, job.NextRun, job.CreatedAt)
	if err != nil {
		return fmt.Errorf("pgstore: save job: %w", err)
	}
	return nil
}

// GetJob retrieves a backup job by ID.
func (s *PgStore) GetJob(ctx context.Context, id string) (*models.BackupJob, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+jobCols+` FROM backup_jobs WHERE id = $1`, id)
	return scanJob(row)
}

// ListJobs returns all backup jobs ordered by creation time.
func (s *PgStore) ListJobs(ctx context.Context) ([]*models.BackupJob, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+jobCols+` FROM backup_jobs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*models.BackupJob
	for rows.Next() {
		job, scanErr := scanJob(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

// SaveRecord inserts or updates a backup record.
func (s *PgStore) SaveRecord(ctx context.Context, r *models.BackupRecord) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO backup_records (`+recordCols+`)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (id) DO UPDATE SET
			status=$3, size_bytes=$4, duration_ms=$5, resource_count=$6,
			storage_path=$7, error_message=$8, completed_at=$10`,
		r.ID, r.JobID, string(r.Status), r.SizeBytes, r.DurationMs,
		r.ResourceCount, r.StoragePath, r.ErrorMessage,
		r.StartedAt, r.CompletedAt)
	if err != nil {
		return fmt.Errorf("pgstore: save record: %w", err)
	}
	return nil
}

// GetRecord retrieves a backup record by ID.
func (s *PgStore) GetRecord(ctx context.Context, id string) (*models.BackupRecord, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT `+recordCols+` FROM backup_records WHERE id = $1`, id)
	return scanRecord(row)
}

// ListRecordsByJob returns all records for a given job.
func (s *PgStore) ListRecordsByJob(ctx context.Context, jobID string) ([]*models.BackupRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+recordCols+` FROM backup_records WHERE job_id = $1 ORDER BY started_at DESC`,
		jobID)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list records by job: %w", err)
	}
	defer rows.Close()

	var records []*models.BackupRecord
	for rows.Next() {
		r, scanErr := scanRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// ListAllRecords returns all backup records across all jobs.
func (s *PgStore) ListAllRecords(ctx context.Context) ([]*models.BackupRecord, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+recordCols+` FROM backup_records ORDER BY started_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list all records: %w", err)
	}
	defer rows.Close()

	var records []*models.BackupRecord
	for rows.Next() {
		r, scanErr := scanRecord(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		records = append(records, r)
	}
	return records, rows.Err()
}

// DeleteRecord removes a backup record by ID.
func (s *PgStore) DeleteRecord(ctx context.Context, id string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM backup_records WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("pgstore: delete record: %w", err)
	}
	return nil
}

func scanJob(s scannable) (*models.BackupJob, error) {
	var job models.BackupJob
	var status string
	err := s.Scan(
		&job.ID, &job.Name, &job.Namespace, &job.ResourceTypes,
		&job.Schedule, &job.RetentionDays, &job.StorageLocation,
		&status, &job.LastRun, &job.NextRun, &job.CreatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pgstore: job not found")
		}
		return nil, fmt.Errorf("pgstore: scan job: %w", err)
	}
	job.Status = models.BackupStatus(status)
	return &job, nil
}

func scanRecord(s scannable) (*models.BackupRecord, error) {
	var r models.BackupRecord
	var status string
	err := s.Scan(
		&r.ID, &r.JobID, &status, &r.SizeBytes, &r.DurationMs,
		&r.ResourceCount, &r.StoragePath, &r.ErrorMessage,
		&r.StartedAt, &r.CompletedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("pgstore: record not found")
		}
		return nil, fmt.Errorf("pgstore: scan record: %w", err)
	}
	r.Status = models.RecordStatus(status)
	return &r, nil
}

// logStoreErr logs a store persistence error without failing the operation.
// This allows the in-memory state to remain authoritative while the store
// catches up on the next successful write.
func logStoreErr(operation string, err error) {
	if err != nil {
		log.Printf("backup: warning: store %s failed: %v", operation, err)
	}
}
