package cmd

import (
	"fmt"
	"github.com/annchain/OG/arefactor/common/format"
	"github.com/annchain/OG/arefactor/common/httplib"
	"github.com/annchain/OG/arefactor/common/io"
	"github.com/annchain/OG/arefactor/common/utilfuncs"
	"github.com/spf13/viper"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// readConfig will respect {configdir}/config.toml first.
// If not found, get config from online source {configurl}
// {configdir}/injected.toml is the config issued by bootstrap server.
// finally merge env config so that any config can be override by env variables.
// Importance order:
// 1, ENV
// 2, injected.toml
// 3, config.toml or online toml if config.toml is not found
func readConfig() {
	configPath := io.FixPrefixPath(viper.GetString("configdir"), "config.toml")

	if io.FileExists(configPath) {
		mergeLocalConfig(configPath)
	} else {
		mergeOnlineConfig(viper.GetString("configurl"))
	}

	// load injected config from ogbootstrap if any
	injectedPath := io.FixPrefixPath(viper.GetString("configdir"), "injected.toml")
	if io.FileExists(injectedPath) {
		mergeLocalConfig(injectedPath)
	}

	mergeEnvConfig()
	// print running config in console.
	b, err := format.PrettyJson(viper.AllSettings())
	utilfuncs.PanicIfError(err, "dump json")
	fmt.Println(b)
}

func mergeEnvConfig() {
	// env override
	viper.SetEnvPrefix("og")
	viper.AutomaticEnv()
}

func writeConfig() {
	configPath := viper.GetString("config")
	if strings.HasSuffix(configPath, ".toml") {
		viper.WriteConfigAs(configPath)
	}
}

func mergeOnlineConfig(configPath string) {
	_, err := url.Parse(configPath)
	if err != nil {
		utilfuncs.PanicIfError(err, "config is should be valid server url or toml file has suffix .toml")
	}
	fileName := "og_config_" + time.Now().Format("20060102_150405") + ".toml"
	fmt.Println("read from config", configPath)
	req := httplib.NewBeegoRequest(configPath, "GET")
	req.Debug(true)
	req.SetTimeout(60*time.Second, 60*time.Second)
	err = req.ToFile(fileName)
	if err != nil {
		_ = os.Remove(fileName)
		fmt.Println(req.String())
	}
	utilfuncs.PanicIfError(err, "get config from server error")

	file, err := os.Open(fileName)
	if err != nil {
		_ = os.Remove(fileName)
	}
	utilfuncs.PanicIfError(err, fmt.Sprintf("Error on opening config file: %s", fileName))
	defer file.Close()

	viper.SetConfigType("toml")
	err = viper.MergeConfig(file)
	_ = os.Remove(fileName)
	utilfuncs.PanicIfError(err, fmt.Sprintf("Error on reading config file: %s", fileName))
}

func mergeLocalConfig(configPath string) {
	absPath, err := filepath.Abs(configPath)
	utilfuncs.PanicIfError(err, fmt.Sprintf("Error on parsing config file path: %s", absPath))

	file, err := os.Open(absPath)
	utilfuncs.PanicIfError(err, fmt.Sprintf("Error on opening config file: %s", absPath))
	defer file.Close()

	viper.SetConfigType("toml")
	err = viper.MergeConfig(file)
	utilfuncs.PanicIfError(err, fmt.Sprintf("Error on reading config file: %s", absPath))
	return
}
