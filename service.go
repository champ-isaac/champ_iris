package champiris

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"runtime"
	"time"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"

	stdContext "context"
)

type Service struct {
	App         *iris.Application
	Config      *NetConfig
	Versions    []string
	HtmlPath    string
}

func (service *Service) New(config *NetConfig) error {
	service.App = iris.New()

	if len(service.Versions) == 0 { // set the default version
		service.Versions = []string{"1"}
	}

	if len(service.HtmlPath) == 0 { // set the default html path
		_, currentFilePath, _, _ := runtime.Caller(0)
		configFilePath := path.Join(path.Dir(currentFilePath), "https/web")
		service.HtmlPath = configFilePath
	}

	if config.Port == "" {
		return errors.New("network port not set")
	}

	service.Config = config

	requestLog, loggerClose := service.newRequestLogger()
	service.App.Use(requestLog)
	service.App.OnAnyErrorCode(requestLog, func(ctx iris.Context) {
		ctx.Values().Set("logger_message", "a dynamic message passed to the logs")
	})

	iris.RegisterOnInterrupt(func() {
		if err := loggerClose(); err != nil {
			service.App.Logger().Fatal(err)
		}

		timeout := 5 * time.Second
		ctx, cancel := stdContext.WithTimeout(stdContext.Background(), timeout)
		defer cancel()
		//close all hosts
		service.App.Shutdown(ctx)
	})

	service.registerStaticWebPages(service.HtmlPath)
	//同時間可能會存在多個版本的API，目前先暫定同時存在2個版本，前一版跟最新版
	service.setVersionRoutingPath(service.Versions, service.Config.LoggerEnable)

	return nil
}

func (service *Service) Run() error {
	err := service.App.Run(
		iris.Addr(":"+service.Config.Port),
		iris.WithOptimizations,
		iris.WithoutServerError(iris.ErrServerClosed),
	)
	return err
}

func (service *Service) Interrupt() error {
	err := service.App.Shutdown(stdContext.Background())
	return err
}

func (service *Service) registerStaticWebPages(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, os.ModePerm)
		if err != nil {
			log.Fatal(fmt.Sprintf("Create %s error: ", path), err)
		}
	}
	service.App.RegisterView(iris.HTML(path, ".html").Reload(true))
}

func (service *Service) setVersionRoutingPath(versions []string, loggerEnable bool) {
	for _, version := range versions {
		if loggerEnable == true {
			mvc.Configure(service.App.Party("/service/v"+version), RoutesWithLogger)
		} else {
			mvc.Configure(service.App.Party("/service/v"+version), Routes)
		}
	}
}