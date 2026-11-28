package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_Validate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{name: "настройки по умолчанию", cfg: Config{}},
		{name: "верный CIDR прокси", cfg: Config{TrustedProxyCIDRs: []string{"10.0.0.0/8"}}},
		{name: "битый CIDR прокси", cfg: Config{TrustedProxyCIDRs: []string{"10.0.0.0/33"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, tt.cfg.Validate() != nil)
		})
	}
}
