package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FileInfoWithDir はFileInfoとディレクトリパスを保持する為の構造体
type FileInfoWithDir struct {
	DirPath string
	FileInfo os.FileInfo
}

// mainは指定されたディレクトリ内のファイル及びディレクトリを内包したzipファイルを
// 指定ディレクトリと同階層に作成します。
func main() {
	flag.Parse()
	args := flag.Args() 
	if len(args) != 1 {
		onError("need to set a directory path")
	}
	// 有効なパスでかつディレクトリパスか判定
	dInfo, err := os.Stat(args[0])
	if err != nil {
		onError(fmt.Sprintf("%v", err.Error()))
	}
	if !dInfo.IsDir() {
		onError(fmt.Sprintf("%v is not directory", err.Error()))
	}

	dirPath := args[0]
	dstFilePath := strings.Join([]string{dirPath, "zip"}, ".")

	fInfoArr, err := ioutil.ReadDir(dirPath)
	if err != nil {
		onError(fmt.Sprintf("%v", err.Error()))
	}
	if len(fInfoArr) < 1 {
		onError(fmt.Sprintf("%v is empty directory", dirPath))
	}
	fInfoWithDirArr := combineDirPathAndFileInfo(dirPath, fInfoArr)

	if err := zipFiles(dstFilePath, fInfoWithDirArr, dInfo.Name()); err != nil {
		onError(fmt.Sprintf("%v", err.Error()))
	}
	fmt.Printf("saved as %v\n", dstFilePath)
}

// onErrorはエラーメッセージを出力し、exitします。
func onError(errMsg string) {
	fmt.Fprintln(os.Stderr, errMsg)
	os.Exit(1)
}

// getFileListInDirは指定ディレクトリ内のファイル及びディレクトリの名称と
// 指定ディレクトリのパスをまとめたFileInfoWithDir構造体の配列を返します。
func getFileListInDir(dirPath string) ([]FileInfoWithDir, error) {
	fInfoArr, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	fInfoWithDirArr := combineDirPathAndFileInfo(dirPath, fInfoArr)

	return fInfoWithDirArr, nil
}

// combineDirPathAndFileInfoは指定されたディレクトリのパスとos.FileInfoをまとめた
// FileInfoWithDir構造体の配列を返します。
func combineDirPathAndFileInfo(dirPath string, fileInfos []os.FileInfo) []FileInfoWithDir {
	var fInfoWithDirArr []FileInfoWithDir
	for _, fileInfo := range fileInfos {
		fInfoWithDirArr = append(fInfoWithDirArr, FileInfoWithDir{DirPath: dirPath, FileInfo: fileInfo})
	}

	return fInfoWithDirArr
}

// zipFilesは指定ディレクトリ以下のファイル及びディレクトリをまとめたzipファイルを作成します。
func zipFiles(dstName string, fInfoWithDirArr []FileInfoWithDir, baseInZip string) error {
	zipFile, err := os.Create(dstName)
	if err != nil {
	   return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, fInfoWithDir := range fInfoWithDirArr {
		addFileToZip(zipWriter, fInfoWithDir, baseInZip)
	}

	return nil
}

// addFileToZipはFileInfoWithDir構造体の配列を受け取り、zip.Writerを使ってzipファイルにファイルを追加します。
func addFileToZip(zipWriter *zip.Writer, fInfoWithDir FileInfoWithDir, baseInZip string) error {
	dstPath := filepath.Join(fInfoWithDir.DirPath, fInfoWithDir.FileInfo.Name())
	file, err := os.Open(dstPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	// ディレクトリが内包されている場合はbaseInZip(=zipファイル内でのカレントディレクトリ)を更新しaddFileToZipを呼び出す
	if info.IsDir() {
		childFInfoWithDirArr, err := getFileListInDir(dstPath)
		if err != nil {
			return err
		}
		for _, childFInfoWithDir := range childFInfoWithDirArr {
			addFileToZip(zipWriter, childFInfoWithDir, strings.Join([]string{baseInZip, info.Name()}, "/"))
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil
		}
		// zipファイル内のファイルパス: 相対パスにする必要がある
		header.Name = filepath.Join(baseInZip, fInfoWithDir.FileInfo.Name())
		// 圧縮率: http://golang.org/pkg/archive/zip/#pkg-constants
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		if _, err = io.Copy(writer, file); err != nil {
			return err
		}
	}

	return nil
}
