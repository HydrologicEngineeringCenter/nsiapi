package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	s "strings"
)

type GisFileType int

const (
	Shapefile GisFileType = iota
	Geopackage
	FileGeodatabase
)

func GetGdalDriverName(gisFileType GisFileType) (string, error) {
	switch gisFileType {
	case Shapefile:
		return "ESRI Shapefile", nil
	case Geopackage:
		return "GPKG", nil
	}
	return "", errors.New("Invalid Gis File Type")
}

func GetGisFileType(files []string) (GisFileType, string, error) {
	for _, f := range files {
		ext := filepath.Ext(s.ToLower(f))
		fmt.Printf("%s >>> %s\n", f, ext)
		if ext == ".shp" {
			return Shapefile, f, nil
		} else if ext == ".gpkg" {
			return Geopackage, f, nil
		}
	}
	return -1, "", errors.New("Invalid geospatial upload.")
}

func CopyPostFileToTemp(tempstorage string, tempname string, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Destination

	err = os.Mkdir(tempstorage+tempname, os.ModePerm)
	if err != nil {
		return "", err
	}

	newfile := tempstorage + tempname + "/" + file.Filename

	dst, err := os.Create(newfile)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// Copy
	if _, err = io.Copy(dst, src); err != nil {
		return "", err
	}
	return newfile, nil
}

func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		// Check for ZipSlip. More Info: http://bit.ly/2MsjAWE
		if !s.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}
