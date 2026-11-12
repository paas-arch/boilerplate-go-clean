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
		{name: "есть адрес базы", cfg: Config{DatabaseDSN: "postgres://app@db/app"}},
		{name: "нет адреса базы", cfg: Config{}, wantErr: true},
		{name: "битый CIDR прокси", cfg: Config{DatabaseDSN: "postgres://app@db/app", TrustedProxyCIDRs: []string{"10.0.0.0/33"}}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantErr, tt.cfg.Validate() != nil)
		})
	}
}
