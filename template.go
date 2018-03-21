package duat

// This contains a simple templating engine wrapper that accepts arguments in
// serveral formats for the variables that will be applied to the template and
// uses the MasterMinds sprig library for additional functions within the templates
//
// A large portion of this code s dserived from an Apache 2.0 Licensed CLI utility
// that can be found at https://github.com/subchen/frep.  This file converts the
// non library packing of the original code based to be workable as a library

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig"
	"github.com/go-yaml/yaml"

	"github.com/karlmutch/errors" // Forked copy of https://github.com/jjeffery/errors
	"github.com/karlmutch/stack"  // Forked copy of https://github.com/go-stack/stack
)

func FuncMap() (f template.FuncMap) {
	// For more documentation about templating see http://masterminds.github.io/sprig/
	f = sprig.TxtFuncMap()

	// marshaling functions that be be inserted into the templated files
	f["toJson"] = toJson
	f["toYaml"] = toYaml
	f["toToml"] = toToml

	return f
}

// toJson takes an interface, marshals it to json, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toJson(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return string(data)
}

// toYaml takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toYaml(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return string(data)
}

// toToml takes an interface, marshals it to toml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toToml(v interface{}) string {
	b := bytes.NewBuffer(nil)
	e := toml.NewEncoder(b)
	err := e.Encode(v)
	if err != nil {
		return err.Error()
	}
	return b.String()
}

// create template context
func (md *MetaData) NewTemplateVariables(jsonVals string, loadFiles []string, overrideVals map[string]string) (vars map[string]interface{}, err errors.Error) {

	vars = map[string]interface{}{}

	// Env
	envs := map[string]interface{}{}
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		envs[kv[0]] = kv[1]
	}
	vars["Env"] = envs

	duatVars := map[string]interface{}{
		"version":   md.SemVer.String(),
		"module":    md.Module,
		"gitTag":    md.Git.Tag,
		"gitHash":   md.Git.Hash,
		"gitBranch": md.Git.Branch,
		"gitURL":    md.Git.URL,
		"gitDir":    md.Git.Dir,
	}
	if runtime, err := md.ContainerRuntime(); err == nil {
		duatVars["runtime"] = runtime
	}

	if ecrURL, err := GetECRDefaultURL(); err == nil {
		duatVars["awsecr"] = ecrURL.Hostname()
	}

	vars["duat"] = duatVars

	if jsonVals != "" {
		obj := map[string]interface{}{}
		if errGo := json.Unmarshal([]byte(jsonVals), &obj); errGo != nil {
			return nil, errors.Wrap(errGo, "bad json format").With("stack", stack.Trace().TrimRuntime())
		}
		for k, v := range obj {
			vars[k] = v
		}
	}

	for _, file := range loadFiles {
		if bytes, errGo := ioutil.ReadFile(file); errGo != nil {
			return nil, errors.Wrap(errGo).With("file", file).With("stack", stack.Trace().TrimRuntime())
		} else {
			obj := map[string]interface{}{}

			switch filepath.Ext(file) {
			case ".json":
				if errGo := json.Unmarshal(bytes, &obj); errGo != nil {
					return nil, errors.Wrap(errGo, "unrecognized json").With("file", file).With("stack", stack.Trace().TrimRuntime())
				}
			case ".yaml", ".yml":
				if errGo := yaml.Unmarshal(bytes, &obj); errGo != nil {
					return nil, errors.Wrap(errGo, "unrecognized yaml").With("file", file).With("stack", stack.Trace().TrimRuntime())
				}
			case ".toml":
				if errGo := toml.Unmarshal(bytes, &obj); errGo != nil {
					return nil, errors.Wrap(errGo, "unrecognized toml").With("file", file).With("stack", stack.Trace().TrimRuntime())
				}
			default:
				return nil, errors.New("unsupported file type (extension)").With("file", file).With("stack", stack.Trace().TrimRuntime())
			}

			for k, v := range obj {
				vars[k] = v
			}
		}
	}

	for k, v := range overrideVals {

		// remove quotes for key="value"
		if strings.HasPrefix(v, "\"") && strings.HasSuffix(v, "\"") {
			v = v[1 : len(v)-1]
		} else if strings.HasPrefix(v, "'") && strings.HasSuffix(v, "'") {
			v = v[1 : len(v)-1]
		}
		vars[k] = v
	}

	return vars, nil
}

func templateExecute(t *template.Template, src io.Reader, dest io.Writer, ctx interface{}) (err errors.Error) {

	readBytes, errGo := ioutil.ReadAll(src)
	if errGo != nil {
		return errors.Wrap(errGo, "pasing failed for template file(s)").With("stack", stack.Trace().TrimRuntime())
	}

	tmpl, errGo := t.Parse(string(readBytes))
	if errGo != nil {
		return errors.Wrap(errGo, "pasing failed for template file(s)").With("stack", stack.Trace().TrimRuntime())
	}

	if errGo = tmpl.Execute(dest, ctx); errGo != nil {
		return errors.Wrap(errGo, "output file could not be created").With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

type TemplateIOFiles struct {
	In  io.Reader
	Out io.Writer
}

type TemplateOptions struct {
	IOFiles        []TemplateIOFiles
	Delimiters     []string
	ValueFiles     []string
	OverrideValues map[string]string
}

func (md *MetaData) Template(opts TemplateOptions) (err errors.Error) {

	defer func() {
		if err := recover(); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}()

	t := template.New("noname").Funcs(FuncMap())
	if len(opts.Delimiters) != 0 {
		if len(opts.Delimiters) != 2 {
			return errors.New("unexpected number of delimiters, tw are expected [\"left\",\"right\"").With("stack", stack.Trace().TrimRuntime())
		}
		t = t.Delims(opts.Delimiters[0], opts.Delimiters[1])
	}

	vars, err := md.NewTemplateVariables("", opts.ValueFiles, opts.OverrideValues)
	if err != nil {
		return err
	}

	for _, files := range opts.IOFiles {
		err = templateExecute(t, files.In, files.Out, vars)
		if err != nil {
			return err
		}
	}

	return nil
}
