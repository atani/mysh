package config

import "testing"

func TestIsRedash(t *testing.T) {
	tests := []struct {
		name string
		conn Connection
		want bool
	}{
		{
			name: "no redash config",
			conn: Connection{Name: "db"},
			want: false,
		},
		{
			name: "redash present with URL",
			conn: Connection{Redash: &RedashConfig{URL: "https://redash.example.com"}},
			want: true,
		},
		{
			name: "redash present but empty URL",
			conn: Connection{Redash: &RedashConfig{URL: ""}},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conn.IsRedash(); got != tt.want {
				t.Errorf("IsRedash() = %v, want %v", got, tt.want)
			}
		})
	}
}
