// Copyright 2019 marmei. All rights reserved.
// Use of this source code is governed by a MIT license that can be found in the
// LICENSE file.
// GitHub: https://github.com/xteve-project/xTeVe

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"xteve/src"
)

// GitHubStruct holds the GitHub repository coordinates used by the self-updater
// to check for and download new releases. Branch, User, and Repo together
// identify the GitHub repository; Update enables or disables the update check.
type GitHubStruct struct {
	Branch string
	Repo   string
	Update bool
	User   string
}

// GitHub is the GitHub repository reference for update checks and binary
// downloads. User and Repo point to the fork that publishes CI release assets.
// Set Update to false to disable the self-updater entirely.
var GitHub = GitHubStruct{Branch: "master", User: "whisper", Repo: "xTeVe-dietpi", Update: true}

// Name is the human-readable application name used in log output, UI labels,
// and the HDHomeRun device advertisement.
const Name = "xTeVe"

// Version is the full application version string in the form
// "MAJOR.MINOR.PATCH.BUILD". The build number (last segment) is separated at
// startup and stored in src.System.Build. Declared as var so the Go linker's
// -X flag can inject the version from a git tag at compile time (e.g., via
// make build or the CI workflow).
var Version = "2.2.0.0200"

// DBVersion is the settings database schema version understood by this build.
// Settings files with a higher version will be rejected; files with a lower
// (but compatible) version are migrated automatically.
const DBVersion = "2.1.0"

// APIVersion is the version of the xTeVe HTTP API exposed to Plex, Emby, and
// third-party integrations.
const APIVersion = "1.1.0"

var homeDirectory = fmt.Sprintf("%s%s.%s%s", src.GetUserHomeDirectory(), string(os.PathSeparator), strings.ToLower(Name), string(os.PathSeparator))
var samplePath = fmt.Sprintf("%spath%sto%sxteve%s", string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator))
var sampleRestore = fmt.Sprintf("%spath%sto%sfile%s", string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator), string(os.PathSeparator))

var configFolder = flag.String("config", "", ": Config Folder        ["+samplePath+"] (default: "+homeDirectory+")")
var port = flag.String("port", "", ": Server port          [34400] (default: 34400)")
var restore = flag.String("restore", "", ": Restore from backup  ["+sampleRestore+"xteve_backup.zip]")

var gitBranch = flag.String("branch", "", ": Git Branch           [master|beta] (default: master)")
var debug = flag.Int("debug", 0, ": Debug level          [0 - 3] (default: 0)")
var info = flag.Bool("info", false, ": Show system info")
var h = flag.Bool("h", false, ": Show help")

// Aktiviert den Entwicklungsmodus. Für den Webserver werden dann die lokalen Dateien verwendet.
var dev = flag.Bool("dev", false, ": Activates the developer mode, the source code must be available. The local files for the web interface are used.")

func main() {

	// Build-Nummer von der Versionsnummer trennen
	var build = strings.Split(Version, ".")

	var system = &src.System
	system.APIVersion = APIVersion
	system.Branch = GitHub.Branch
	system.Build = build[len(build)-1:][0]
	system.DBVersion = DBVersion
	system.GitHub = GitHub
	system.Name = Name
	system.Version = strings.Join(build[0:len(build)-1], ".")

	// Panic !!!
	defer func() {

		if r := recover(); r != nil {

			fmt.Println()
			fmt.Println("* * * * * FATAL ERROR * * * * *")
			fmt.Println("OS:  ", runtime.GOOS)
			fmt.Println("Arch:", runtime.GOARCH)
			fmt.Println("Err: ", r)
			fmt.Println()

			pc := make([]uintptr, 20)
			runtime.Callers(2, pc)

			for i := range pc {

				if runtime.FuncForPC(pc[i]) != nil {

					f := runtime.FuncForPC(pc[i])
					file, line := f.FileLine(pc[i])

					if string(file)[0:1] != "?" {
						fmt.Printf("%s:%d %s\n", filepath.Base(file), line, f.Name())
					}

				}

			}

			fmt.Println()
			fmt.Println("* * * * * * * * * * * * * * * *")

		}

	}()

	flag.Parse()

	if *h {
		flag.Usage()
		return
	}

	system.Dev = *dev

	// Systeminformationen anzeigen
	if *info {

		system.Flag.Info = true

		err := src.Init()
		if err != nil {
			src.ShowError(err, 0)
			os.Exit(0)
		}

		src.ShowSystemInfo()
		return

	}

	// Webserver Port
	if len(*port) > 0 {
		system.Flag.Port = *port
	}

	// Branch
	system.Flag.Branch = *gitBranch
	if len(system.Flag.Branch) > 0 {
		fmt.Println("Git Branch is now:", system.Flag.Branch)
	}

	// Debug Level
	system.Flag.Debug = *debug
	if system.Flag.Debug > 3 {
		flag.Usage()
		return
	}

	// Speicherort für die Konfigurationsdateien
	if len(*configFolder) > 0 {
		system.Folder.Config = *configFolder
	}

	// Backup wiederherstellen
	if len(*restore) > 0 {

		system.Flag.Restore = *restore

		err := src.Init()
		if err != nil {
			src.ShowError(err, 0)
			os.Exit(0)
		}

		err = src.XteveRestoreFromCLI(*restore)
		if err != nil {
			src.ShowError(err, 0)
		}

		os.Exit(0)
	}

	err := src.Init()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	err = src.BinaryUpdate()
	if err != nil {
		src.ShowError(err, 0)
	}

	err = src.StartSystem(false)
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	err = src.InitMaintenance()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

	err = src.StartWebserver()
	if err != nil {
		src.ShowError(err, 0)
		os.Exit(0)
	}

}
