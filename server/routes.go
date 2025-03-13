package server

import (
	"os"

	"github.com/rs/cors"
)

const (
	MustAuth     = true
	OptionalAuth = false
)

func (s *Server) routes() {
	s.router.Use(cors.AllowAll().Handler)
	s.router.Use(Logger(os.Stdout))
	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()

	noAuth := apiRouter.PathPrefix("").Subrouter()
	{
		noAuth.Handle("/health", healthCheck())
		noAuth.Handle("/auth/signup", s.createUser()).Methods("POST")
		noAuth.Handle("/auth/login", s.loginUser()).Methods("POST")
	}

	authApiRoutes := apiRouter.PathPrefix("").Subrouter()
	authApiRoutes.Use(s.authenticate(MustAuth))
	{
		authApiRoutes.Handle("/user", s.getCurrentUser()).Methods("GET")
		authApiRoutes.Handle("/user", s.updateUser()).Methods("PUT", "PATCH")
		authApiRoutes.Handle("/collections", s.createCollection()).Methods("POST")
		authApiRoutes.Handle("/collections", s.listCollections()).Methods("GET")
		authApiRoutes.Handle("/collections/{id}", s.getCollection()).Methods("GET")
		authApiRoutes.Handle("/collections/{id}", s.updateCollection()).Methods("PUT", "PATCH")
		authApiRoutes.Handle("/collections/{id}", s.deleteCollection()).Methods("DELETE")
		authApiRoutes.Handle("/collections/{id}/images", s.saveImageToCollection()).Methods("POST")
		authApiRoutes.Handle("/collections/{id}/images/{imagePath}", s.deleteImageFromCollection()).Methods("DELETE")
	}
}
