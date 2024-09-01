package config

import "github.com/spf13/viper"

type Config struct {
	Server struct {
		GrpcPort int `mapstructure:"grpc_port"`
		HttpPort int `mapstructure:"http_port"`
	} `mapstructure:"server"`

	Database struct {
		URI      string `mapstructure:"uri"`
		DB       string `mapstructure:"db"`
		Password string `mapstructure:"password"`
	} `mapstructure:"database"`

	Redis struct {
		Host string `mapstructure:"host"`
		Port int    `mapstructure:"port"`
	} `mapstructure:"redis"`

	RabbitMQ struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
	} `mapstructure:"rabbitmq"`

	UserService struct {
		JwtSecret       string `mapstructure:"jwt_secret"`
		TokenExpiration string `mapstructure:"token_expiration"`
	} `mapstructure:"user_service"`

	PostService struct {
		MaxPostLength int `mapstructure:"max_post_length"`
	} `mapstructure:"post_service"`

	CommentService struct {
		MaxCommentLength int `mapstructure:"max_comment_length"`
	} `mapstructure:"comment_service"`

	Logging struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logging"`
}

var cfg *Config

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	cfg = &Config{}
	err = viper.Unmarshal(cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
