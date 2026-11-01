package inventory

import (
	"reflect"
	"testing"

	"github.com/moby/moby/api/types/container"
)

func TestRunningContainer(t *testing.T) {
	it := containerItem(container.Summary{
		ID: "abc123", Names: []string{"/redis"}, Image: "redis:7", State: container.StateRunning,
		Ports: []container.PortSummary{{PrivatePort: 6379, PublicPort: 6379, Type: "tcp"}, {PrivatePort: 6379, PublicPort: 6379, Type: "tcp"}, {PrivatePort: 9000}},
	})
	if it.ID != "docker:abc123" || it.Name != "redis" || it.Image != "redis:7" || it.Status != StatusOK || !it.HasLogs ||
		!reflect.DeepEqual(it.Actions, []string{"stop", "restart"}) || !reflect.DeepEqual(it.Ports, []int{6379}) || it.Kind != KindDocker {
		t.Fatalf("%+v", it)
	}
}

func TestRestartingContainer(t *testing.T) {
	if it := containerItem(container.Summary{ID: "x", State: container.StateRestarting}); it.Status != StatusWarn {
		t.Fatalf("%+v", it)
	}
}

func TestExitedContainer(t *testing.T) {
	it := containerItem(container.Summary{ID: "x", State: container.StateExited})
	if it.Status != StatusDown || !reflect.DeepEqual(it.Actions, []string{"start"}) {
		t.Fatalf("%+v", it)
	}
}
