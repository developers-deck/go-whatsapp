package cmd

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"

	"github.com/aldinokemal/go-whatsapp-web-multidevice/config"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/helpers"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/rest/middleware"
	"github.com/aldinokemal/go-whatsapp-web-multidevice/ui/websocket"
	"github.com/dustin/go-humanize"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/basicauth"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/filesystem"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var restCmd = &cobra.Command{
	Use:   "rest",
	Short: "Send whatsapp API over http",
	Long:  `This application is from clone https://github.com/aldinokemal/go-whatsapp-web-multidevice`,
	Run:   restServer,
}

func init() {
	rootCmd.AddCommand(restCmd)
}
func restServer(_ *cobra.Command, _ []string) {
	engine := html.NewFileSystem(http.FS(EmbedIndex), ".html")
	app := fiber.New(fiber.Config{
		Views:     engine,
		BodyLimit: int(config.WhatsappSettingMaxVideoSize),
		Network:   "tcp",
	})

	app.Static(config.AppBasePath+"/statics", "./statics")
	app.Use(config.AppBasePath+"/components", filesystem.New(filesystem.Config{
		Root:       http.FS(EmbedViews),
		PathPrefix: "views/components",
		Browse:     true,
	}))
	app.Use(config.AppBasePath+"/assets", filesystem.New(filesystem.Config{
		Root:       http.FS(EmbedViews),
		PathPrefix: "views/assets",
		Browse:     true,
	}))

	app.Use(middleware.Recovery())
	if config.AppDebug {
		app.Use(logger.New())
	}
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowHeaders: "Origin, Content-Type, Accept",
	}))

	if len(config.AppBasicAuthCredential) > 0 {
		account := make(map[string]string)
		for _, basicAuth := range config.AppBasicAuthCredential {
			ba := strings.Split(basicAuth, ":")
			if len(ba) != 2 {
				logrus.Fatalln("Basic auth is not valid, please this following format <user>:<secret>")
			}
			account[ba[0]] = ba[1]
		}

		app.Use(basicauth.New(basicauth.Config{
			Users:           account,
			ContextUsername: "_user",
			ContextPassword: "_pass",
		}))
	}

	// Create base path group or use app directly
	var apiGroup fiber.Router = app
	if config.AppBasePath != "" {
		apiGroup = app.Group(config.AppBasePath)
	}

	// Rest
	rest.InitRestApp(apiGroup, appUsecase)
	rest.InitRestChat(apiGroup, chatUsecase)
	rest.InitRestSend(apiGroup, sendUsecase)
	rest.InitRestUser(apiGroup, userUsecase)
	rest.InitRestMessage(apiGroup, messageUsecase)
	rest.InitRestGroup(apiGroup, groupUsecase)
	rest.InitRestNewsletter(apiGroup, newsletterUsecase)
	rest.InitRestFileManager(apiGroup)
	rest.InitRestMonitor(apiGroup)
	rest.InitRestTemplates(apiGroup)
	rest.InitRestQueue(apiGroup)
	rest.InitRestUpdater(apiGroup)
	rest.InitRestWebhook(apiGroup)
	rest.InitRestCache(apiGroup)
	rest.InitRestBackup(apiGroup)
	rest.InitRestSystem(apiGroup)
	analyticsHandler := rest.InitRestAnalytics(apiGroup)

	// Add analytics tracking middleware
	apiGroup.Use(analyticsHandler.TrackingMiddleware())

	apiGroup.Get("/", func(c *fiber.Ctx) error {
		// Get basic auth credentials if available
		var basicAuthToken string
		if username := c.Locals("_user"); username != nil {
			if password := c.Locals("_pass"); password != nil {
				// Create base64 encoded credentials for Authorization header
				credentials := fmt.Sprintf("%s:%s", username, password)
				basicAuthToken = "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
			}
		}

		// Always provide a valid string value to avoid template issues
		if basicAuthToken == "" {
			basicAuthToken = "none"
		}

		return c.Render("views/index", fiber.Map{
			"AppHost":        fmt.Sprintf("%s://%s", c.Protocol(), c.Hostname()),
			"AppVersion":     config.AppVersion,
			"AppBasePath":    config.AppBasePath,
			"BasicAuthToken": basicAuthToken,
			"MaxFileSize":    humanize.Bytes(uint64(config.WhatsappSettingMaxFileSize)),
			"MaxVideoSize":   humanize.Bytes(uint64(config.WhatsappSettingMaxFileSize)),
		})
	})

	websocket.RegisterRoutes(apiGroup, appUsecase)
	go websocket.RunHub()

	// Set auto reconnect to whatsapp server after booting
	go helpers.SetAutoConnectAfterBooting(appUsecase)
	// Set auto reconnect checking
	go helpers.SetAutoReconnectChecking(whatsappCli)

	if err := app.Listen(":" + config.AppPort); err != nil {
		logrus.Fatalln("Failed to start: ", err.Error())
	}
}
