package gpbckpconfig

import "testing"

func TestCheckTimestamp(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "Test valid timestamp",
			value:   "20230822120000",
			wantErr: false,
		},
		{
			name:    "Test invalid timestamp",
			value:   "invalid",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckTimestamp(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestCheckFullPath(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{
			name:    "Test full path",
			value:   "/som/path/test.txt",
			wantErr: false,
		},
		{
			name:    "Test zero length path",
			value:   "",
			wantErr: true,
		},
		{
			name:    "Test not full path",
			value:   "test.txt",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckFullPath(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("\nVariables do not match:\n%v\nwantErr:\n%v", err, tt.wantErr)
			}
		})
	}
}

func TestIsBackupActive(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{
			name:  "Test empty delete date",
			value: "",
			want:  true,
		},
		{
			name:  "Test plugin error",
			value: DateDeletedPluginFailed,
			want:  true,
		},
		{
			name:  "Test local error",
			value: DateDeletedLocalFailed,
			want:  true,
		},
		{
			name:  "Test in progress",
			value: DateDeletedInProgress,
			want:  true,
		},
		{
			name:  "Test deleted",
			value: "20220401102430",
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsBackupActive(tt.value); got != tt.want {
				t.Errorf("\nVariables do not match:\n%v\nwant:\n%v", got, tt.want)
			}
		})
	}
}
