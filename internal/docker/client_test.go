package docker

import (
	"bufio"
	"bytes"
	"context"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/pkg/stdcopy"
)

type fakeDockerAPI struct {
	containers        []types.Container
	containerListErr  error
	restartedName     string
	restartErr        error
	inspectContainer  types.ContainerJSON
	containerPingErr  error
	execCreateResp    types.IDResponse
	execCreateErr     error
	execAttachResp    types.HijackedResponse
	execAttachErr     error
	execInspectResp   types.ContainerExecInspect
	execInspectErr    error
}

func (f *fakeDockerAPI) Ping(ctx context.Context) (types.Ping, error) {
	return types.Ping{}, f.containerPingErr
}

func (f *fakeDockerAPI) ContainerList(ctx context.Context, options container.ListOptions) ([]types.Container, error) {
	return f.containers, f.containerListErr
}

func (f *fakeDockerAPI) ContainerExecCreate(ctx context.Context, container string, config types.ExecConfig) (types.IDResponse, error) {
	return f.execCreateResp, f.execCreateErr
}

func (f *fakeDockerAPI) ContainerExecAttach(ctx context.Context, execID string, config types.ExecStartCheck) (types.HijackedResponse, error) {
	return f.execAttachResp, f.execAttachErr
}

func (f *fakeDockerAPI) ContainerExecInspect(ctx context.Context, execID string) (types.ContainerExecInspect, error) {
	return f.execInspectResp, f.execInspectErr
}

func (f *fakeDockerAPI) ContainerInspect(ctx context.Context, container string) (types.ContainerJSON, error) {
	return f.inspectContainer, nil
}

func (f *fakeDockerAPI) ContainerRestart(ctx context.Context, container string, options container.StopOptions) error {
	f.restartedName = container
	return f.restartErr
}

func (f *fakeDockerAPI) Close() error {
	return nil
}

func TestListContainersByNetwork(t *testing.T) {
	t.Parallel()

	api := &fakeDockerAPI{
		containers: []types.Container{
			{
				ID:     "1",
				Names:  []string{"/app"},
				Image:  "app:latest",
				State:  "running",
				Status: "Up",
				NetworkSettings: &types.SummaryNetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"swag": {IPAddress: "10.0.0.2"},
					},
				},
			},
			{
				ID:     "2",
				Names:  []string{"/db"},
				Image:  "db:latest",
				State:  "running",
				Status: "Up",
				NetworkSettings: &types.SummaryNetworkSettings{
					Networks: map[string]*network.EndpointSettings{
						"other": {IPAddress: "10.0.0.3"},
					},
				},
			},
		},
	}

	client := &Client{cli: api}
	got, err := client.ListContainersByNetwork(context.Background(), "swag")
	if err != nil {
		t.Fatalf("ListContainersByNetwork error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 container, got %d", len(got))
	}
	if got[0].Name != "app" || got[0].IP != "10.0.0.2" {
		t.Fatalf("unexpected container info: %+v", got[0])
	}
}

func TestListContainersByNetworkReturnsErrorWhenEmpty(t *testing.T) {
	t.Parallel()

	client := &Client{cli: &fakeDockerAPI{}}
	if _, err := client.ListContainersByNetwork(context.Background(), "swag"); err == nil {
		t.Fatalf("expected error when no containers matched")
	}
}

func TestRestartContainerPassesContainerName(t *testing.T) {
	t.Parallel()

	api := &fakeDockerAPI{}
	client := &Client{cli: api}
	if err := client.RestartContainer(context.Background(), "swag"); err != nil {
		t.Fatalf("RestartContainer error: %v", err)
	}
	if api.restartedName != "swag" {
		t.Fatalf("expected container name swag, got %q", api.restartedName)
	}
}

func TestExecReturnsStdoutOnSuccess(t *testing.T) {
	t.Parallel()

	api := &fakeDockerAPI{
		execCreateResp:  types.IDResponse{ID: "exec-1"},
		execAttachResp:  newHijackedResponse(t, "hello\n", ""),
		execInspectResp: types.ContainerExecInspect{ExitCode: 0},
	}

	client := &Client{cli: api}
	out, err := client.Exec(context.Background(), "swag", []string{"echo", "hello"})
	if err != nil {
		t.Fatalf("Exec error: %v", err)
	}
	if out != "hello\n" {
		t.Fatalf("Exec output = %q, want %q", out, "hello\n")
	}
}

func TestReloadNginxReturnsStderrOnFailure(t *testing.T) {
	t.Parallel()

	api := &fakeDockerAPI{
		execCreateResp:  types.IDResponse{ID: "exec-1"},
		execAttachResp:  newHijackedResponse(t, "", "reload failed"),
		execInspectResp: types.ContainerExecInspect{ExitCode: 1},
	}

	client := &Client{cli: api}
	err := client.ReloadNginx(context.Background(), "swag")
	if err == nil {
		t.Fatalf("expected ReloadNginx error")
	}
	if got := err.Error(); got == "" || !strings.Contains(got, "reload failed") {
		t.Fatalf("ReloadNginx error = %q, want stderr details", got)
	}
}

type dummyConn struct{}

func (dummyConn) Read(b []byte) (int, error)         { return 0, nil }
func (dummyConn) Write(b []byte) (int, error)        { return len(b), nil }
func (dummyConn) Close() error                       { return nil }
func (dummyConn) LocalAddr() net.Addr                { return dummyAddr("local") }
func (dummyConn) RemoteAddr() net.Addr               { return dummyAddr("remote") }
func (dummyConn) SetDeadline(time.Time) error        { return nil }
func (dummyConn) SetReadDeadline(time.Time) error    { return nil }
func (dummyConn) SetWriteDeadline(time.Time) error   { return nil }

type dummyAddr string

func (a dummyAddr) Network() string { return string(a) }
func (a dummyAddr) String() string  { return string(a) }

func newHijackedResponse(t *testing.T, stdout string, stderr string) types.HijackedResponse {
	t.Helper()

	var mux bytes.Buffer
	if stdout != "" {
		w := stdcopy.NewStdWriter(&mux, stdcopy.Stdout)
		if _, err := w.Write([]byte(stdout)); err != nil {
			t.Fatalf("write stdout multiplexed data: %v", err)
		}
	}
	if stderr != "" {
		w := stdcopy.NewStdWriter(&mux, stdcopy.Stderr)
		if _, err := w.Write([]byte(stderr)); err != nil {
			t.Fatalf("write stderr multiplexed data: %v", err)
		}
	}

	return types.HijackedResponse{
		Conn:   dummyConn{},
		Reader: bufio.NewReader(bytes.NewReader(mux.Bytes())),
	}
}
