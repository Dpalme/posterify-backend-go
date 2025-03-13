package postgres

import (
	"context"
	"fmt"
	"log"

	"github.com/Dpalme/posterify-backend/app"
	"github.com/jmoiron/sqlx"
)

type CollectionService struct {
	db *DB
}

func NewCollectionService(db *DB) *CollectionService {
	return &CollectionService{db}
}

func (cs *CollectionService) CreateCollection(ctx context.Context, collection *app.Collection) error {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		return err
	}

	defer tx.Rollback()

	if err := createCollection(ctx, tx, collection); err != nil {
		return err
	}

	return tx.Commit()
}

func (cs *CollectionService) CollectionByID(ctx context.Context, id int) (*app.Collection, error) {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		fmt.Println("could not begin transaction: ", err)
		return nil, err
	}

	defer tx.Rollback()

	collection, err := findCollectionByID(ctx, tx, id)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return collection, nil
}

func (cs *CollectionService) Collections(ctx context.Context, cf app.CollectionFilter) ([]*app.Collection, error) {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		return nil, err
	}

	defer tx.Rollback()

	collections, err := findCollections(ctx, tx, cf)

	if err != nil {
		return nil, err
	}

	return collections, tx.Commit()
}

func (cs *CollectionService) UpdateCollection(ctx context.Context, collection *app.Collection, patch app.CollectionPatch) error {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	defer tx.Rollback()

	if err := updateCollection(ctx, tx, collection, patch); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	return nil
}

func (cs *CollectionService) DeleteCollection(ctx context.Context, id int) error {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	defer tx.Rollback()

	_, err = cs.CollectionByID(ctx, id)
	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := deleteCollection(ctx, tx, &id); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	return nil
}

func (cs *CollectionService) SaveImageToCollection(ctx context.Context, c_id int, image string) error {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	defer tx.Rollback()

	collection, err := cs.CollectionByID(ctx, c_id)
	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := saveToCollection(ctx, tx, collection, image); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	return nil
}

func (cs *CollectionService) DeleteImageFromCollection(ctx context.Context, c_id int, imagePath string) error {
	tx, err := cs.db.BeginTxx(ctx, nil)

	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	defer tx.Rollback()

	collection, err := cs.CollectionByID(ctx, c_id)
	if err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := removeFromCollection(ctx, tx, collection, &imagePath); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	if err := tx.Commit(); err != nil {
		log.Println(err)
		return app.ErrInternal
	}

	return nil
}

func markCollectionUpdate(ctx context.Context, tx *sqlx.Tx, collection *app.Collection) error {
	query := `
	UPDATE collections
	SET updated_at = NOW()
	WHERE id = $1
	RETURNING updated_at`
	err := tx.QueryRowxContext(ctx, query, collection.ID).Scan(&collection.UpdatedAt)
	if err != nil {
		return err
	}
	return nil
}

func createCollection(ctx context.Context, tx *sqlx.Tx, collection *app.Collection) error {
	query := `
	INSERT INTO collections (name, description, poster, author_id)
	VALUES ($1, $2, $3, $4) RETURNING id, created_at, updated_at, author_id
	`
	args := []interface{}{collection.Name, collection.Description, collection.Poster, collection.AuthorID}
	err := tx.QueryRowxContext(ctx, query, args...).Scan(&collection.ID, &collection.CreatedAt, &collection.UpdatedAt, &collection.AuthorID)

	if err != nil {
		switch {
		case err.Error() == `pq: duplicate key value violates unique constraint "collection_name_user_key"`:
			return app.ErrDuplicateEmail
		default:
			return err
		}
	}

	return nil
}

func findCollectionByID(ctx context.Context, tx *sqlx.Tx, id int) (*app.Collection, error) {
	return findOneCollection(ctx, tx, app.CollectionFilter{ID: &id})
}

func findOneCollection(ctx context.Context, tx *sqlx.Tx, filter app.CollectionFilter) (*app.Collection, error) {
	cs, err := findCollections(ctx, tx, filter)

	if err != nil {
		fmt.Println("Error from findOneCollection: ", err)
		return nil, err
	} else if len(cs) == 0 {
		return nil, app.ErrNotFound
	}

	return cs[0], nil
}

func findCollections(ctx context.Context, tx *sqlx.Tx, filter app.CollectionFilter) ([]*app.Collection, error) {
	where, args := []string{}, []interface{}{}
	argPosition := 0

	if v := filter.ID; v != nil {
		argPosition++
		where, args = append(where, fmt.Sprintf("id = $%d", argPosition)), append(args, *v)
	}

	if v := filter.Name; v != nil {
		argPosition++
		where, args = append(where, fmt.Sprintf("name = $%d", argPosition)), append(args, *v)
	}

	query := "SELECT * from collections" + formatWhereClause(where) +
		" ORDER BY id ASC" + formatLimitOffset(filter.Limit, filter.Offset)
	collections, err := queryCollections(ctx, tx, query, args...)

	if err != nil {
		fmt.Println("Error from findCollections: ", err)
		return nil, err
	}

	return collections, nil
}

func updateCollection(ctx context.Context, tx *sqlx.Tx, collection *app.Collection, patch app.CollectionPatch) error {
	if v := patch.Name; v != nil {
		collection.Name = *v
	}

	if v := patch.Description; v != nil {
		collection.Description = *v
	}

	if v := patch.Poster; v != nil {
		collection.Poster = *v
	}

	args := []interface{}{
		collection.Name,
		collection.Description,
		collection.Poster,
		collection.ID,
	}

	query := `
	UPDATE collections 
	SET name = $1, description = $2, poster = $3, updated_at = NOW()
	WHERE id = $4
	RETURNING updated_at`

	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&collection.UpdatedAt); err != nil {
		log.Printf("error updating record: %v", err)
		return app.ErrInternal
	}

	return nil
}

func queryCollections(ctx context.Context, tx *sqlx.Tx, query string, args ...interface{}) ([]*app.Collection, error) {
	collections := []*app.Collection{}
	err := tx.SelectContext(ctx, &collections, query, args...)
	if err != nil {
		fmt.Println("Error from tx.SelectContext: ", err)
		return nil, err
	}

	return collections, nil
}

func saveToCollection(ctx context.Context, tx *sqlx.Tx, collection *app.Collection, imgPath string) error {
	args := []interface{}{
		imgPath,
		collection.ID,
	}

	query := `
	INSERT INTO collections_images (img_path, collection_id, created_at)
	VALUES ($1, $2, NOW())
	RETURNING created_at`

	if err := tx.QueryRowxContext(ctx, query, args...).Scan(&collection.UpdatedAt); err != nil {
		log.Printf("error creating record: %v", err)
		return app.ErrInternal
	}

	markCollectionUpdate(ctx, tx, collection)

	return nil
}

func removeFromCollection(ctx context.Context, tx *sqlx.Tx, collection *app.Collection, imgPath *string) error {
	args := []interface{}{
		imgPath,
		collection.ID,
	}

	query := `
	DELETE
	FROM collections_images
	WHERE img_path = $1 AND collection_id = $2`

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		log.Printf("error deleting record: %v", err)
		return app.ErrInternal
	}

	markCollectionUpdate(ctx, tx, collection)

	return nil
}

func deleteCollection(ctx context.Context, tx *sqlx.Tx, collection_id *int) error {
	args := []interface{}{
		collection_id,
	}

	query := `
	DELETE
	FROM collections
	WHERE id = $1`

	if _, err := tx.ExecContext(ctx, query, args...); err != nil {
		log.Printf("error deleting record: %v", err)
		return app.ErrInternal
	}

	return nil
}
