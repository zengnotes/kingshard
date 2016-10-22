// Copyright 2016 The kingshard Authors. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"

	"config"
	"core/golog"
	//"core/hack"
	"proxy/server"
	"web"
	"io/ioutil"
)

//var configFile *string = flag.String("c", "", "kingshard config file")
var configDir *string = flag.String("d", "etc", "kingshard config files dir ")
var logLevel *string = flag.String("l", "warn", "log level [debug|info|warn|error], default error")
var version *bool = flag.Bool("v", false, "the version of kingshard")

const (
	sqlLogName = "sql.log"
	sysLogName = "sys.log"
	MaxLogSize = 1024 * 1024 * 1024
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	flag.Parse()
	//fmt.Print(banner)
	//fmt.Printf("Git commit:%s\n", hack.Version)
	//fmt.Printf("Build time:%s\n", hack.Compile)
	if *version {
		return
	}
	if len(*configDir) == 0 {
		fmt.Println("must use a config file")
		return
	}
	//文件解析

	configFiles, err := ListDir(*configDir, ".yaml")
	if err != nil || len(configFiles) == 0 {
		fmt.Println("must use a config file in ", *configDir)
		return
	}
	for _, v := range configFiles {
		start(v)
	}
	fmt.Println("start ok!")
	//退出
	sigChan := make(chan os.Signal, 3)
	signal.Notify(sigChan, os.Interrupt, os.Kill)
	<-sigChan
}

func start(configFile string) {

	//文件解析

	cfg, err := config.ParseConfigFile(configFile)
	if err != nil {
		fmt.Printf("parse config file error:%v\n", err.Error())
		return
	}

	//when the log file size greater than 1GB, kingshard will generate a new file
	if len(cfg.LogPath) != 0 {
		sysFilePath := path.Join(cfg.LogPath, sysLogName)
		sysFile, err := golog.NewRotatingFileHandler(sysFilePath, MaxLogSize, 1)
		if err != nil {
			fmt.Printf("new log file error:%v\n", err.Error())
			return
		}
		golog.GlobalSysLogger = golog.New(sysFile, golog.Lfile | golog.Ltime | golog.Llevel)

		sqlFilePath := path.Join(cfg.LogPath, sqlLogName)
		sqlFile, err := golog.NewRotatingFileHandler(sqlFilePath, MaxLogSize, 1)
		if err != nil {
			fmt.Printf("new log file error:%v\n", err.Error())
			return
		}
		golog.GlobalSqlLogger = golog.New(sqlFile, golog.Lfile | golog.Ltime | golog.Llevel)
	}

	if *logLevel != "" {
		setLogLevel(*logLevel)
	} else {
		setLogLevel(cfg.LogLevel)
	}

	var svr *server.Server
	var apiSvr *web.ApiServer
	svr, err = server.NewServer(cfg)
	if err != nil {
		golog.Error("main", "main", err.Error(), 0)
		golog.GlobalSysLogger.Close()
		golog.GlobalSqlLogger.Close()
		return
	}
	apiSvr, err = web.NewApiServer(cfg, svr)
	if err != nil {
		golog.Error("main", "main", err.Error(), 0)
		golog.GlobalSysLogger.Close()
		golog.GlobalSqlLogger.Close()
		svr.Close()
		return
	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGPIPE,
	)

	go func() {
		for {
			sig := <-sc
			if sig == syscall.SIGINT || sig == syscall.SIGTERM || sig == syscall.SIGQUIT {
				golog.Info("main", "main", "Got signal", 0, "signal", sig)
				golog.GlobalSysLogger.Close()
				golog.GlobalSqlLogger.Close()
				svr.Close()
				os.Remove(cfg.Addr)
			} else if sig == syscall.SIGPIPE {
				golog.Info("main", "main", "Ignore broken pipe signal", 0)
			}
		}
	}()
	go apiSvr.Run()
	go svr.Run()
}

func setLogLevel(level string) {
	switch strings.ToLower(level) {
	case "debug":
		golog.GlobalSysLogger.SetLevel(golog.LevelDebug)
	case "info":
		golog.GlobalSysLogger.SetLevel(golog.LevelInfo)
	case "warn":
		golog.GlobalSysLogger.SetLevel(golog.LevelWarn)
	case "error":
		golog.GlobalSysLogger.SetLevel(golog.LevelError)
	default:
		golog.GlobalSysLogger.SetLevel(golog.LevelError)
	}
}

func ListDir(dirPth string, suffix string) (files []string, err error) {
	files = make([]string, 0, 10)
	dir, err := ioutil.ReadDir(dirPth)
	if err != nil {
		return nil, err
	}
	PthSep := string(os.PathSeparator)
	suffix = strings.ToUpper(suffix) //忽略后缀匹配的大小写
	for _, fi := range dir {
		if fi.IsDir() {
			// 忽略目录
			continue
		}
		if strings.HasSuffix(strings.ToUpper(fi.Name()), suffix) {
			//匹配文件
			files = append(files, dirPth + PthSep + fi.Name())
		}
	}
	return files, nil
}
