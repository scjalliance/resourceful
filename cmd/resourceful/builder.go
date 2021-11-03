//go:build generate
// +build generate

package main

import (
	"log"
	"net/http"
	"runtime"
	"strconv"
	"strings"

	"github.com/josephspurrier/goversioninfo"
	"github.com/shurcooL/vfsgen"
)

func main() {
	buildPublicFiles()
	buildVersionInfo()
}

func buildPublicFiles() {
	err := vfsgen.Generate(http.Dir("www"), vfsgen.Options{
		Filename:        "www.gen.go",
		PackageName:     "main",
		VariableName:    "webfiles",
		VariableComment: "webfiles is a directory of public files to be served over HTTP.",
	})
	if err != nil {
		log.Fatalln(err)
	}
}

func buildVersionInfo() {

	major, minor, patch, build := splitVersion(Version)
	fileVersion := goversioninfo.FileVersion{
		Major: major,
		Minor: minor,
		Patch: patch,
		Build: build,
	}
	vi := goversioninfo.VersionInfo{
		IconPath:     "icon.ico",
		ManifestPath: "resourceful.manifest",
		FixedFileInfo: goversioninfo.FixedFileInfo{
			FileVersion:    fileVersion,
			ProductVersion: fileVersion,
			FileFlagsMask:  "3f",
			FileFlags:      "00",
			FileOS:         "040004",
			FileType:       "01",
			FileSubType:    "00",
		},
		StringFileInfo: goversioninfo.StringFileInfo{
			CompanyName:      "SCJ Alliance",
			FileDescription:  "Resourceful",
			FileVersion:      Version,
			OriginalFilename: "resourceful.exe",
			ProductName:      "Resourceful",
			ProductVersion:   Version,
		},
		VarFileInfo: goversioninfo.VarFileInfo{
			Translation: goversioninfo.Translation{
				LangID:    goversioninfo.LngUSEnglish,
				CharsetID: goversioninfo.CsUnicode,
			},
		},
	}
	vi.Build()
	vi.Walk()
	vi.WriteSyso("resourceful.syso", runtime.GOARCH)
}

func splitVersion(version string) (major, minor, patch, build int) {
	parts := strings.Split(Version, ".")
	switch len(parts) {
	case 4:
		if val, err := strconv.Atoi(parts[3]); err == nil {
			build = val
		}
		fallthrough
	case 3:
		if val, err := strconv.Atoi(parts[2]); err == nil {
			patch = val
		}
		fallthrough
	case 2:
		if val, err := strconv.Atoi(parts[1]); err == nil {
			minor = val
		}
		fallthrough
	case 1:
		if val, err := strconv.Atoi(parts[0]); err == nil {
			major = val
		}
	}
	return
}
