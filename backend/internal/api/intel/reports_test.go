package intel

import (
	"errors"
	"net/http"
	"testing"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/pocketbase/pocketbase/tools/router"
)

func TestNormalizeCreateReportError_ValidationBecomesBadRequest(t *testing.T) {
	in := validation.Errors{
		"channel": validation.NewError("validation_required", "Channel is required."),
	}

	out := normalizeCreateReportError(in)
	apiErr := router.ToApiError(out)

	if apiErr.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", apiErr.Status, http.StatusBadRequest)
	}
	if apiErr.Message != "Invalid report payload." {
		t.Fatalf("message = %q", apiErr.Message)
	}
}

func TestNormalizeCreateReportError_InternalErrorStaysInternal(t *testing.T) {
	out := normalizeCreateReportError(errors.New("db unavailable"))
	apiErr := router.ToApiError(out)

	if apiErr.Status != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", apiErr.Status, http.StatusInternalServerError)
	}
	if apiErr.Message != "Failed to save report." {
		t.Fatalf("message = %q", apiErr.Message)
	}
}

func TestNormalizeCreateReportError_ApiErrorPassthrough(t *testing.T) {
	original := router.NewBadRequestError("already bad request", nil)
	out := normalizeCreateReportError(original)
	apiErr := router.ToApiError(out)

	if apiErr.Status != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", apiErr.Status, http.StatusBadRequest)
	}
	if apiErr.Message != "Already bad request." {
		t.Fatalf("message = %q", apiErr.Message)
	}
}

func TestNoSystemsValidationPayloadUses422(t *testing.T) {
	err := router.NewApiError(
		http.StatusUnprocessableEntity,
		"Report text must include at least one valid system.",
		validation.Errors{
			"systems": validation.NewError("validation_missing_system", "At least one valid system name is required."),
		},
	)
	apiErr := router.ToApiError(err)

	if apiErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d", apiErr.Status, http.StatusUnprocessableEntity)
	}
	if apiErr.Message != "Report text must include at least one valid system." {
		t.Fatalf("message = %q", apiErr.Message)
	}
	if _, ok := apiErr.Data["systems"]; !ok {
		t.Fatalf("expected validation data to include systems key")
	}
}
