package db

import (
	"time"

	"github.com/company/go-fintech/internal/config"
)

func NewPoolFromConfig(cfg config.DatabaseConfig) (*Pool, error) {
	return NewPool(Config{
		Host:            cfg.Host,
		Port:            cfg.Port,
		User:            cfg.User,
		Password:        cfg.Password,
		DBName:          cfg.Name,
		MaxConns:        int32(cfg.MaxOpenConns),
		MinConns:        int32(cfg.MaxIdleConns),
		MaxConnLifetime: time.Duration(cfg.ConnMaxLifetime) * time.Second,
		MaxConnIdleTime: time.Duration(cfg.ConnMaxIdleTime) * time.Second,
	})
}
