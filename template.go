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
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/sprig"
	"github.com/go-yaml/yaml"

	"github.com/go-stack/stack" // Forked copy of https://github.com/go-stack/stack
	"github.com/jjeffery/kv"    // Forked copy of https://github.com/jjeffery/kv
)

// FuncMap augments the template functions with some standard string maniuplation functions
// for document format conversions
//
func FuncMap(funcs map[string]interface{}) (f template.FuncMap) {
	// For more documentation about templating see http://masterminds.github.io/sprig/
	f = sprig.TxtFuncMap()

	// marshaling functions that be be inserted into the templated files
	f["toJson"] = toJson
	f["toYaml"] = toYaml
	f["toToml"] = toToml

	for name, fun := range funcs {
		f[name] = fun
	}
	return f
}

// toJson takes an interface, marshals it to json, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toJson(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		// Swallow kv.inside of a template.
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
		// Swallow kv.inside of a template.
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
func (md *MetaData) NewTemplateVariables(jsonVals string, loadFiles []string, overrideVals map[string]string, ignoreAWSErrors bool) (vars map[string]interface{}, err kv.Error, warnings []kv.Error) {

	vars = map[string]interface{}{}

	// Env
	envs := map[string]interface{}{}
	for _, env := range os.Environ() {
		kv := strings.SplitN(env, "=", 2)
		envs[kv[0]] = kv[1]
	}
	vars["Env"] = envs

	duatVars := map[string]interface{}{
		"version":     md.SemVer.String(),
		"module":      md.Module,
		"gitTag":      md.Git.Tag,
		"gitHash":     md.Git.Hash,
		"gitBranch":   md.Git.Branch,
		"gitURL":      md.Git.URL,
		"gitDir":      md.Git.Dir,
		"userID":      md.user.Uid,
		"userName":    md.user.Username,
		"userGroupID": md.user.Gid,
	}

	ecrURL, err := GetECRDefaultURL()
	if err == nil && ecrURL != nil {
		duatVars["awsecr"] = ecrURL.Hostname()
	} else {
		if !ignoreAWSErrors {
			warnings = append(warnings, err)
		}
	}

	vars["duat"] = duatVars

	if jsonVals != "" {
		obj := map[string]interface{}{}
		if errGo := json.Unmarshal([]byte(jsonVals), &obj); errGo != nil {
			return nil, kv.Wrap(errGo, "bad json format").With("stack", stack.Trace().TrimRuntime()), warnings
		}
		for k, v := range obj {
			vars[k] = v
		}
	}

	for _, file := range loadFiles {
		if bytes, errGo := ioutil.ReadFile(file); errGo != nil {
			return nil, kv.Wrap(errGo).With("file", file).With("stack", stack.Trace().TrimRuntime()), warnings
		} else {
			obj := map[string]interface{}{}

			switch filepath.Ext(file) {
			case ".json":
				if errGo := json.Unmarshal(bytes, &obj); errGo != nil {
					return nil, kv.Wrap(errGo, "unrecognized json").With("file", file).With("stack", stack.Trace().TrimRuntime()), warnings
				}
			case ".yaml", ".yml":
				if errGo := yaml.Unmarshal(bytes, &obj); errGo != nil {
					return nil, kv.Wrap(errGo, "unrecognized yaml").With("file", file).With("stack", stack.Trace().TrimRuntime()), warnings
				}
			case ".toml":
				if errGo := toml.Unmarshal(bytes, &obj); errGo != nil {
					return nil, kv.Wrap(errGo, "unrecognized toml").With("file", file).With("stack", stack.Trace().TrimRuntime()), warnings
				}
			default:
				return nil, kv.NewError("unsupported file type (extension)").With("file", file).With("stack", stack.Trace().TrimRuntime()), warnings
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

	return vars, nil, warnings
}

func templateExecute(t *template.Template, src io.Reader, dest io.Writer, ctx interface{}) (err kv.Error) {

	readBytes, errGo := ioutil.ReadAll(src)
	if errGo != nil {
		return kv.Wrap(errGo, "pasing failed for template file(s)").With("stack", stack.Trace().TrimRuntime())
	}

	tmpl, errGo := t.Parse(string(readBytes))
	if errGo != nil {
		return kv.Wrap(errGo, "pasing failed for template file(s)").With("stack", stack.Trace().TrimRuntime())
	}

	if errGo = tmpl.Execute(dest, ctx); errGo != nil {
		return kv.Wrap(errGo, "output file could not be created").With("stack", stack.Trace().TrimRuntime())
	}
	return nil
}

// TemplateIOFiles is used to encapsulate some streaming interfaces for input and output documents
type TemplateIOFiles struct {
	In  io.Reader
	Out io.Writer
}

// TemplateOptions is used to pass into the Template function both streams and key values
// for the template engine
type TemplateOptions struct {
	IOFiles         []TemplateIOFiles
	Delimiters      []string
	ValueFiles      []string
	OverrideValues  map[string]string
	IgnoreAWSErrors bool
}

// Template takes the TemplateOptions and processes the template execution, it also
// it used to catch and report errors the user raises within the template from
// validation checking etc
//
func (md *MetaData) Template(opts TemplateOptions) (err kv.Error, warnings []kv.Error) {

	tmplErrs := []kv.Error{}
	funcs := template.FuncMap{
		"RaiseError": func(msg string) string {
			tmplErrs = append(tmplErrs, kv.NewError(msg).With("stack", stack.Trace().TrimRuntime()))
			return ""
		},
	}

	t := template.New("noname").Funcs(FuncMap(funcs))

	if len(opts.Delimiters) != 0 {
		if len(opts.Delimiters) != 2 {
			return kv.NewError("unexpected number of delimiters, tw are expected [\"left\",\"right\"").With("stack", stack.Trace().TrimRuntime()), warnings
		}
		t = t.Delims(opts.Delimiters[0], opts.Delimiters[1])
	}

	vars, err, warnings := md.NewTemplateVariables("", opts.ValueFiles, opts.OverrideValues, opts.IgnoreAWSErrors)
	if err != nil {
		return err, warnings
	}

	for _, files := range opts.IOFiles {
		err = templateExecute(t, files.In, files.Out, vars)
		if err != nil {
			return err, warnings
		}
	}
	if len(tmplErrs) != 0 {
		return tmplErrs[0], tmplErrs
	}

	return nil, warnings
}
