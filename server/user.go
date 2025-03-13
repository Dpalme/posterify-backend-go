package server

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/Dpalme/posterify-backend/app"
	"github.com/go-playground/validator/v10"
	"github.com/gorilla/mux"
)

var validate *validator.Validate

func init() {
	validate = validator.New()
	validate.RegisterTagNameFunc(func(fid reflect.StructField) string {
		name := strings.SplitN(fid.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			name = ""
		}
		return name
	})
}

func (s *Server) createUser() http.HandlerFunc {
	type Input struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required,min=8,max=72"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		input := &Input{}

		if err := readJSON(r.Body, &input); err != nil {
			errorResponse(w, http.StatusUnprocessableEntity, err)
			return
		}

		if err := validate.Struct(input); err != nil {
			validationError(w, err)
			return
		}

		user := app.User{
			Email: input.Email,
		}

		user.SetPassword(input.Password)

		if err := s.userService.CreateUser(r.Context(), &user); err != nil {
			switch {
			case errors.Is(err, app.ErrDuplicateEmail):
				err = ErrorM{"email": []string{"this email is already in use"}}
				errorResponse(w, http.StatusConflict, err)
			default:
				serverError(w, err)
			}
			return
		}

		writeJSON(w, http.StatusCreated, M{"user": user})
	}
}

func (s *Server) loginUser() http.HandlerFunc {
	type Input struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		input := Input{}
		if err := readJSON(r.Body, &input); err != nil {
			errorResponse(w, http.StatusUnprocessableEntity, err)
			return
		}
		log.Printf("Authenticating user %v %v", input.Email, input.Password)

		user, err := s.userService.Authenticate(r.Context(), input.Email, input.Password)
		if err != nil || user == nil {
			invalidUserCredentialsError(w)
			return
		}

		token, err := generateUserToken(user)

		if err != nil {
			serverError(w, err)
			return
		}

		user.Token = token

		writeJSON(w, http.StatusOK, M{"user": user})

	}
}

func (s *Server) getCurrentUser() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		user := userFromContext(ctx)
		user.Token = userTokenFromContext(ctx)

		writeJSON(w, http.StatusOK, M{"user": user})
	}
}

func (s *Server) updateUser() http.HandlerFunc {
	type Input struct {
		Email    *string `json:"email,omitempty"`
		Password *string `json:"password,omitempty"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		input := &Input{}

		if err := readJSON(r.Body, &input); err != nil {
			badRequestError(w)
			return
		}

		if err := validate.Struct(input); err != nil {
			validationError(w, err)
			return
		}

		ctx := r.Context()
		user := userFromContext(ctx)
		patch := app.UserPatch{
			Email: input.Email,
		}

		if v := input.Password; v != nil {
			user.SetPassword(*v)
		}

		err := s.userService.UpdateUser(ctx, user, patch)
		if err != nil {
			serverError(w, err)
			return
		}

		user.Token = userTokenFromContext(ctx)

		writeJSON(w, http.StatusOK, M{"user": user})
	}
}

func (s *Server) getProfile() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		ctx := r.Context()
		user, err := s.userService.UserByEmail(ctx, vars["email"])

		if err != nil {
			switch {
			case errors.Is(err, app.ErrNotFound):
				err := ErrorM{"user": []string{"user not found"}}
				notFoundError(w, err)
			default:
				serverError(w, err)
			}

			return
		}

		writeJSON(w, http.StatusOK, M{"user": user})
	}
}
