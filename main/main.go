package main

//go:generate go run $GOPATH/src/v2ray.com/core/common/errors/errorgen/main.go -pkg main -path Main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	
	//log support for pprof
	"log"

	"v2ray.com/core"
	"v2ray.com/core/common/platform"
	_ "v2ray.com/core/main/distro/all"
	
	// import pprof
	_ "net/http/pprof"
	
	// http server support for pprof
	"net/http"
)

var (
	configFile = flag.String("config", "", "Config file for V2Ray.")
	version    = flag.Bool("version", false, "Show current version of V2Ray.")
	test       = flag.Bool("test", false, "Test config file only, without launching V2Ray server.")
	format     = flag.String("format", "json", "Format of input file.")
	plugin     = flag.Bool("plugin", false, "True to load plugins.")
)

func fileExists(file string) bool {
	info, err := os.Stat(file)
	return err == nil && !info.IsDir()
}

func getConfigFilePath() string {
	if len(*configFile) > 0 {
		return *configFile
	}

	if workingDir, err := os.Getwd(); err == nil {
		configFile := filepath.Join(workingDir, "config.json")
		if fileExists(configFile) {
			return configFile
		}
	}

	if configFile := platform.GetConfigurationPath(); fileExists(configFile) {
		return configFile
	}

	return ""
}

func GetConfigFormat() core.ConfigFormat {
	switch strings.ToLower(*format) {
	case "json":
		return core.ConfigFormat_JSON
	case "pb", "protobuf":
		return core.ConfigFormat_Protobuf
	default:
		return core.ConfigFormat_JSON
	}
}

func startV2Ray() (core.Server, error) {
	configFile := getConfigFilePath()
	var configInput io.Reader
	if configFile == "stdin:" {
		configInput = os.Stdin
	} else {
		fixedFile := os.ExpandEnv(configFile)
		file, err := os.Open(fixedFile)
		if err != nil {
			return nil, newError("config file not readable").Base(err)
		}
		defer file.Close()
		configInput = file
	}
	config, err := core.LoadConfig(GetConfigFormat(), configInput)
	if err != nil {
		return nil, newError("failed to read config file: ", configFile).Base(err)
	}

	server, err := core.New(config)
	if err != nil {
		return nil, newError("failed to create server").Base(err)
	}

	return server, nil
}

func main() {
	flag.Parse()

	core.PrintVersion()

	if *version {
		return
	}

	if *plugin {
		if err := core.LoadPlugins(); err != nil {
			fmt.Println("Failed to load plugins:", err.Error())
			os.Exit(-1)
		}
	}

	server, err := startV2Ray()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(-1)
	}

	if *test {
		fmt.Println("Configuration OK.")
		os.Exit(0)
	}
	
	// start a http server before start V2Ray
	go func(){log.Println(http.ListenAndServe("localhost:6060", nil))}()
	
	if err := server.Start(); err != nil {
		fmt.Println("Failed to start", err)
		os.Exit(-1)
	}

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, os.Kill, syscall.SIGTERM)

	<-osSignals
	server.Close()
}
