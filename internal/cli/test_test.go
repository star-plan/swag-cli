package cli

import (
	"testing"

	"swag-cli/internal/nginx"
)

func TestInternalTargetURL(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		site nginx.SiteConfig
		want string
	}{
		{
			name: "uses explicit http port",
			site: nginx.SiteConfig{TargetDest: "app", ContainerPort: "8080", UpstreamProto: "http"},
			want: "http://app:8080",
		},
		{
			name: "defaults http port",
			site: nginx.SiteConfig{TargetDest: "app", UpstreamProto: "http"},
			want: "http://app:80",
		},
		{
			name: "defaults https port",
			site: nginx.SiteConfig{TargetDest: "secure-app", UpstreamProto: "https"},
			want: "https://secure-app:443",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := internalTargetURL(tc.site); got != tc.want {
				t.Fatalf("internalTargetURL() = %q, want %q", got, tc.want)
			}
		})
	}
}
