package config

import (
	"go.mau.fi/whatsmeow/proto/waCompanionReg"
)

var (
	AppVersion             = "v7.4.1"
	AppPort                = "3000"
	AppDebug               = false
	AppOs                  = "AldinoKemal"
	AppPlatform            = waCompanionReg.DeviceProps_PlatformType(1)
	AppBasicAuthCredential []string
	AppBasePath            = ""

	McpPort = "8080"
	McpHost = "localhost"

	PathQrCode    = "statics/qrcode"
	PathSendItems = "statics/senditems"
	PathMedia     = "statics/media"
	PathStorages  = "storages"

	DBURI     = "postgres://postgres:password@localhost:5432/whatsapp_db?sslmode=disable"
	DBKeysURI = "postgres://postgres:password@localhost:5432/whatsapp_keys?sslmode=disable"

	WhatsappAutoReplyMessage       string
	WhatsappAutoMarkRead           = false // Auto-mark incoming messages as read
	WhatsappWebhook                []string
	WhatsappWebhookSecret                = "secret"
	WhatsappLogLevel                     = "ERROR"
	WhatsappSettingMaxImageSize    int64 = 20000000  // 20MB
	WhatsappSettingMaxFileSize     int64 = 50000000  // 50MB
	WhatsappSettingMaxVideoSize    int64 = 100000000 // 100MB
	WhatsappSettingMaxDownloadSize int64 = 500000000 // 500MB
	WhatsappTypeUser                     = "@s.whatsapp.net"
	WhatsappTypeGroup                    = "@g.us"
	WhatsappAccountValidation            = true

	ChatStorageURI               = "postgres://postgres:password@localhost:5432/whatsapp_chat?sslmode=disable"
	ChatStorageEnableForeignKeys = true
	ChatStorageEnableWAL         = true
	ChatStorageType              = "postgres" // "sqlite" or "postgres"

	// Session Persistence Settings
	SessionBackupEnabled       = true
	SessionBackupInterval      = 300 // seconds (5 minutes)
	SessionBackupRetention     = 7   // days
	SessionAutoRestore         = true
	SessionHealthCheckInterval = 60 // seconds

	// Redis Cache Settings
	RedisEnabled  = true
	RedisHost     = "modern-mantis-13814.upstash.io"
	RedisPort     = 6379
	RedisPassword = "ATX2AAIjcDEyYzY5OGExZGE3Njc0NTJlODk2MDgxYmI3YzE3YTE3ZnAxMA"
	RedisDB       = 0
	RedisPrefix   = "whatsapp"
	RedisURL      = "rediss://:ATX2AAIjcDEyYzY5OGExZGE3Njc0NTJlODk2MDgxYmI3YzE3YTE3ZnAxMA@modern-mantis-13814.upstash.io:6379" // Upstash Redis URL (rediss:// for SSL)

	// Cloud Backup Settings (Backblaze B2 by default)
	BackupEnabled         = true
	BackupProvider        = "b2" // "b2" for Backblaze B2, "gcs" for Google Cloud Storage
	BackupBucket          = "whatsapp-backups"
	BackupRegion          = "us-east-1"
	BackupKeyID           = "" // Set via environment variable B2_KEY_ID
	BackupApplicationKey  = "" // Set via environment variable B2_APPLICATION_KEY
	BackupPrefix          = "whatsapp-backups"
	BackupRetentionDays   = 30
	BackupScheduleEnabled = true
	BackupScheduleCron    = "0 2 * * *" // Daily at 2 AM
)
