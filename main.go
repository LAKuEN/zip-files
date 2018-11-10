package zipfiles

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// FileInfoWithDir はFileInfoとディレクトリパスを保持する為の構造体
type FileInfoWithDir struct {
	DirPath  string
	FileInfo os.FileInfo
}

// InDir は指定されたディレクトリの構造を保ったままzipファイル化します。
func InDir(dirPath string) (string, error) {
	// 有効なパスでかつディレクトリパスか判定
	dInfo, err := os.Stat(dirPath)
	if err != nil {
		return "", err
	}
	if !dInfo.IsDir() {
		return "", fmt.Errorf("%v is not directory", err.Error())
	}

	dstFilePath := strings.Join([]string{dirPath, "zip"}, ".")

	fInfoArr, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return "", err
	}
	if len(fInfoArr) < 1 {
		return "", fmt.Errorf("%v is empty directory", dirPath)
	}
	fInfoWithDirArr := combineDirPathAndFileInfo(dirPath, fInfoArr)

	if err := zipFiles(dstFilePath, fInfoWithDirArr, dInfo.Name()); err != nil {
		return "", err
	}

	return dstFilePath, nil
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
		fInfoWithDirArr = append(fInfoWithDirArr,
			FileInfoWithDir{DirPath: dirPath, FileInfo: fileInfo})
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
// baseInZipはzipファイル内でのカレントディレクトリを指します。
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
	// 子ディレクトリであればbaseInZipを更新しaddFileToZipを呼び出す
	if info.IsDir() {
		childFInfoWithDirArr, err := getFileListInDir(dstPath)
		if err != nil {
			return err
		}
		for _, childFInfoWithDir := range childFInfoWithDirArr {
			addFileToZip(zipWriter, childFInfoWithDir,
				strings.Join([]string{baseInZip, info.Name()}, "/"))
		}
	} else {
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return nil
		}
		// zip内でのファイルの相対パス
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
