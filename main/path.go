package main

import (
	"path/filepath"
	"strings"
	tmpl "text/template"
)

const (
	Creator   = "<ks:creator>"
	Post      = "<ks:post>"
	Index     = "<ks:index>"
	Filename  = "<ks:filename>"
	Extension = "<ks:extension>"
)

const (
	TmplDefault          = Creator + "/" + Post + "/" + Filename + Extension
	TmplWithPrefixNumber = Creator + "/" + Post + "/" + Index + "_" + Filename + Extension
	TmplIndexNumber      = Creator + "/" + Post + "/" + Index + Extension
)

type PathConfig struct {
	Creator   string
	Post      string
	Index     int
	Filename  string
	Extension string
}

func LoadPathTmpl(templateStr string, output string) (*tmpl.Template, error) {
	//data, err := ioutil.ReadFile(configPath)
	//if err != nil {
	//	return nil, err
	//}
	//templateStr := string(data)
	templateStr = filepath.Join(output, templateStr)
	templateStr = strings.ReplaceAll(templateStr, Creator, "{{.Creator}}")
	templateStr = strings.ReplaceAll(templateStr, Post, "{{.Post}}")
	templateStr = strings.ReplaceAll(templateStr, Index, "{{.Index}}")
	templateStr = strings.ReplaceAll(templateStr, Filename, "{{.Filename}}")
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
