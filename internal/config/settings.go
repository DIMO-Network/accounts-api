package config

import (
	"github.com/DIMO-Network/shared/db"
)

// Settings contains the application config
type Settings struct {
	Environment             string      `yaml:"ENVIRONMENT"`
	Port                    string      `yaml:"PORT"`
	GRPCPort                string      `yaml:"GRPC_PORT"`
	LogLevel                string      `yaml:"LOG_LEVEL"`
	DB                      db.Settings `yaml:"DB"`
	ServiceName             string      `yaml:"SERVICE_NAME"`
	EmailHost               string      `yaml:"EMAIL_HOST"`
	EmailPort               string      `yaml:"EMAIL_PORT"`
	EmailUsername           string      `yaml:"EMAIL_USERNAME"`
	EmailPassword           string      `yaml:"EMAIL_PASSWORD"`
	EmailFrom               string      `yaml:"EMAIL_FROM"`
	JWTKeySetURL            string      `yaml:"JWT_KEY_SET_URL"`
	KafkaBrokers            string      `yaml:"KAFKA_BROKERS"`
	EventsTopic             string      `yaml:"EVENTS_TOPIC"`
	MonitoringPort          string      `yaml:"MON_PORT"`
	DevicesAPIGRPCAddr      string      `yaml:"DEVICES_API_GRPC_ADDR"`
	IdentityAPIURL          string      `yaml:"IDENTITY_API_URL"`
	EmailCodeDuration       string      `yaml:"EMAIL_CODE_DURATION"`
	CustomerIOURL           string      `yaml:"CUSTOMER_IO_URL"`
	CustomerIOAPIKey        string      `yaml:"CUSTOMER_IO_API_KEY"`
	DisableCustomerIOEvents bool        `yaml:"DISABLE_CUSTOMER_IO_EVENTS"`
}

func (s *Settings) IsProduction() bool {
	return s.Environment == "prod" // this string is set in the helm chart values-prod.yaml
}
