package cmd

import (
	"github.com/annchain/OG/arefactor/core"
	"github.com/annchain/commongo/files"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"os"
	"path"
)

func ensureFolder(folder string, perm os.FileMode) {
	err := files.MkDirPermIfNotExists(folder, perm)
	if err != nil {
		logrus.WithError(err).WithField("path", folder).Fatal("failed to create folder")
	}
}

func defaultPath(givenPath string, defaultRoot string, suffix string) string {
	if givenPath == "" {
		return path.Join(defaultRoot, suffix)
	}
	if path.IsAbs(givenPath) {
		return givenPath
	}
	return path.Join(defaultRoot, givenPath)
}

func ensureFolders() core.FolderConfig {
	config := core.FolderConfig{
		Root:    viper.GetString("dir.root"),
		Log:     defaultPath(viper.GetString("dir.log"), viper.GetString("dir.root"), "log"),
		Data:    defaultPath(viper.GetString("dir.data"), viper.GetString("dir.root"), "data"),
		Config:  defaultPath(viper.GetString("dir.config"), viper.GetString("dir.root"), "config"),
		Private: defaultPath(viper.GetString("dir.private"), viper.GetString("dir.root"), "private"),
	}
	ensureFolder(config.Root, 0755)
	ensureFolder(config.Log, 0755)
	ensureFolder(config.Data, 0755)
	ensureFolder(config.Config, 0755)
	ensureFolder(config.Private, 0700)
	return config

}
