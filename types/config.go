package types

import (
	"net/url"
	"time"
)

type Config struct {
	LogEnv                    string        `env:"LOG_ENV"                              envDefault:"development"`
	Host                      string        `env:"HOST"                                 envDefault:""`
	Port                      int           `env:"PORT,notEmpty"                        envDefault:"8080"`
	DBUser                    string        `env:"DB_USER,notEmpty"`
	DBPassword                string        `env:"DB_PASSWORD,notEmpty"`
	DBHost                    string        `env:"DB_HOST"                              envDefault:"localhost"`
	DBPort                    int           `env:"DB_PORT"                              envDefault:"5432"`
	DBName                    string        `env:"DB_NAME,notEmpty"`
	DBMaxOpenConnections      int           `env:"DB_MAX_OPEN_CONNECTIONS"              envDefault:"10"`
	DBMaxIdleConnections      int           `env:"DB_MAX_IDLE_CONNECTIONS"              envDefault:"10"`
	DBConnectionMaxLifetime   time.Duration `env:"DB_CONNECTION_MAX_LIFETIME"           envDefault:"1h"`
	BaseURL                   url.URL       `env:"BASE_URL"                             envDefault:"http://localhost:8080"`
	PostmarkToken             string        `env:"POSTMARK_TOKEN"`
	MarketingEmailAddress     string        `env:"MARKETING_EMAIL_ADDRESS,notEmpty"`
	TransactionalEmailAddress string        `env:"TRANSACTIONAL_EMAIL_ADDRESS,notEmpty"`
	AWSAccessKeyID            string        `env:"AWS_ACCESS_KEY_ID,notEmpty"`
	AWSSecretAccessKey        string        `env:"AWS_SECRET_ACCESS_KEY,notEmpty"`
	AdminPassword             string        `env:"ADMIN_PASSWORD,notEmpty"`
}
