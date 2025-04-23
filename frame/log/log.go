package log

import "github.com/sirupsen/logrus"

// Config 日志配置
// example:
//
//	{
//		"name": "example",
//		"level": 5,
//		"json": false,
//		"outputs": {
//			"file": {
//				"path": "./logs"
//				"rotate": true,
//				"json": true,
//			},
//			"logstash": {
//				"addr": "localhost:5000",
//			}
//		}
//	}
type Config struct {
	Name    string                    `json:"name"`
	Level   int                       `json:"level"`
	UseJSON bool                      `json:"json"`
	Outputs map[string]map[string]any `json:"outputs"`
}

// InitLogrus 根据配置初始化 logrus, 添加配置的 Hooks
func InitLogrus(cfg *Config) (close func(), err error) {
	logrus.SetLevel(logrus.Level(cfg.Level))
	if cfg.UseJSON {
		logrus.SetFormatter(new(logrus.JSONFormatter))
	} else {
		text := new(logrus.TextFormatter)
		text.FullTimestamp = true
		logrus.SetFormatter(text)
	}
	return addLogHooks(logrus.StandardLogger(), cfg)
}

func addLogHooks(logger *logrus.Logger, cfg *Config) (close func(), err error) {
	var fcloses []func()
	defer func() {
		if err != nil {
			for _, f := range fcloses {
				f()
			}
		}
	}()
	if _, ok := cfg.Outputs["file"]; ok {
		close, err = addFileHook(logger, cfg)
		if err != nil {
			return nil, err
		}
		fcloses = append(fcloses, close)
	}
	if _, ok := cfg.Outputs["logstash"]; ok {
		close, err = addLogstashHook(logger, cfg)
		if err != nil {
			return nil, err
		}
		fcloses = append(fcloses, close)
	}
	return func() {
		for _, f := range fcloses {
			f()
		}
	}, nil
}

func addFileHook(logger *logrus.Logger, cfg *Config) (close func(), err error) {
	var path string
	var rotate bool
	var useJSON = cfg.UseJSON
	var maxSize int

	if v, ok := cfg.Outputs["file"]["path"].(string); ok {
		path = v
	}
	if v, ok := cfg.Outputs["file"]["rotate"].(bool); ok {
		rotate = v
	}
	if v, ok := cfg.Outputs["file"]["json"].(bool); ok {
		useJSON = v
	}
	if v, ok := cfg.Outputs["file"]["maxsize"].(float64); ok {
		maxSize = int(v)
	}

	hook, close, err := NewFileLogHook(path, cfg.Name, useJSON, rotate, maxSize)
	if err != nil {
		return nil, err
	}

	logger.AddHook(hook)

	return close, nil
}

func addLogstashHook(logger *logrus.Logger, cfg *Config) (close func(), err error) {
	var addr string

	if v, ok := cfg.Outputs["logstash"]["addr"].(string); ok {
		addr = v
	}

	log := logger.WithField("addr", addr)

	log.Info("Connecting logstash...")
	hook, close, err := NewLogstashHook(addr, cfg.Name)
	if err != nil {
		log.WithError(err).Error("Connect logstash failed")
		return nil, err
	}
	log.Info("Connect logstash succ")

	logger.AddHook(hook)

	return close, nil
}
