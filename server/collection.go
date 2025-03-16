package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Dpalme/posterify-backend/app"
	"github.com/gorilla/mux"
)

func collectionBelongsToUser(r *http.Request, collection *app.Collection) bool {
	user := userFromContext(r.Context())
	if user.IsAnonymous() {
		return false
	}
	return user.ID == collection.AuthorID
}

func (s *Server) createCollection() http.HandlerFunc {
	type Input struct {
		Name        string `json:"name" validate:"required,min=3,max=48"`
		Description string `json:"description" validate:"required,min=0,max=96"`
		Poster      string `json:"poster,omitempty"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		input := &Input{}

		shouldReturn := parseInput(r, input, w)
		if shouldReturn {
			return
		}

		user := userFromContext(r.Context())

		collection := app.Collection{
			Name:        input.Name,
			Description: input.Description,
			Poster:      input.Poster,
			AuthorID:    user.ID,
		}

		if err := s.collectionService.CreateCollection(r.Context(), &collection); err != nil {
			serverError(w, err)
			return
		}

		writeJSON(w, http.StatusCreated, M{"collection": collection})
	}
}

func (s *Server) updateCollection() http.HandlerFunc {
	type Input struct {
		ID          *int    `json:"id,omitempty"`
		Name        *string `json:"name,omitempty"`
		Description *string `json:"description,omitempty"`
		Poster      *string `json:"poster,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		input := &Input{}
		vars := mux.Vars(r)

		shouldReturn := parseInput(r, input, w)
		if shouldReturn {
			return
		}

		n, err := strconv.ParseInt(vars["id"], 0, 0)
		if err != nil {
			err := ErrorM{"collection": []string{"authorId is not valid"}}
			validationError(w, err)
			return
		}
		nInt := int(n)

		input.ID = &nInt

		if err := validate.Struct(input); err != nil {
			validationError(w, err)
			return
		}

		ctx := r.Context()
		collection, err := s.collectionService.CollectionByID(ctx, *input.ID)
		if err != nil {
			appErr := ErrorM{"collection": []string{"no collection with given id found"}}
			notFoundError(w, appErr)
			return
		}

		if !collectionBelongsToUser(r, collection) {
			unauthorizedForActionError(w)
			return
		}

		patch := app.CollectionPatch{
			Name:        input.Name,
			Description: input.Description,
			Poster:      input.Poster,
		}

		err = s.collectionService.UpdateCollection(ctx, collection, patch)
		if err != nil {
			serverError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, M{"collection": collection})
	}
}

func (s *Server) getCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		c_id := vars["id"]
		ctx := r.Context()

		n, err := strconv.ParseInt(c_id, 0, 0)
		if err != nil {
			err := ErrorM{"collection": []string{"authorId is not valid"}}
			validationError(w, err)
			return
		}
		nInt := int(n)

		collection, err := s.collectionService.CollectionByID(ctx, nInt)

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}

			return
		}

		if !collectionBelongsToUser(r, collection) {
			unauthorizedForActionError(w)
			return
		}

		writeJSON(w, http.StatusOK, M{"collection": collection})
	}
}

func (s *Server) deleteCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		c_id := vars["id"]

		ctx := r.Context()
		n, err := strconv.ParseInt(c_id, 0, 0)
		if err != nil {
			err := ErrorM{"collection": []string{"authorId is not valid"}}
			validationError(w, err)
			return
		}
		nInt := int(n)
		collection, err := s.collectionService.CollectionByID(ctx, nInt)

		if err != nil {
			if errors.Is(err, app.ErrNotFound) {
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)

			} else {
				serverError(w, err)
			}

			return
		}

		if !collectionBelongsToUser(r, collection) {
			unauthorizedForActionError(w)
			return
		}

		err = s.collectionService.DeleteCollection(ctx, nInt)
		if err != nil {
			err := ErrorM{"collection": []string{"could not delete collection"}}
			serverError(w, err)
			return
		}

		writeJSON(w, http.StatusOK, M{"collection": collection})
	}
}

func (s *Server) listCollections() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		ctx := r.Context()
		filter := app.CollectionFilter{}

		if v := query.Get("name"); v != "" {
			filter.Name = &v
		}

		user := userFromContext(ctx)
		filter.AuthorId = &user.ID

		if v := query.Get("id"); v != "" {
			n, err := strconv.ParseInt(v, 0, 0)
			if err != nil {
				err := ErrorM{"collection": []string{"id is not valid"}}
				validationError(w, err)
				return
			}
			nInt := int(n)
			filter.ID = &nInt
		}

		if v := query.Get("limit"); v != "" {
			limit, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				err := ErrorM{"collection": []string{"limit is not valid"}}
				validationError(w, err)
				return
			}
			uint_limit := int(limit)
			filter.Limit = uint_limit
		}
		if v := query.Get("offset"); v != "" {
			offset, err := strconv.ParseUint(v, 10, 32)
			if err != nil {
				err := ErrorM{"collection": []string{"offset is not valid"}}
				validationError(w, err)
				return
			}
			uint_offset := int(offset)
			filter.Offset = uint_offset
		}

		collection, err := s.collectionService.Collections(ctx, filter)

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}

			return
		}

		writeJSON(w, http.StatusOK, M{"collections": collection, "offset": filter.Offset, "limit": filter.Limit})
	}
}

func (s *Server) saveImageToCollection() http.HandlerFunc {
	type Input struct {
		ImagePath *string `json:"imgPath"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ctx := r.Context()
		input := &Input{}

		shouldReturn := parseInput(r, input, w)
		if shouldReturn {
			return
		}

		if input.ImagePath == nil {
			err := ErrorM{"collection": []string{"imgPath is not valid"}}
			validationError(w, err)
			return
		}

		c_id := vars["id"]
		n, err := strconv.ParseInt(c_id, 0, 0)
		if err != nil {
			err := ErrorM{"collection": []string{"authorId is not valid"}}
			validationError(w, err)
			return
		}
		nInt := int(n)
		collection, err := s.collectionService.CollectionByID(ctx, nInt)

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}
			return
		}

		if !collectionBelongsToUser(r, collection) {
			unauthorizedForActionError(w)
			return
		}

		fmt.Printf("%+v\n", input)

		err = s.collectionService.SaveImageToCollection(ctx, nInt, *input.ImagePath)

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}
			return
		}

		collection, err = collection.SaveImageToCollection(input.ImagePath)

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}
			return
		}

		writeJSON(w, http.StatusOK, M{"collection": collection})
	}
}

func (s *Server) deleteImageFromCollection() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ctx := r.Context()
		imagePath := vars["imagePath"]

		c_id := vars["id"]
		n, err := strconv.ParseInt(c_id, 0, 0)
		if err != nil {
			err := ErrorM{"collection": []string{"authorId is not valid"}}
			validationError(w, err)
			return
		}
		nInt := int(n)
		collection, err := s.collectionService.CollectionByID(ctx, nInt)

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"collection": []string{"collection not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}
			return
		}

		if !collectionBelongsToUser(r, collection) {
			unauthorizedForActionError(w)
			return
		}

		s.collectionService.DeleteImageFromCollection(ctx, collection.ID, imagePath)

		writeJSON(w, http.StatusOK, M{"collection": collection})
	}
}
