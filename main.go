package main

import (
	"github.com/pocketbase/pocketbase/cmd"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"

	"pocketbase/auditlog"
	hooks "pocketbase/hooks"

	"github.com/labstack/echo/v5"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/jsvm"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
)

func defaultPublicDir() string {
	if strings.HasPrefix(os.Args[0], os.TempDir()) {
		// most likely ran with go run
		return "./pb_public"
	}

	return filepath.Join(os.Args[0], "../pb_public")
}

func main() {
	app := pocketbase.New()

	var publicDirFlag string

	// add "--publicDir" option flag
	app.RootCmd.PersistentFlags().StringVar(
		&publicDirFlag,
		"publicDir",
		defaultPublicDir(),
		"the directory to serve static files",
	)

	// load js files to allow loading external JavaScript migrations
	jsvm.MustRegister(app, jsvm.Config{
		HooksWatch: true, // make this false for production
	})

	// register the `migrate` command
	migratecmd.MustRegister(app, app.RootCmd, migratecmd.Config{
		TemplateLang: migratecmd.TemplateLangJS, // or migratecmd.TemplateLangGo (default)
		Automigrate:  true,
	})

	// call this only if you want to auditlog tables named in AUDITLOG env var
	auditlog.Register(app)

	// call this only if you want to use the configurable "hooks" functionality
	hooks.PocketBaseInit(app)

	app.OnBeforeServe().Add(func(e *core.ServeEvent) error {
		// serves static files from the provided public dir (if exists)
		e.Router.GET("/*", apis.StaticDirectoryHandler(os.DirFS(publicDirFlag), true))

		e.Router.AddRoute(echo.Route{
			Method: http.MethodGet,
			Path:   "/api/hello",
			Handler: func(c echo.Context) error {
				obj := map[string]interface{}{"message": "Hello world!"}
				return c.JSON(http.StatusOK, obj)
			},
			// Middlewares: []echo.MiddlewareFunc{
			// 	apis.RequireAdminOrUserAuth(),
			// },
		})

		return nil
	})

	if err := app.Bootstrap(); err != nil {
		log.Fatal(err)
	}
	gratefulStartApp(app)
}

func gratefulStartApp(app *pocketbase.PocketBase) error {
	// clear all args, if we want to customize pocketbase with any args, it should be written in code
	cmdServe := cmd.NewServeCommand(app, true)
	os.Args = os.Args[:1]

	var wg sync.WaitGroup

	wg.Add(1)

	// wait for interrupt signal to gracefully shutdown the application
	go func() {
		defer wg.Done()
		quit := make(chan os.Signal, 1) // we need to reserve to buffer size 1, so the notifier are not blocked
		signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
		<-quit
	}()

	go func() {
		defer wg.Done()
		if err := cmdServe.Execute(); err != nil {
			log.Println(err)
		}
	}()

	wg.Wait()

	//TODO: cleanup
	return nil
}