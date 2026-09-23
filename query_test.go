package incident

import (
	"net/url"
	"testing"

	"github.com/oapi-codegen/runtime"
)

// TestStyleParamDeepObject verifies filters are sent with repeated, unindexed
// keys, which is the only form in which the API keeps every value.
func TestStyleParamDeepObject(t *testing.T) {
	opts := runtime.StyleParamOptions{ParamLocation: runtime.ParamLocationQuery, Type: "object"}

	for _, tc := range []struct {
		name  string
		param string
		value any
		want  url.Values
	}{
		{
			name:  "single value",
			param: "created_at",
			value: map[string][]string{"gte": {"2024-05-01"}},
			want:  url.Values{"created_at[gte]": {"2024-05-01"}},
		},
		{
			name:  "multiple values keep their order",
			param: "status",
			value: map[string][]string{"one_of": {"triage", "active", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
			want:  url.Values{"status[one_of]": {"triage", "active", "a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}},
		},
		{
			name:  "nested filter",
			param: "custom_field",
			value: map[string]map[string][]string{"01FIELD": {"one_of": {"x", "y"}}},
			want:  url.Values{"custom_field[01FIELD][one_of]": {"x", "y"}},
		},
		{
			name:  "value with characters that need escaping",
			param: "tags",
			value: map[string][]string{"one_of": {"a&b=c"}},
			want:  url.Values{"tags[one_of]": {"a&b=c"}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frag, err := styleParamDeepObject(true, tc.param, tc.value, opts)
			if err != nil {
				t.Fatalf("styleParamDeepObject: %v", err)
			}
			if want := tc.want.Encode(); frag != want {
				t.Errorf("got %s, want %s", frag, want)
			}
		})
	}
}
