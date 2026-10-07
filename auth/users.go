package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/arfajhf/copytygo/v4/database"
	"github.com/arfajhf/copytygo/v4/database/drivers"
	"github.com/arfajhf/copytygo/v4/database/query"
	"github.com/arfajhf/copytygo/v4/security"
	"github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	errUserMissing = errors.New("User no longer exists.")
	errAdminOnly   = errors.New("Administrator access is required.")
	errOwnAdmin    = errors.New("You cannot delete your own account or remove your own administrator role.")
)

type userInput struct{ Name, Email, Role, Password string }
type userStats struct{ Total, Admins, Members int64 }
type userStore interface {
	Find(context.Context, string) (User, error)
	List(context.Context, string, int) ([]User, int64, error)
	Stats(context.Context) (userStats, error)
	Save(context.Context, string, string, userInput) error
	Delete(context.Context, string, string) error
}
type databaseUsers struct{ multi bool }

func usersDatabase() (*sql.DB, string, error) {
	drivers.Register()
	db, err := database.Connect()
	return db, database.LoadConfig().Driver, err
}

func (store databaseUsers) Find(ctx context.Context, id string) (User, error) {
	db, driver, err := usersDatabase()
	if err != nil {
		return User{}, err
	}
	columns := []string{"id", "name", "email"}
	if store.multi {
		columns = append(columns, "role")
	}
	row, found, err := query.Table(db, driver, "users").Select(columns...).WhereEq("id", id).FirstMap(ctx)
	if err != nil {
		return User{}, err
	}
	if !found {
		return User{}, errUserMissing
	}
	return userFromRow(row), nil
}
func userFromRow(row map[string]any) User {
	return User{ID: stringValue(row["id"]), Name: stringValue(row["name"]), Email: stringValue(row["email"]), Role: stringValue(row["role"])}
}
func (store databaseUsers) List(ctx context.Context, search string, page int) ([]User, int64, error) {
	db, driver, err := usersDatabase()
	if err != nil {
		return nil, 0, err
	}
	builder := query.Table(db, driver, "users").Select("id", "name", "email", "role").WhereAnyLike([]string{"name", "email"}, search)
	count, err := builder.Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	rows, err := builder.OrderBy("id", "DESC").Limit(10).Offset((page - 1) * 10).AllMaps(ctx)
	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFromRow(row))
	}
	return users, count, err
}
func (store databaseUsers) Stats(ctx context.Context) (userStats, error) {
	db, driver, err := usersDatabase()
	if err != nil {
		return userStats{}, err
	}
	total, err := query.Table(db, driver, "users").Count(ctx)
	if err != nil {
		return userStats{}, err
	}
	admins, err := query.Table(db, driver, "users").WhereEq("role", "admin").Count(ctx)
	return userStats{Total: total, Admins: admins, Members: total - admins}, err
}
func sqlMark(driver string, n int) string {
	if driver == "postgres" {
		return "$" + strconv.Itoa(n)
	}
	return "?"
}

// Lock administrators in one order so concurrent changes cannot remove access
// between the permission check and a write. Own-account demotion/deletion is denied.
func lockAdmin(ctx context.Context, tx *sql.Tx, actor string) error {
	rows, err := tx.QueryContext(ctx, "SELECT id FROM users WHERE role = 'admin' ORDER BY id FOR UPDATE")
	if err != nil {
		return err
	}
	allowed := false
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		allowed = allowed || id == actor
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if !allowed {
		return errAdminOnly
	}
	return nil
}
func duplicateEmail(err error) error {
	var my *mysql.MySQLError
	var pg *pgconn.PgError
	if errors.As(err, &my) && my.Number == 1062 || errors.As(err, &pg) && pg.Code == "23505" {
		return ErrEmailRegistered
	}
	return err
}
func (store databaseUsers) Save(ctx context.Context, actor, id string, input userInput) error {
	if !store.multi {
		return errAdminOnly
	}
	input.Name = strings.TrimSpace(input.Name)
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if input.Name == "" || input.Email == "" || (input.Role != "user" && input.Role != "admin") {
		return ErrInvalidRegistration
	}
	if id == actor && input.Role != "admin" {
		return errOwnAdmin
	}
	hashed := ""
	var err error
	if input.Password != "" {
		hashed, err = security.HashPassword(input.Password)
		if err != nil {
			return err
		}
	}
	if id == "" && hashed == "" {
		return ErrInvalidRegistration
	}
	db, driver, err := usersDatabase()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockAdmin(ctx, tx, actor); err != nil {
		return err
	}
	if id == "" {
		_, err = tx.ExecContext(ctx, "INSERT INTO users (name,email,role,password) VALUES ("+sqlMark(driver, 1)+","+sqlMark(driver, 2)+","+sqlMark(driver, 3)+","+sqlMark(driver, 4)+")", input.Name, input.Email, input.Role, hashed)
	} else {
		var exists string
		err = tx.QueryRowContext(ctx, "SELECT id FROM users WHERE id = "+sqlMark(driver, 1)+" FOR UPDATE", id).Scan(&exists)
		if errors.Is(err, sql.ErrNoRows) {
			return errUserMissing
		}
		if err != nil {
			return err
		}
		args := []any{input.Name, input.Email, input.Role}
		statement := "UPDATE users SET name=" + sqlMark(driver, 1) + ",email=" + sqlMark(driver, 2) + ",role=" + sqlMark(driver, 3)
		if hashed != "" {
			args = append(args, hashed)
			statement += ",password=" + sqlMark(driver, 4)
		}
		args = append(args, id)
		statement += " WHERE id=" + sqlMark(driver, len(args))
		_, err = tx.ExecContext(ctx, statement, args...)
	}
	if err != nil {
		return duplicateEmail(err)
	}
	return tx.Commit()
}
func (store databaseUsers) Delete(ctx context.Context, actor, id string) error {
	if !store.multi {
		return errAdminOnly
	}
	if actor == id {
		return errOwnAdmin
	}
	db, driver, err := usersDatabase()
	if err != nil {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := lockAdmin(ctx, tx, actor); err != nil {
		return err
	}
	result, err := tx.ExecContext(ctx, "DELETE FROM users WHERE id="+sqlMark(driver, 1), id)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if count == 0 {
		return errUserMissing
	}
	return tx.Commit()
}

// PromoteAdmin grants the first administrator through a trusted local CLI.
// Register the account normally first. Public forms never grant this role.
func PromoteAdmin(ctx context.Context, email string) error {
	db, driver, err := usersDatabase()
	if err != nil {
		return err
	}
	email = strings.ToLower(strings.TrimSpace(email))
	row, found, err := query.Table(db, driver, "users").Select("id", "role").WhereEq("email", email).FirstMap(ctx)
	if err != nil {
		return fmt.Errorf("multi-role users table is required: %w", err)
	}
	if !found {
		return errUserMissing
	}
	_, err = query.Table(db, driver, "users").WhereEq("id", row["id"]).Update(ctx, map[string]any{"role": "admin"})
	return err
}
