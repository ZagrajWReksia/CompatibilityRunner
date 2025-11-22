package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"time"
)

type crashReportInfo struct {
	DDrawVariant string   `json:"ddraw"`
	Game         string   `json:"Game"`
	Platform     string   `json:"platform"`
	SkippedFiles []string `json:"skippedFiles,omitempty"`
}

func sendCrashes(detectedGame Game, variant DDrawVariant, crashesFilenames []string) {
	fmt.Println("=== Crash detected ===")
	fmt.Println("Game path:", detectedGame.binary)
	fmt.Println("Variant filename:", variant.filename)
	fmt.Println("Crash filenames:", crashesFilenames)

	report := crashReportInfo{
		DDrawVariant: variant.filename,
		Game:         detectedGame.binary,
		Platform:     runtime.GOOS,
	}

	if err := packCrash(detectedGame, report, crashesFilenames); err != nil {
		fmt.Println(err)
	}
}

func packCrash(detectedGame Game, reportInfo crashReportInfo, crashesFilenames []string) error {
	archive, err := os.Create(path.Join(crashLogsPath, fmt.Sprintf("crashlog_%d.zip", time.Now().Nanosecond())))
	if err != nil {
		return err
	}
	defer archive.Close()

	zipWriter := zip.NewWriter(archive)
	var skippedFiles []string

	for _, crashFilename := range crashesFilenames {
		var crashPath = path.Join(crashLogsPath, crashFilename)
		if err = addFileToZip(zipWriter, crashPath, crashFilename); err != nil {
			fmt.Printf("Skipped file %s", crashPath)
			skippedFiles = append(skippedFiles, crashPath)
		}
		_ = os.Remove(path.Join(crashLogsPath, crashFilename))
	}

	for _, filename := range detectedGame.saveFiles {
		matches, err := filepath.Glob(filename)
		if err != nil {
			continue
		}

		for _, resolvedFilename := range matches {
			var targetPath = path.Join("saves", resolvedFilename)
			if err = addFileToZip(zipWriter, resolvedFilename, targetPath); err != nil {
				fmt.Printf("Skipped file %s", resolvedFilename)
				skippedFiles = append(skippedFiles, resolvedFilename)
			}
		}
	}

	reportInfo.SkippedFiles = skippedFiles
	reportJson, err := json.MarshalIndent(reportInfo, "", "  ")
	if err != nil {
		return err
	}

	reportWriter, err := zipWriter.Create("report.json")
	if err != nil {
		return err
	}
	if _, err = reportWriter.Write(reportJson); err != nil {
		return err
	}

	if err = zipWriter.Close(); err != nil {
		return err
	}

	return nil
}

func addFileToZip(zipWriter *zip.Writer, src string, dst string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	writer, err := zipWriter.Create(dst)
	if err != nil {
		return err
	}
	if _, err = io.Copy(writer, file); err != nil {
		return err
	}

	return file.Close()
}
