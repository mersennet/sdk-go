package mersennet

import "testing"

func TestToHexAmount(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"zero", "0", "0x0"},
		{"small decimal", "255", "0xff"},
		{"already hex lowercase", "0x1f4", "0x1f4"},
		{"already hex uppercase prefix", "0X1f4", "0X1f4"},
		{"large decimal beyond uint64", "618970019642690137449562111", "0x1ffffffffffffffffffffff"},
		{"invalid falls back to 0x0", "not-a-number", "0x0"},
		{"empty falls back to 0x0", "", "0x0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := toHexAmount(tc.in); got != tc.want {
				t.Fatalf("toHexAmount(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}
