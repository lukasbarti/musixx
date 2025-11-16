package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

const (
	keyHTTPPort     = "http.port"
	keyMusicDir     = "library.music_dir"
	keyDatabasePath = "storage.database_path"
	keyYTDLPPath    = "tools.ytdlp_path"

	defaultHTTPPort    = 8080
	defaultMusicDir    = "music"
	defaultDatabase    = "musixx.db"
	defaultYTDLPBinary = "yt-dlp"
)

// Load configures Viper with defaults, files, and environment overrides.
func Load() error {
	setDefaults()

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.config/musixx")
	viper.AddConfigPath("/etc/musixx")

	viper.SetEnvPrefix("musixx")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		var notFound viper.ConfigFileNotFoundError
		if !errors.As(err, &notFound) {
			return fmt.Errorf("load config: %w", err)
		}
	}

	return nil
}

// HTTPPort returns the configured HTTP listen port.
func HTTPPort() int {
	return viper.GetInt(keyHTTPPort)
}

// MusicDir returns the path where Opus files are stored.
func MusicDir() string {
	return viper.GetString(keyMusicDir)
}

// DatabasePath returns the SQLite database file location.
func DatabasePath() string {
	return viper.GetString(keyDatabasePath)
}

// YTDLPPath returns the path to the yt-dlp executable.
func YTDLPPath() string {
	return viper.GetString(keyYTDLPPath)
}

func setDefaults() {
	viper.SetDefault(keyHTTPPort, defaultHTTPPort)
	viper.SetDefault(keyMusicDir, defaultMusicDir)
	viper.SetDefault(keyDatabasePath, defaultDatabase)
	viper.SetDefault(keyYTDLPPath, defaultYTDLPBinary)
}
