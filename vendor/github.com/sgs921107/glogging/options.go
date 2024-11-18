package glogging

import (
	"time"
)

// Options	logger options
type Options struct {
	Level          string
	FilePath       string
	Formatter      string
	RotationMaxAge time.Duration
	RotationTime   time.Duration
	Caller         string
	// 仅logrus
	NoLock       bool
	TimeFormater string
	// 使用utc时间
	UseUTC bool
}
