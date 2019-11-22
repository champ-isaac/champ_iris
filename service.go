package champiris

import (
	"errors"

	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/mvc"
)

type API struct {
	app      *iris.Application
	config   NetConfig
	version  []string
	htmlPath string
}

func (api *API) Application() *iris.Application {
	return api.app
}

func (api *API) SetHtmlPath(htmlPath string) {
	api.htmlPath = htmlPath
}

func (api *API) SetVersion(version []string) {
	api.version = version
}

func (api *API) NewService(config NetConfig) error {
	api.app = iris.New()
	if len(api.version) == 0 {
		api.version = []string{"1"}
	}
	if len(api.htmlPath) == 0 {
		api.htmlPath = "./https/web"
	}
	if config.Port == "" {
		return errors.New("network port not set")
	}
	api.config = config
	requestLog, loggerClose := api.newRequestLogger()
	api.app.Use(requestLog)
	api.app.OnAnyErrorCode(requestLog, func(ctx iris.Context) {
		ctx.Values().Set("logger_message", "a dynamic message passed to the logs")
	})
	iris.RegisterOnInterrupt(func() {
		if err := loggerClose(); err != nil {
			api.app.Logger().Fatal(err)
		}
	})
	api.addHtmlDirectory(api.htmlPath)
	api.setApiVersion(api.version)
	return nil
}

func (api *API) Run(addSchema func(*iris.Application)) error {
	err := api.app.Run(
		iris.Addr(":"+api.config.Port),
		iris.WithOptimizations,
		iris.WithoutServerError(iris.ErrServerClosed),
		addSchema,
	)
	return err
}

func (api *API) addHtmlDirectory(path string) {
	api.app.RegisterView(iris.HTML(path, ".html").Reload(true))
}

func (api *API) setApiVersion(v []string) {
	for _, version := range v {
		mvc.Configure(api.app.Party("/api/v"+version), Routes)
	}
}
