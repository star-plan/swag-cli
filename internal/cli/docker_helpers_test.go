package cli

import (
	"context"
	"errors"
	"testing"
)

type fakeRestarter struct {
	restarted string
	err       error
}

func (f *fakeRestarter) RestartContainer(ctx context.Context, containerName string) error {
	f.restarted = containerName
	return f.err
}

func TestRestartWithClient(t *testing.T) {
	t.Parallel()

	restarter := &fakeRestarter{}
	if err := restartWithClient(restarter, "swag"); err != nil {
		t.Fatalf("restartWithClient() error = %v", err)
	}
	if restarter.restarted != "swag" {
		t.Fatalf("expected restart target swag, got %q", restarter.restarted)
	}
}

func TestRestartWithClientReturnsWrappedError(t *testing.T) {
	t.Parallel()

	restarter := &fakeRestarter{err: errors.New("boom")}
	if err := restartWithClient(restarter, "swag"); err == nil {
		t.Fatalf("expected restartWithClient error")
	}
}

func TestRestartSwagContainerByNameRejectsEmptyName(t *testing.T) {
	t.Parallel()

	if err := restartSwagContainerByName("   "); err == nil {
		t.Fatalf("expected empty-name error")
	}
}
