package main

import (
	"path/filepath"
	"strings"
	tmpl "text/template"
)

const (
	Service   = "<ks:service>"
	Creator   = "<ks:creator>"
	Post      = "<ks:post>"
	Index     = "<ks:index>"
	Filename  = "<ks:filename>"
	Filehash  = "<ks:filehash>"
	Extension = "<ks:extension>"
)

const (
	TmplDefault          = "[" + Service + "]" + Creator + "/" + Post + "/" + Filename + Extension
	TmplWithPrefixNumber = "[" + Service + "]" + Creator + "/" + Post + "/" + Index + "_" + Filename + Extension
	TmplIndexNumber      = "[" + Service + "]" + Creator + "/" + Post + "/" + Index + Extension
)

type PathConfig struct {
	Service   string
	Creator   string
	Post      string
	Index     int
	Filename  string
	Filehash  string
	Extension string
}

func LoadPathTmpl(templateStr string, output string) (*tmpl.Template, error) {
	templateStr = filepath.Join(output, templateStr)
	templateStr = strings.ReplaceAll(templateStr, Service, "{{.Service}}")
	templateStr = strings.ReplaceAll(templateStr, Creator, "{{.Creator}}")
	templateStr = strings.ReplaceAll(templateStr, Post, "{{.Post}}")
	templateStr = strings.ReplaceAll(templateStr, Index, "{{.Index}}")
	templateStr = strings.ReplaceAll(templateStr, Filename, "{{.Filename}}")
	templateStr = strings.ReplaceAll(templateStr, Filehash, "{{.Filehash}}")
	templateStr = strings.ReplaceAll(templateStr, Extension, "{{.Extension}}")
	tmpl, err := tmpl.New("path").Parse(templateStr)
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func ExecutePathTmpl(tmpl *tmpl.Template, config *PathConfig) string {
	var path strings.Builder
	err := tmpl.Execute(&path, config)
	if err != nil {
		panic(err)
	}

	return path.String()
}

type TmplCache struct {
	tmpl map[string]*tmpl.Template
}

func NewTmplCache() *TmplCache {
	return &TmplCache{
		tmpl: make(map[string]*tmpl.Template),
	}
}

func (t *TmplCache) init() {
	// default template
	tmpl, err := LoadPathTmpl(template, output)
	if err != nil {
		panic(err)
	}
	t.tmpl["default"] = tmpl
	if imageTemplate != "" {
		tmpl, err := LoadPathTmpl(imageTemplate, output)
		if err != nil {
			panic(err)
		}
		t.tmpl["image"] = tmpl
	}
	if videoTemplate != "" {
		tmpl, err := LoadPathTmpl(videoTemplate, output)
		if err != nil {
			panic(err)
		}
		t.tmpl["video"] = tmpl
	}
	if audioTemplate != "" {
		tmpl, err := LoadPathTmpl(audioTemplate, output)
		if err != nil {
			panic(err)
		}
		t.tmpl["audio"] = tmpl
	}
	if archiveTemplate != "" {
		tmpl, err := LoadPathTmpl(archiveTemplate, output)
		if err != nil {
			panic(err)
		}
		t.tmpl["archive"] = tmpl
	}
}

func (t *TmplCache) GetTmpl(typ string) *tmpl.Template {
	if tmpl, ok := t.tmpl[typ]; ok {
		return tmpl
	}
	return t.tmpl["default"]
}

func (t *TmplCache) Execute(typ string, config *PathConfig) string {
	return ExecutePathTmpl(t.GetTmpl(typ), config)
}

func getTyp(ext string) string {
	switch ext {
	case ".apng", ".avif", ".bmp", ".gif", ".ico", ".cur", ".jpg", ".jpeg", ".jfif", ".pjpeg", ".pjp", ".png", ".svg", ".tif", ".tiff", ".webp", ".jpe":
		return "image"
	case ".mp4", ".webm", ".mkv", ".avi", ".mov", ".wmv", ".flv", ".f4v", ".m4v", ".rmvb", ".rm", ".3gp", ".dat", ".ts", ".mts", ".vob":
		return "video"
	case ".mp3", ".wav", ".flac", ".ape", ".aac", ".ogg", ".wma", ".m4a", ".aiff", ".alac":
		return "audio"
	case ".zip", ".rar", ".7z", ".tar", ".gz", ".bz2", ".xz", ".zipmod":
		return "archive"
	default:
		return "default"
	}
}
