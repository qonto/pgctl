package cli

import (
	"testing"
)

func TestIsNotEmpty(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Non-empty string",
			value:   "test",
			wantErr: false,
		},
		{
			name:    "Empty string",
			value:   "",
			wantErr: true,
			errMsg:  "value is required",
		},
		{
			name:    "Whitespace only",
			value:   "   ",
			wantErr: true,
			errMsg:  "value is required",
		},
		{
			name:    "Tab character",
			value:   "\t",
			wantErr: true,
			errMsg:  "value is required",
		},
		{
			name:    "Newline character",
			value:   "\n",
			wantErr: true,
			errMsg:  "value is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isNotEmpty(tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("isNotEmpty() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && err.Error() != tt.errMsg {
				t.Errorf("isNotEmpty() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestIsValidHost(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Valid hostname",
			host:    "example.com",
			wantErr: false,
		},
		{
			name:    "Empty hostname",
			host:    "",
			wantErr: true,
			errMsg:  "host is required",
		},
		{
			name:    "Too long hostname",
			host:    string(make([]byte, 254)),
			wantErr: true,
			errMsg:  "host name too long",
		},
		{
			name:    "Invalid characters",
			host:    "example@domain.com",
			wantErr: true,
			errMsg:  "host contains invalid characters",
		},
		{
			name:    "Starts with dot",
			host:    ".example.com",
			wantErr: true,
			errMsg:  "host cannot start or end with dots or hyphens",
		},
		{
			name:    "Ends with hyphen",
			host:    "example.com-",
			wantErr: true,
			errMsg:  "host cannot start or end with dots or hyphens",
		},
		{
			name:    "Valid with hyphen",
			host:    "my-example.com",
			wantErr: false,
		},
		{
			name:    "Valid subdomain",
			host:    "sub.example.com",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidHost(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("isValidHost() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("isValidHost() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}

func TestIsValidPort(t *testing.T) {
	tests := []struct {
		name    string
		port    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Empty string",
			port:    "",
			wantErr: false,
		},
		{
			name:    "Valid port",
			port:    "8080",
			wantErr: false,
		},
		{
			name:    "Non-numeric port",
			port:    "abc",
			wantErr: true,
			errMsg:  "port must be a number",
		},
		{
			name:    "Port too low",
			port:    "0",
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name:    "Port too high",
			port:    "65536",
			wantErr: true,
			errMsg:  "port must be between 1 and 65535",
		},
		{
			name:    "Minimum valid port",
			port:    "1",
			wantErr: false,
		},
		{
			name:    "Maximum valid port",
			port:    "65535",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidPort(tt.port)
			if (err != nil) != tt.wantErr {
				t.Errorf("isValidPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr && err.Error() != tt.errMsg {
				t.Errorf("isValidPort() error message = %v, want %v", err.Error(), tt.errMsg)
			}
		})
	}
}
