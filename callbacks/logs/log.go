package callback

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloudwego/eino/callbacks"
	"io"
	"log"
	"os"
)

type LogCallbackConfig struct {
	Detail bool
	Debug  bool
	Writer io.Writer
}

func LogCallback(config *LogCallbackConfig) callbacks.Handler {
	if config == nil {
		config = &LogCallbackConfig{
			Detail: true,
			Writer: os.Stdout,
		}
	}
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		_, err := fmt.Fprintf(config.Writer, "[view]: start [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		if err != nil {
			log.Printf("[logCallBack build OnStartFn] writr [view] err=%v", err)
			return nil
		}
		if config.Detail {
			var b []byte
			if config.Debug {
				b, _ = json.MarshalIndent(input, "", "  ")
			} else {
				b, _ = json.Marshal(input)
			}
			_, err = fmt.Fprintf(config.Writer, "%s\n", string(b))
			if err != nil {
				log.Printf("[logCallBack build OnStartFn] writr [input] err=%v", err)
				return nil
			}
		}
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		_, err := fmt.Fprintf(config.Writer, "[view]: end [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		if err != nil {
			log.Printf("[logCallBack build OnEndFn] writr [view] err=%v", err)
			return nil
		}
		return ctx
	})
	return builder.Build()
}

// NewLogCallBack 初始化LogCallBack
func NewLogCallBack() (callbacks.Handler, error) {
	err := os.MkdirAll("log", 0755)
	if err != nil {
		log.Printf("mkdirAll failed, err=%v", err)
		return nil, err
	}
	var f *os.File
	f, err = os.OpenFile("log/agent.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("open logfile err=%v", err)
		return nil, err
	}
	cbConfig := &LogCallbackConfig{
		Detail: true,
		Writer: f,
	}
	if os.Getenv("DEBUG") == "true" {
		cbConfig.Debug = true
	}
	// this is for invoke option of WithCallback
	return LogCallback(cbConfig), nil
}
