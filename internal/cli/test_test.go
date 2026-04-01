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

func TestFilterSitesByName(t *testing.T) {
	t.Parallel()

	sites := []nginx.SiteConfig{
		{Name: "app", Type: nginx.TypeSubdomain},
		{Name: "example.com", Type: nginx.TypeHomepage},
	}

	got := filterSitesByName(sites, "app")
	if len(got) != 1 || got[0].Name != "app" {
		t.Fatalf("filterSitesByName(app) = %+v", got)
	}

	got = filterSitesByName(sites, "homepage")
	if len(got) != 1 || got[0].Type != nginx.TypeHomepage {
		t.Fatalf("filterSitesByName(homepage) = %+v", got)
	}
}

func TestFormatCheckError(t *testing.T) {
	t.Parallel()

	errText := "line1\nline2\r\nline3"
	got := formatCheckError(assertErr(errText))
	want := "line1 line2 line3"
	if got != want {
		t.Fatalf("formatCheckError() = %q, want %q", got, want)
	}
}

type stringErr string

func (e stringErr) Error() string { return string(e) }

func assertErr(s string) error { return stringErr(s) }
