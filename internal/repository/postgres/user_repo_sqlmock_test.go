package postgres

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"2026_2_Na_Vse_200/internal/models"
)

const insertUser = `
	INSERT INTO account (email, password_hash, display_name, avatar_file_id)
	VALUES ($1, $2, $3, $4)
	RETURNING id, created_at`

const findUser = `
	SELECT id, email, password_hash, display_name, avatar_file_id::text, created_at
	FROM account
	WHERE email = $1`

const updateUser = `
	UPDATE account SET display_name = $3
	WHERE id = $1::uuid AND email = $2
	RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at`

const setAvatar = `
	UPDATE account SET avatar_file_id = $2::uuid
	WHERE id = $1::uuid
	RETURNING id::text, email, password_hash, display_name, avatar_file_id::text, created_at`

var userCols = []string{"id", "email", "password_hash", "display_name", "avatar_file_id", "created_at"}

type sqlDB struct {
	db *sql.DB
}

// QueryRow передаёт запрос репозитория в sqlmock через database/sql.
func (d sqlDB) QueryRow(ctx context.Context, query string, args ...any) pgx.Row {
	return sqlRow{row: d.db.QueryRowContext(ctx, query, args...)}
}

type sqlRow struct {
	row *sql.Row
}

// Scan возвращает ошибку отсутствия строки в том же виде, что и pgx.
func (r sqlRow) Scan(dst ...any) error {
	err := r.row.Scan(dst...)
	if errors.Is(err, sql.ErrNoRows) {
		return pgx.ErrNoRows
	}
	return err
}

// newUserMock создаёт репозиторий с sqlmock и проверяет ожидания после теста.
func newUserMock(t *testing.T) (*UserRepo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("создание sqlmock: %v", err)
	}
	t.Cleanup(func() {
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Error(err)
		}
		_ = db.Close()
	})
	return &UserRepo{pool: sqlDB{db: db}}, mock
}

// TestUserCreate проверяет аргументы INSERT и поля, которые возвращает база.
func TestUserCreate(t *testing.T) {
	repo, mock := newUserMock(t)
	user := models.User{
		Email:        "andrey@example.com",
		PasswordHash: "stored-hash",
		DisplayName:  "Андрей",
	}
	created := time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC)
	id := "11111111-1111-4111-8111-111111111111"
	mock.ExpectQuery(insertUser).
		WithArgs(user.Email, user.PasswordHash, user.DisplayName, nil).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at"}).AddRow(id, created))

	got, err := repo.Create(context.Background(), user)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	user.ID = models.UserID(id)
	user.CreatedAt = created
	if !reflect.DeepEqual(got, user) {
		t.Errorf("пользователь = %+v, ожидался %+v", got, user)
	}
}

// TestUserCreateErrors проверяет конфликт email и передачу остальных ошибок БД.
func TestUserCreateErrors(t *testing.T) {
	dbErr := errors.New("соединение потеряно")
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "email занят",
			err:  &pgconn.PgError{Code: "23505", ConstraintName: "account_email_uk"},
			want: models.ErrEmailTaken,
		},
		{name: "ошибка БД", err: dbErr, want: dbErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newUserMock(t)
			user := models.User{Email: "andrey@example.com", PasswordHash: "stored-hash", DisplayName: "Андрей"}
			mock.ExpectQuery(insertUser).
				WithArgs(user.Email, user.PasswordHash, user.DisplayName, nil).
				WillReturnError(tt.err)

			_, err := repo.Create(context.Background(), user)
			if !errors.Is(err, tt.want) {
				t.Errorf("ошибка = %v, ожидалась %v", err, tt.want)
			}
		})
	}
}

// TestUserGetByEmail проверяет чтение пользователя, отсутствие строки и ошибку БД.
func TestUserGetByEmail(t *testing.T) {
	avatar := models.MediaFileID("22222222-2222-4222-8222-222222222222")
	user := models.User{
		ID:           "11111111-1111-4111-8111-111111111111",
		Email:        "andrey@example.com",
		PasswordHash: "stored-hash",
		DisplayName:  "Андрей",
		AvatarFileID: &avatar,
		AvatarURL:    "/api/v1/media/" + string(avatar),
		CreatedAt:    time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC),
	}
	dbErr := errors.New("соединение потеряно")
	tests := []struct {
		name    string
		rows    *sqlmock.Rows
		err     error
		want    models.User
		wantErr error
	}{
		{
			name: "найден",
			rows: sqlmock.NewRows(userCols).AddRow(string(user.ID), user.Email,
				user.PasswordHash, user.DisplayName, string(avatar), user.CreatedAt),
			want: user,
		},
		{name: "не найден", rows: sqlmock.NewRows(userCols), wantErr: models.ErrUserNotFound},
		{name: "ошибка БД", err: dbErr, wantErr: dbErr},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo, mock := newUserMock(t)
			query := mock.ExpectQuery(findUser).WithArgs(user.Email)
			if tt.err != nil {
				query.WillReturnError(tt.err)
			} else {
				query.WillReturnRows(tt.rows)
			}

			got, err := repo.GetByEmail(context.Background(), user.Email)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ошибка = %v, ожидалась %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("пользователь = %+v, ожидался %+v", got, tt.want)
			}
		})
	}
}

// TestUserUpdateProfile проверяет изменение профиля с сохранением аватара.
func TestUserUpdateProfile(t *testing.T) {
	repo, mock := newUserMock(t)
	avatar := models.MediaFileID("22222222-2222-4222-8222-222222222222")
	want := models.User{
		ID:           "11111111-1111-4111-8111-111111111111",
		Email:        "andrey@example.com",
		PasswordHash: "stored-hash",
		DisplayName:  "Андрей",
		AvatarFileID: &avatar,
		AvatarURL:    "/api/v1/media/" + string(avatar),
		CreatedAt:    time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC),
	}
	mock.ExpectQuery(updateUser).
		WithArgs(string(want.ID), want.Email, want.DisplayName).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(string(want.ID), want.Email,
			want.PasswordHash, want.DisplayName, string(avatar), want.CreatedAt))

	got, err := repo.UpdateProfile(context.Background(), want.ID, want.Email, want.DisplayName)
	if err != nil {
		t.Fatalf("UpdateProfile: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("профиль = %+v, ожидался %+v", got, want)
	}
}

// TestUserSetAvatar проверяет привязку файла к пользователю.
func TestUserSetAvatar(t *testing.T) {
	repo, mock := newUserMock(t)
	avatar := models.MediaFileID("22222222-2222-4222-8222-222222222222")
	want := models.User{
		ID:           "11111111-1111-4111-8111-111111111111",
		Email:        "andrey@example.com",
		PasswordHash: "stored-hash",
		DisplayName:  "Андрей",
		AvatarFileID: &avatar,
		AvatarURL:    "/api/v1/media/" + string(avatar),
		CreatedAt:    time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC),
	}
	mock.ExpectQuery(setAvatar).
		WithArgs(string(want.ID), string(avatar)).
		WillReturnRows(sqlmock.NewRows(userCols).AddRow(string(want.ID), want.Email,
			want.PasswordHash, want.DisplayName, string(avatar), want.CreatedAt))

	got, err := repo.SetAvatar(context.Background(), want.ID, avatar)
	if err != nil {
		t.Fatalf("SetAvatar: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("профиль = %+v, ожидался %+v", got, want)
	}
}
