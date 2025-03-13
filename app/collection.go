package app

import (
	"context"
	"slices"
	"time"
)

type Collection struct {
	ID          int       `json:"id" db:"id"`
	AuthorID    int       `json:"author" db:"author_id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	Poster      string    `json:"poster,omitempty" db:"poster"`
	Images      []*Image  `json:"images,omitempty"`
	CreatedAt   time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt   time.Time `json:"updatedAt" db:"updated_at"`
}

type Image struct {
	Path         string    `json:"image" db:"img_path"`
	CollectionId int       `json:"collectionId" db:"collection_id"`
	SavedAt      time.Time `json:"savedAt" db:"created_at"`
}

func (c *Collection) Collection() *Collection {
	return &Collection{
		ID:       c.ID,
		AuthorID: c.AuthorID,
		Name:     c.Name,
		Poster:   c.Poster,
	}
}

func (c *Collection) CollectionWithImages() *Collection {
	return &Collection{
		ID:       c.ID,
		AuthorID: c.AuthorID,
		Name:     c.Name,
		Poster:   c.Poster,
		Images:   c.Images,
	}
}

func isSaved(collection *Collection, imgPath *string) bool {
	if len(collection.Images) == 0 {
		collection = collection.CollectionWithImages()
	}
	for _, i := range collection.Images {
		if imgPath == &i.Path {
			return true
		}
	}
	return false
}

type CollectionFilter struct {
	ID       *int
	Name     *string
	AuthorId *int

	Limit  int
	Offset int
}

type CollectionPatch struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Poster      *string `json:"poster"`
}

func (c *Collection) SaveImageToCollection(imgPath *string) (*Collection, error) {
	if !isSaved(c, imgPath) {
		c.Images = append(c.Images, &Image{
			Path:         *imgPath,
			CollectionId: c.ID,
		})
		return c, nil
	}

	return nil, ErrImageAlreadySaved
}

func (c *Collection) DeleteImageFromCollection(imgPath string) (*Collection, error) {
	for i, image := range c.Images {
		if imgPath == image.Path {
			c.Images = slices.Delete(c.Images, i, i)
			return c, nil
		}
	}

	return nil, ErrImageNotSaved
}

type CollectionService interface {
	CreateCollection(context.Context, *Collection) error

	CollectionByID(context.Context, int) (*Collection, error)

	Collections(context.Context, CollectionFilter) ([]*Collection, error)

	UpdateCollection(context.Context, *Collection, CollectionPatch) error

	DeleteCollection(context.Context, int) error

	SaveImageToCollection(context.Context, int, string) error

	DeleteImageFromCollection(context.Context, int, string) error
}
