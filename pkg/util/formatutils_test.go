package util

import "testing"

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		name string
		size int64
		want string
	}{
		{"Bytes", 500, "500 B"},
		{"KB", 1024, "1.00 KB"},
		{"KB_plus", 1500, "1.46 KB"},
		{"MB", 1024 * 1024, "1.00 MB"},
		{"GB", 1024 * 1024 * 1024, "1.00 GB"},
		{"TB", 1024 * 1024 * 1024 * 1024, "1.00 TB"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatFileSize(tt.size); got != tt.want {
				t.Errorf("FormatFileSize(%d) = %v, want %v", tt.size, got, tt.want)
			}
		})
	}
}
