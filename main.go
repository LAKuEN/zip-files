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
	// FIXME リファクタリング
	// TODO 圧縮率の外部からの指定
	flag.Parse()
	args := flag.Args() 
	if len(args) != 1 {
		fmt.Fprintf(os.Stderr, "need to set a directory path\n")
		os.Exit(1)
	}
	// 有効なパスでかつディレクトリパスか判定
	dInfo, err := os.Stat(args[0])
	if err != nil {
		onError("%v\n", err)
	}
	if !dInfo.IsDir() {
		onError("%v is not directory\n", err)
	}
	dirName := dInfo.Name()
	dirPath := args[0]
	dstFilePath := strings.Join([]string{dirPath, "zip"}, ".")

	fInfoArr, err := ioutil.ReadDir(dirPath)
	fInfoWzDirArr := combineDirPathAndFileInfo(dirPath, fInfoArr)
	if err != nil {
		onError("%v\n", err)
	}
	if err := zipFiles(dstFilePath, fInfoWzDirArr, dirName); err != nil {
		onError("%v\n", err)
	}
	fmt.Printf("saved as %v\n", dstFilePath)
}

func onError(template string, err error) {
	fmt.Fprintf(os.Stderr, template, err)
// onErrorはエラーメッセージを出力し、exitします。
	os.Exit(1)
}

// getFileListInDirは指定ディレクトリ内のファイル及びディレクトリの名称と
// 指定ディレクトリのパスをまとめたFileInfoWithDir構造体の配列を返します。
func getFileListInDir(dirPath string) ([]FileInfoWithDir, error) {
	fInfoArr, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	fInfoWzDirArr := combineDirPathAndFileInfo(dirPath, fInfoArr)

	return fInfoWzDirArr, nil
}

// combineDirPathAndFileInfoは指定されたディレクトリのパスとos.FileInfoをまとめた
// FileInfoWithDir構造体の配列を返します。
func combineDirPathAndFileInfo(dirPath string, fileInfos []os.FileInfo) []FileInfoWithDir {
	var fInfoWzDirArr []FileInfoWithDir
	for _, fileInfo := range fileInfos {
		fInfoWzDirArr = append(fInfoWzDirArr, FileInfoWithDir{DirPath: dirPath, FileInfo: fileInfo})
	}

	return fInfoWzDirArr
}

// zipFilesは指定ディレクトリ以下のファイル及びディレクトリをまとめたzipファイルを作成します。
func zipFiles(dstName string, fInfoWzDirArr []FileInfoWithDir, baseInZip string) error {
	zipFile, err := os.Create(dstName)
	if err != nil {
	   return err
	}
	defer zipFile.Close()
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, fInfoWzDir := range fInfoWzDirArr {
		addFileToZip(zipWriter, fInfoWzDir, baseInZip)
	}

	return nil
}

// addFileToZipはFileInfoWithDir構造体の配列を受け取り、zip.Writerを使ってzipファイルにファイルを追加します。
func addFileToZip(zipWriter *zip.Writer, fInfoWzDir FileInfoWithDir, baseInZip string) error {
	absPath := filepath.Join(fInfoWzDir.DirPath, fInfoWzDir.FileInfo.Name())
	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		childFInfoWzDirArr, err := getFileListInDir(absPath)
		if err != nil {
			return err
		}
		for _, childFInfoWzDir := range childFInfoWzDirArr {
			addFileToZip(zipWriter, childFInfoWzDir, strings.Join([]string{baseInZip, info.Name()}, "/"))
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil
		}

		// 圧縮率の指定: http://golang.org/pkg/archive/zip/#pkg-constants
		// zipファイル内のファイルパス: 絶対パスを渡すと/homeからのファイルパスでファイルが登録されてしまう為、
		// 元のディレクトリのパスを除いたパスに変換する
		header.Name = filepath.Join(baseInZip, fInfoWzDir.FileInfo.Name())
		// 圧縮率
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
