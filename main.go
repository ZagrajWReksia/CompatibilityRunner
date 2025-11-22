package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/sqweek/dialog"
)

const crashLogsPath = "./compatibility/crashlogs/"
const procdumpPath = "./compatibility/procdump.exe"
const ddrawsPath = "./compatibility/ddraws/"
const exitCodeText = "Exit Code "

type Game struct {
	binary    string
	saveFiles []string
}

type DDrawVariant struct {
	filename  string
	platforms []string
}

var games = []Game{
	{
		binary:    "ReksioPiraci.exe",
		saveFiles: []string{"Piraci.ini", "DDrawCompat-*.log"},
	},
	{
		binary:    "ReksioUfo.exe",
		saveFiles: []string{"Ufo.ini", "DDrawCompat-*.log"},
	},
	{
		binary:    "Czarodzieje.exe",
		saveFiles: []string{"Czarodzieje.ini", "DDrawCompat-*.log", "common/*.dta", "common/*.arr"},
	},
	{
		binary:    "Wehikul.exe",
		saveFiles: []string{"Wehikul.ini", "DDrawCompat-*.log", "common/*.dta", "common/*.arr"},
	},
	{
		binary:    "Nemo.exe",
		saveFiles: []string{"Nemo.ini", "DDrawCompat-*.log", "common/save/*"},
	},
	{
		binary:    "Rex5.exe",
		saveFiles: []string{"rex5.ini", "DDrawCompat-*.log", "common/save/*", "common/save_bd/*"},
	},
}

var ddrawsOrder = []DDrawVariant{
	{
		filename:  "ddraw_compat.dll",
		platforms: []string{"windows"},
	},
	{
		filename:  "cnc_ddraw_experimental.dll",
		platforms: []string{"windows", "linux"},
	},
	{
		filename:  "cnc_ddraw_71.dll",
		platforms: []string{"windows", "linux"},
	},
}

func detectGame() (Game, error) {
	entries, err := os.ReadDir("./")
	if err != nil {
		return Game{}, err
	}

	for _, entry := range entries {
		for _, game := range games {
			if game.binary == entry.Name() {
				return game, nil
			}
		}
	}

	return Game{}, errors.New("no Game binary found")
}

func findNewCrashes(after time.Time) []string {
	fileEntries, err := os.ReadDir(crashLogsPath)
	if err != nil {
		return []string{}
	}

	var filenames []string
	for _, fileEntry := range fileEntries {
		info, err := fileEntry.Info()
		if err != nil {
			continue
		}

		if strings.HasSuffix(info.Name(), ".dmp") && info.ModTime().After(after) {
			filenames = append(filenames, fileEntry.Name())
		}
	}

	return filenames
}

func runWithProcDump(binaryPath string) error {
	_ = os.MkdirAll(crashLogsPath, os.ModePerm)

	cmd := exec.Command(procdumpPath, "-accepteula", "-e", "-h", "-x", crashLogsPath, binaryPath)
	cmd.Dir = "."
	cmd.SysProcAttr = getSysProcAttr()
	cmd.Env = append(os.Environ(), "__COMPAT_LAYER=WinXP,RUNASINVOKER,DisableWER", "WINEDLLOVERRIDES=ddraw=n,b")

	output, err := cmd.Output()
	if err != nil {
		return err
	}
	outputStr := string(output)

	exitCodePrefixPosition := strings.LastIndex(outputStr, exitCodeText)
	if exitCodePrefixPosition != -1 {
		exitCodeStart := exitCodePrefixPosition + len(exitCodeText) + 2
		exitCodeStr := outputStr[exitCodeStart : exitCodeStart+8]

		exitCodeInt, err := strconv.ParseInt(exitCodeStr, 16, 64)
		if err != nil {
			return err
		}

		if exitCodeInt != 0 {
			return errors.New("failed to run game")
		}
	}

	return nil
}

func runWithDDrawVariant(variant DDrawVariant, detectedGame Game) bool {
	fmt.Printf("Running with ddraw variant %s...\n", variant.filename)

	if err := copyFile(path.Join(ddrawsPath, variant.filename), "ddraw.dll"); err != nil {
		fmt.Println(err)
		return false
	}

	startTime := time.Now()
	err := runWithProcDump(detectedGame.binary)
	if err != nil {
		crashes := findNewCrashes(startTime)
		sendCrashes(detectedGame, variant, crashes)
	}

	// Hack
	exec.Command("taskkill", "/IM", detectedGame.binary, "/T", "/F").Run()

	return err == nil
}

func run(detectedGame Game) *DDrawVariant {
	for _, ddraw := range ddrawsOrder {
		if slices.Contains(ddraw.platforms, runtime.GOOS) {
			if runWithDDrawVariant(ddraw, detectedGame) {
				return &ddraw
			}
		}
	}
	return nil
}

func main() {
	detectedGame, err := detectGame()
	if err != nil {
		dialog.Message("Could not find the Game binary").Title("Running error").Error()
		log.Fatal(err)
	}

	workingVariant := run(detectedGame)
	if workingVariant == nil {
		dialog.Message("We tried to fix the Game but it keeps crashing").Title("Game keeps crashing").Error()
	}
}
