package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/joho/godotenv/autoload"
)

var (
	database = os.Getenv("BLUEPRINT_DB_DATABASE")
	password = os.Getenv("BLUEPRINT_DB_PASSWORD")
	username = os.Getenv("BLUEPRINT_DB_USERNAME")
	port     = os.Getenv("BLUEPRINT_DB_PORT")
	host     = os.Getenv("BLUEPRINT_DB_HOST")
	schema   = os.Getenv("BLUEPRINT_DB_SCHEMA")
)

type Database struct {
	Pool *pgxpool.Pool
}

func NewDatabase() (*Database, func(), error) {
	connStr := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable&search_path=%s", username, password, host, port, database, schema)
	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, nil, err
	}

	// Performance optimizations
	config.MaxConns = int32(runtime.NumCPU() * 4) // 4x CPU cores
	config.MinConns = int32(runtime.NumCPU())     // Min connections
	config.MaxConnLifetime = time.Hour            // Connection lifetime
	config.MaxConnIdleTime = time.Minute * 30     // Idle timeout
	config.HealthCheckPeriod = time.Minute        // Health check interval

	// Connection-level optimizations
	config.ConnConfig.RuntimeParams = map[string]string{
		"application_name":                    "aktel",
		"search_path":                         "public",
		"timezone":                            "UTC",
		"statement_timeout":                   "30s",
		"lock_timeout":                        "10s",
		"idle_in_transaction_session_timeout": "60s",
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		log.Printf("Disconnected from database: %s", database)
		pool.Close()
	}

	return &Database{Pool: pool}, cleanup, nil
}

type Team struct {
	Id          int64
	Name        string
	FastCupLink string
	LogoLink    string
}

type Submission struct {
	Id                int64
	ExternalId        string
	TournamentId      int64
	TeamId            int64
	SubmittedAt       time.Time
	SubmissionIp      string
	SubmissionUrl     string
	SubmissionEditUrl string
	LastUpdatedAt     time.Time
}

type Role string

const (
	Captain = "CAPTAIN"
	Player  = "PLAYER"
	Reserve = "RESERVE"
)

type Member struct {
	Id          int64
	TeamId      int64
	Name        string
	Surname     string
	GradYear    int32
	Role        Role
	PhoneNumber string
}

func (d *Database) ApplySubmission(ctx context.Context, team Team, submission Submission, members []Member) error {
	var teamId int64

	tx, err := d.Pool.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		} else {
			err = tx.Commit(ctx)
		}
	}()

	err = tx.QueryRow(
		ctx,
		`insert into team (name, fastcup_link, logo_link)
			values ($1, $2, $3)
			returning id`,
		team.Name,
		team.FastCupLink,
		team.LogoLink,
	).Scan(&teamId)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`insert into submission (external_id, team_id, tournament_id, submitted_at, submission_ip, submission_url, 
                        submission_edit_url, last_updated_at)
			values ($1, $2, $3, $4, $5, $6, $7, $8)`,
		submission.ExternalId,
		teamId,
		submission.TournamentId,
		submission.SubmittedAt,
		submission.SubmissionIp,
		submission.SubmissionUrl,
		submission.SubmissionEditUrl,
		submission.LastUpdatedAt,
	)
	if err != nil {
		return err
	}

	batch := &pgx.Batch{}

	for _, member := range members {
		batch.Queue(
			`INSERT INTO member (team_id, name, surname, grad_year, role, phone_number) 
             VALUES ($1, $2, $3, $4, $5, $6)`,
			teamId,
			member.Name,
			member.Surname,
			member.GradYear,
			member.Role,
			member.PhoneNumber,
		)
	}

	results := tx.SendBatch(ctx, batch)
	defer results.Close()

	// Process results to catch any errors
	for i := 0; i < len(members); i++ {
		_, err = results.Exec()
		if err != nil {
			return fmt.Errorf("failed to insert member %d: %w", i, err)
		}
	}

	return err
}

func (d *Database) ListRegisteredTeams(ctx context.Context) ([]Team, error) {
	var tournamentId int64

	err := d.Pool.QueryRow(ctx,
		`select id from tournament where is_active order by start_date limit 1`).Scan(&tournamentId)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (d *Database) SaveUserFromTelegram(ctx context.Context, user *tgbotapi.User) error {
	_, err := d.Pool.Exec(ctx,
		`INSERT INTO users (
            tg_id, username, first_name, last_name, language_code, is_bot,
            can_join_groups, can_read_all_group_messages,
            supports_inline_queries, phone
        ) VALUES (
            $1, $2, $3, $4, $5, $6,
            $7, $8,
            $9, $10
        )
        ON CONFLICT (tg_id) DO NOTHING`,
		user.ID,
		user.UserName,
		user.FirstName,
		user.LastName,
		user.LanguageCode,
		user.IsBot,
		user.CanJoinGroups,
		user.CanReadAllGroupMessages,
		user.SupportsInlineQueries,
		nil, // phone is nil by default unless you collect it via contact request
	)
	return err
}
