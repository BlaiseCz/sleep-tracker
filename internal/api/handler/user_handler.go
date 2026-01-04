package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/blaisecz/sleep-tracker/internal/api/validation"
	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/service"
	"github.com/blaisecz/sleep-tracker/pkg/problem"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// @title Sleep Tracker API
// @version 1.0
// @description API for tracking sleep patterns and quality
// @BasePath /v1

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create handles POST /v1/users
// @Summary Create a new user
// @Description Create a new user with timezone preference
// @Tags users
// @Accept json
// @Produce json
// @Param request body domain.CreateUserRequest true "User creation request"
// @Success 201 {object} domain.UserResponse
// @Failure 400 {object} problem.Problem
// @Failure 500 {object} problem.Problem
// @Router /users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		problem.BadRequest("Invalid JSON body").Write(w)
		return
	}

	if fieldErrors := validation.Validate(req); fieldErrors != nil {
		problem.ValidationError("Request body contains invalid fields", fieldErrors).Write(w)
		return
	}

	user, err := h.service.Create(r.Context(), &req)
	if err != nil {
		problem.InternalError("Failed to create user").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(user.ToResponse())
}

// GetByID handles GET /v1/users/{userId}
// @Summary Get user by ID
// @Description Get a user's details by their UUID
// @Tags users
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Success 200 {object} domain.UserResponse
// @Failure 400 {object} problem.Problem
// @Failure 404 {object} problem.Problem
// @Failure 500 {object} problem.Problem
// @Router /users/{userId} [get]
func (h *UserHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	user, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			problem.NotFound("User not found").Write(w)
			return
		}
		problem.InternalError("Failed to get user").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user.ToResponse())
}
