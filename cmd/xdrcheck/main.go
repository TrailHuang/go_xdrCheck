package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/user/go_xdrCheck/internal/config"
	"github.com/user/go_xdrCheck/internal/core"

	_ "net/http/pprof"

	"github.com/spf13/pflag"
)

var (
	version   = "2025_v1.1.18"
	buildTime = "unknown"
	svnNo     = "unknown"
	timeParam string
	scanNum   int
	checkNow  bool
	noSubPath bool
	workerNum int
)

func main() {
	// 解析命令行参数
	pflag.StringVarP(&timeParam, "time", "t", time.Now().Format("20060102"), "xdr time")
	pflag.IntVarP(&scanNum, "num", "n", 0, "scan num per dir")
	pflag.BoolVarP(&checkNow, "now", "o", false, "now time")
	pflag.BoolVarP(&noSubPath, "nosubpath", "p", false, "do not check sub path")
	pflag.IntVarP(&workerNum, "routines", "r", 4, "number of worker routines (default: 4)")
	pflag.BoolP("help", "h", false, "help info")
	pflag.BoolP("version", "v", false, "version info")

	pflag.Parse()

	// 处理帮助和版本信息
	if pflag.Lookup("help").Value.String() == "true" {
		printHelp()
		return
	}

	if pflag.Lookup("version").Value.String() == "true" {
		printVersion()
		return
	}

	// 处理时间参数
	if checkNow {
		timeParam = time.Now().Format("20060102")
	}

	// 如果没有指定时间参数，使用当前时间
	if timeParam == "" {
		timeParam = time.Now().Format("20060102")
	}

	if noSubPath {
		timeParam = ""
	}

	// 创建临时目录
	tmpDir := filepath.Join("/tmp/xdr_check", time.Now().Format("20060102"))
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		fmt.Printf("创建临时目录失败: %v\n", err)
		return
	}

	// 清理旧的临时目录
	core.ClearOldTmpDirs("/tmp/xdr_check", 30)

	go func() {
		http.ListenAndServe("127.0.0.1:8899", nil)
	}()

	// 启动主程序
	if err := startCheck(); err != nil {
		fmt.Printf("检查失败: %v\n", err)
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("========================================================")
	fmt.Println("-h,\t--help\t\t\thelp info")
	fmt.Println("-v,\t--version\t\tversion info")
	fmt.Println("-t,\t--time\t\t\txdr time")
	fmt.Println("-o,\t--now\t\t\tnow time")
	fmt.Println("-n,\t--num\t\t\tscan num per dir")
	fmt.Println("-p,\t--nosubpath\t\tdo not check sub path")
	fmt.Println("-r,\t--routines\t\tnumber of worker routines (default: 4)")
	fmt.Println("========================================================")
}

func printVersion() {
	fmt.Printf("版本信息: %s\n", version)
	fmt.Printf("构建时间: %s\n", buildTime)
	fmt.Printf("svn版本: %s\n", svnNo)
}

func startCheck() error {
	// 加载配置文件
	configFile := config.GetConfigFile()
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置文件失败: %v", err)
	}

	// 创建检查器
	checker := core.NewXDRChecker(cfg, timeParam, scanNum, noSubPath, workerNum)

	// 开始检查
	return checker.StartCheck()
}
