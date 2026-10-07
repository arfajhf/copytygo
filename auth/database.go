package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/arfajhf/copytygo/v3/database"
	"github.com/arfajhf/copytygo/v3/database/drivers"
	"github.com/arfajhf/copytygo/v3/database/query"
	"github.com/arfajhf/copytygo/v3/security"
)

var (
	ErrInvalidCredentials  = errors.New("copytygo: invalid credentials")
	ErrEmailRegistered     = errors.New("copytygo: email already registered")
	ErrInvalidRegistration = errors.New("copytygo: name and email are required")
)

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

type Session struct {
	User  User   `json:"user"`
	Token string `json:"token"`
}

func Register(name, email, password, role string) (Session, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(strings.TrimSpace(email))
	role = strings.TrimSpace(role)

	if name == "" || email == "" {
		return Session{}, ErrInvalidRegistration
	}

	drivers.Register()
	db, err := database.Connect()
	if err != nil {
		return Session{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	existing, found, err := query.Table(db, database.LoadConfig().Driver, "users").
		WhereEq("email", email).
		FirstMap(ctx)
	if err != nil {
		return Session{}, err
	}
	if found && existing != nil {
		return Session{}, ErrEmailRegistered
	}

	hashed, err := security.HashPassword(password)
	if err != nil {
		return Session{}, err
	}

	values := map[string]any{
		"name":     name,
		"email":    email,
		"password": hashed,
	}
	if role != "" {
		values["role"] = role
	}

	id, err := query.Table(db, database.LoadConfig().Driver, "users").InsertID(ctx, values)
	if err != nil {
		return Session{}, err
	}

	user := User{
		ID:    strconv.FormatInt(id, 10),
		Name:  name,
		Email: email,
		Role:  role,
	}

	token, err := Issue(user.ID, user.Role, map[string]string{
		"name":  user.Name,
		"email": user.Email,
	}, 24*time.Hour)
	if err != nil {
		return Session{}, err
	}

	return Session{User: user, Token: token}, nil
}

func Login(email, password string) (Session, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	drivers.Register()
	db, err := database.Connect()
	if err != nil {
		return Session{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row, found, err := query.Table(db, database.LoadConfig().Driver, "users").
		WhereEq("email", email).
		FirstMap(ctx)
	if err != nil {
		return Session{}, err
	}
	if !found {
		return Session{}, ErrInvalidCredentials
	}

	hashed := stringValue(row["password"])
	if hashed == "" || !security.VerifyPassword(password, hashed) {
		return Session{}, ErrInvalidCredentials
	}

	user := User{
		ID:    stringValue(row["id"]),
		Name:  stringValue(row["name"]),
		Email: stringValue(row["email"]),
		Role:  stringValue(row["role"]),
	}

	token, err := Issue(user.ID, user.Role, map[string]string{
		"name":  user.Name,
		"email": user.Email,
	}, 24*time.Hour)
	if err != nil {
		return Session{}, err
	}

	return Session{User: user, Token: token}, nil
}

func stringValue(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	case nil:
		return ""
	default:
		return fmt.Sprint(typed)
	}
}
