package docker

import "testing"

func TestDeriveProjectStatus(t *testing.T) {
	tests := []struct {
		name       string
		containers []ProjectContainer
		want       string
	}{
		{
			name: "not found when project has no containers",
			want: StatusNotFound,
		},
		{
			name: "running when all containers are running",
			containers: []ProjectContainer{
				{Name: "demo-app", State: "running"},
				{Name: "demo-db", State: "running"},
			},
			want: StatusRunning,
		},
		{
			name: "stopped when all containers are exited",
			containers: []ProjectContainer{
				{Name: "demo-app", State: "exited"},
				{Name: "demo-db", State: "dead"},
			},
			want: StatusStopped,
		},
		{
			name: "degraded when container states are mixed",
			containers: []ProjectContainer{
				{Name: "demo-app", State: "running"},
				{Name: "demo-db", State: "exited"},
			},
			want: StatusDegraded,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveProjectStatus(tt.containers)
			if got != tt.want {
				t.Fatalf("expected status %s, got %s", tt.want, got)
			}
		})
	}
}
