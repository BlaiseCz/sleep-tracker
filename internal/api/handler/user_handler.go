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

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

// Create handles POST /v1/users
// @Summary Create user
// @Description Register a new user with their preferred timezone. The timezone is used for displaying sleep times in local format.
// @Tags users
// @Accept json
// @Produce json
// @Param request body domain.CreateUserRequest true "User data" example({"timezone": "Europe/Prague"})
// @Success 201 {object} domain.UserResponse "User created successfully"
// @Failure 400 {object} problem.Problem "Invalid request (malformed JSON or invalid timezone)"
// @Failure 500 {object} problem.Problem "Server error"
// @Router /users [post]
func (h *UserHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&req); err != nil {
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
// @Summary Get user
// @Description Retrieve user details including timezone preference.
// @Tags users
// @Produce json
// @Param userId path string true "User UUID" format(uuid) example(550e8400-e29b-41d4-a716-446655440000)
// @Success 200 {object} domain.UserResponse "User details"
// @Failure 400 {object} problem.Problem "Invalid UUID format"
// @Failure 404 {object} problem.Problem "User not found"
// @Failure 500 {object} problem.Problem "Server error"
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
