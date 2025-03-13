package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"

	"github.com/Dpalme/posterify-backend/app"
	"github.com/jmoiron/sqlx"
)

type UserService struct {
	db *DB
}

func NewUserService(db *DB) *UserService {
	return &UserService{db}
}

func (us *UserService) CreateUser(ctx context.Context, user *app.User) error {
	tx, err := us.db.BeginTxx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := createUser(ctx, tx, user); err != nil {
		return err
	}

	return tx.Commit()
}

func (us *UserService) UserByID(ctx context.Context, id uint) (*app.User, error) {
	tx, err := us.db.BeginTxx(ctx, nil)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	user, err := findUserByID(ctx, tx, id)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) UserByEmail(ctx context.Context, email string) (*app.User, error) {
	tx, err := us.db.BeginTxx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	user, err := findOneUser(ctx, tx, app.UserFilter{Email: &email})

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return user, nil
}

func (us *UserService) Users(ctx context.Context, uf app.UserFilter) ([]*app.User, error) {
	tx, err := us.db.BeginTxx(ctx, nil)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	users, err := findUsers(ctx, tx, uf)

	if err != nil {
		return nil, err
	}

	return users, tx.Commit()
}

func (us *UserService) Authenticate(ctx context.Context, email string, password string) (*app.User, error) {
	user, err := us.UserByEmail(ctx, email)

	if err != nil {
		return nil, err
	}

	if !user.VerifyPassword(password) {
		return nil, app.ErrUnAuthorized
	}

	return user, nil
}

func (us *UserService) UpdateUser(ctx context.Context, user *app.User, patch app.UserPatch) error {
	tx, err := us.db.BeginTxx(ctx, nil)

	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	defer tx.Rollback()

	if err := updateUser(ctx, tx, user, patch); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	return nil
}

func (us *UserService) DeleteUser(ctx context.Context, id uint) error {
	tx, err := us.db.BeginTxx(ctx, nil)

	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	defer tx.Rollback()

	query := `
	DELETE
	FROM users 
	WHERE id = $1`

	if _, err := tx.ExecContext(ctx, query, id); err != nil {
		log.Printf("error deleting record: %v", err)
		return app.ErrInternal
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	return nil
}

func createUser(ctx context.Context, tx *sqlx.Tx, user *app.User) error {
	query := `
	INSERT INTO users (email, password_hash)
	VALUES ($1, $2) RETURNING id, created_at, updated_at
	`
	args := []interface{}{user.Email, user.PasswordHash}
	err := tx.QueryRowxContext(ctx, query, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "users_email_key"`:
			return app.ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func findUserByID(ctx context.Context, tx *sqlx.Tx, id uint) (*app.User, error) {
	return findOneUser(ctx, tx, app.UserFilter{ID: &id})
}

func findOneUser(ctx context.Context, tx *sqlx.Tx, filter app.UserFilter) (*app.User, error) {
	users, err := findUsers(ctx, tx, filter)

	if err != nil {
		return nil, err
	} else if len(users) == 0 {
		return nil, app.ErrNotFound
	}

	return users[0], nil
}

func findUsers(ctx context.Context, tx *sqlx.Tx, filter app.UserFilter) ([]*app.User, error) {
	where, args := []string{}, []any{}
	argPosition := 0

	if v := filter.ID; v != nil {
		argPosition++
		where, args = append(where, fmt.Sprintf("id = $%d", argPosition)), append(args, *v)
	}

	if v := filter.Email; v != nil {
		argPosition++
		where, args = append(where, fmt.Sprintf("email = $%d", argPosition)), append(args, *v)
	}

	query := "SELECT * from users" + formatWhereClause(where) +
		" ORDER BY id ASC" + formatLimitOffset(filter.Limit, filter.Offset)
	users, err := queryUsers(ctx, tx, query, args...)

	if err != nil {
		return nil, err
	}

	return users, nil
}

func updateUser(ctx context.Context, tx *sqlx.Tx, user *app.User, patch app.UserPatch) error {
	if v := patch.Email; v != nil {
		user.Email = *v
	}

	if v := patch.PasswordHash; v != nil {
		user.PasswordHash = *v
	}

	args := []interface{}{
		user.Email,
		user.PasswordHash,
		user.ID,
	}

	query := `
	UPDATE users 
	SET email = $1, password_hash = $2, updated_at = NOW()
	WHERE id = $3
	RETURNING updated_at`

	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&user.UpdatedAt); err != nil {
		log.Printf("error updating record: %v", err)
		return app.ErrInternal
	}

	return nil
}

func queryUsers(ctx context.Context, tx *sqlx.Tx, query string, args ...interface{}) ([]*app.User, error) {
	users := []*app.User{}
	err := tx.SelectContext(ctx, &users, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("User not found.")
			tx.Rollback() // Rollback immediately
			return nil, errors.New("user not found")
		}
		fmt.Println("Error in the users query from tx.SelectContext: ", err)
		return nil, err
	}

	return users, nil
}
