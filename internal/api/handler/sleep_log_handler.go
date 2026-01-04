package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/api/validation"
	"github.com/blaisecz/sleep-tracker/internal/domain"
	"github.com/blaisecz/sleep-tracker/internal/service"
	"github.com/blaisecz/sleep-tracker/pkg/problem"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type SleepLogHandler struct {
	service service.SleepLogService
}

func NewSleepLogHandler(service service.SleepLogService) *SleepLogHandler {
	return &SleepLogHandler{service: service}
}

// Create handles POST /v1/users/{userId}/sleep-logs
// @Summary Create a sleep log
// @Description Record a new sleep session for a user. Supports idempotency via client_request_id.
// @Tags sleep-logs
// @Accept json
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Param request body domain.CreateSleepLogRequest true "Sleep log creation request"
// @Success 201 {object} domain.SleepLogResponse "Created new sleep log"
// @Success 200 {object} domain.SleepLogResponse "Returned existing sleep log (idempotent)"
// @Failure 400 {object} problem.Problem
// @Failure 404 {object} problem.Problem "User not found"
// @Failure 409 {object} problem.Problem "Overlapping sleep period"
// @Failure 500 {object} problem.Problem
// @Router /users/{userId}/sleep-logs [post]
func (h *SleepLogHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	var req domain.CreateSleepLogRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		problem.BadRequest("Invalid JSON body").Write(w)
		return
	}

	if fieldErrors := validation.Validate(req); fieldErrors != nil {
		problem.ValidationError("Request body contains invalid fields", fieldErrors).Write(w)
		return
	}

	log, isExisting, err := h.service.Create(r.Context(), userID, &req)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			problem.NotFound("User not found").Write(w)
			return
		}
		if errors.Is(err, domain.ErrOverlappingSleep) {
			problem.Conflict("Overlapping sleep period detected").Write(w)
			return
		}
		problem.InternalError("Failed to create sleep log").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if isExisting {
		w.WriteHeader(http.StatusOK) // Return 200 for idempotent duplicate
	} else {
		w.WriteHeader(http.StatusCreated)
	}
	json.NewEncoder(w).Encode(log.ToResponse())
}

// List handles GET /v1/users/{userId}/sleep-logs
// @Summary List sleep logs
// @Description Get paginated sleep logs for a user with optional date filtering
// @Tags sleep-logs
// @Produce json
// @Param userId path string true "User ID" format(uuid)
// @Param from query string false "Filter logs starting from this time" format(date-time)
// @Param to query string false "Filter logs up to this time" format(date-time)
// @Param limit query integer false "Number of results per page" default(20) minimum(1) maximum(100)
// @Param cursor query string false "Pagination cursor from previous response"
// @Success 200 {object} domain.SleepLogListResponse
// @Failure 400 {object} problem.Problem
// @Failure 404 {object} problem.Problem "User not found"
// @Failure 500 {object} problem.Problem
// @Router /users/{userId}/sleep-logs [get]
func (h *SleepLogHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := uuid.Parse(chi.URLParam(r, "userId"))
	if err != nil {
		problem.BadRequest("Invalid user ID format").Write(w)
		return
	}

	filter, fieldErrors := parseListFilter(r)
	if fieldErrors != nil {
		problem.ValidationError("Invalid query parameters", fieldErrors).Write(w)
		return
	}

	response, err := h.service.List(r.Context(), userID, filter)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			problem.NotFound("User not found").Write(w)
			return
		}
		problem.InternalError("Failed to list sleep logs").Write(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func parseListFilter(r *http.Request) (domain.SleepLogFilter, []problem.FieldError) {
	var filter domain.SleepLogFilter
	var fieldErrors []problem.FieldError

	// Parse 'from' parameter
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			fieldErrors = append(fieldErrors, problem.FieldError{
				Field:   "from",
				Message: "must be a valid RFC3339 timestamp",
			})
		} else {
			filter.From = &from
		}
	}

	// Parse 'to' parameter
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			fieldErrors = append(fieldErrors, problem.FieldError{
				Field:   "to",
				Message: "must be a valid RFC3339 timestamp",
			})
		} else {
			filter.To = &to
		}
	}

	// Parse 'limit' parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			fieldErrors = append(fieldErrors, problem.FieldError{
				Field:   "limit",
				Message: "must be a positive integer",
			})
		} else {
			filter.Limit = limit
		}
	}

	// Parse 'cursor' parameter
	filter.Cursor = r.URL.Query().Get("cursor")

	if len(fieldErrors) > 0 {
		return filter, fieldErrors
	}

	return filter, nil
}
