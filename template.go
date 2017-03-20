package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"

	"github.com/jwilder/gojq"
)

type ConfigData struct {
	IsDynamic bool   `json:"is_dynamic"`
	Key       string `json:"key"`
	Value     string `json:"value"`
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func fetchRemoteConfig(url string, defaultVal string) (string, error) {
	var config ConfigData
	var client = &http.Client{}
	var urlParts = strings.Split(url, "/")
	var configKey = urlParts[len(urlParts)-1]

	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Fatalf("Create request error %s: %s", url, err)
		return "", err
	}

	response, err := client.Do(request)
	if err != nil {
		log.Fatalf("Get response error %s: %s", url, err)
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode == 200 {
		body, _ := ioutil.ReadAll(response.Body)
		err := json.Unmarshal(body, &config)
		if err != nil {
			log.Fatalf("Parse response error %s", err)
		}
		log.Printf("%s: 200 Success, fetch **Remote* '%s'", configKey, config.Value)
		return config.Value, nil
	} else if response.StatusCode == 404 {
		if defaultVal != "" {
			log.Printf("%s: 404 Not Found, use *Default* '%s'", configKey, defaultVal)
			return defaultVal, nil
		} else {
			return "", fmt.Errorf("%s: (Fatal!) 404 Not Found, no *Default* value.", configKey)
		}
	}
	return "", fmt.Errorf("Response for %s error, Status code : %d", url, response.StatusCode)
}

func configer(args ...interface{}) (string, error) {
	var (
		configHost         string
		configEndpoint     string
		defaultConfigValue string
	)

	if len(args) < 2 {
		return "", fmt.Errorf("configer called with no values!")
	}

	if len(args) >= 2 {
		if args[0] == nil {
			return "", fmt.Errorf("configer configHost is nil value!")
		}
		if args[1] == nil {
			return "", fmt.Errorf("configer configUrl is nil value!")
		}

		if _, ok := args[0].(string); !ok {
			return "", fmt.Errorf("configer configHost is not a string value. hint: surround it w/ double quotes.")
		}
		if _, ok := args[1].(string); !ok {
			return "", fmt.Errorf("configer configUrl is not a string value. hint: surround it w/ double quotes.")
		}
		if len(args) == 3 {
			defaultConfigValue = args[2].(string)
		}
		configHost = args[0].(string)
		configEndpoint = args[1].(string)
	}

	configUrl, err := url.Parse(configHost)
	if err != nil {
		log.Fatal(err)
	}
	configUrl.Path = path.Join(configUrl.Path, configEndpoint)

	configValue, err := fetchRemoteConfig(configUrl.String(), defaultConfigValue)
	if err != nil {
		log.Fatal(err)
		return "", err
	}
	return configValue, nil
}

func contains(item map[string]string, key string) bool {
	if _, ok := item[key]; ok {
		return true
	}
	return false
}

func defaultValue(args ...interface{}) (string, error) {
	if len(args) == 0 {
		return "", fmt.Errorf("default called with no values!")
	}

	if len(args) > 0 {
		if args[0] != nil {
			return args[0].(string), nil
		}
	}

	if len(args) > 1 {
		if args[1] == nil {
			return "", fmt.Errorf("default called with nil default value!")
		}

		if _, ok := args[1].(string); !ok {
			return "", fmt.Errorf("default is not a string value. hint: surround it w/ double quotes.")
		}

		return args[1].(string), nil
	}

	return "", fmt.Errorf("default called with no default value")
}

func parseUrl(rawurl string) *url.URL {
	u, err := url.Parse(rawurl)
	if err != nil {
		log.Fatalf("unable to parse url %s: %s", rawurl, err)
	}
	return u
}

func add(arg1, arg2 int) int {
	return arg1 + arg2
}

func isTrue(s string) bool {
	b, err := strconv.ParseBool(strings.ToLower(s))
	if err == nil {
		return b
	}
	return false
}

func jsonQuery(jsonObj string, query string) (interface{}, error) {
	parser, err := gojq.NewStringQuery(jsonObj)
	if err != nil {
		return "", err
	}
	res, err := parser.Query(query)
	if err != nil {
		return "", err
	}
	return res, nil
}

func generateFile(templatePath, destPath string) bool {
	tmpl := template.New(filepath.Base(templatePath)).Funcs(template.FuncMap{
		"configer":  configer,
		"contains":  contains,
		"exists":    exists,
		"split":     strings.Split,
		"replace":   strings.Replace,
		"default":   defaultValue,
		"parseUrl":  parseUrl,
		"atoi":      strconv.Atoi,
		"add":       add,
		"isTrue":    isTrue,
		"lower":     strings.ToLower,
		"upper":     strings.ToUpper,
		"jsonQuery": jsonQuery,
	})

	if len(delims) > 0 {
		tmpl = tmpl.Delims(delims[0], delims[1])
	}
	tmpl, err := tmpl.ParseFiles(templatePath)
	if err != nil {
		log.Fatalf("unable to parse template: %s", err)
	}

	dest := os.Stdout
	if destPath != "" {
		dest, err = os.Create(destPath)
		if err != nil {
			log.Fatalf("unable to create %s", err)
		}
		defer dest.Close()
	}
	// log.Println(ConfigHostFlag)

	err = tmpl.ExecuteTemplate(dest, filepath.Base(templatePath), &Context{ConfigCenterHost: ConfigHostFlag.String()})
	if err != nil {
		log.Fatalf("template error: %s\n", err)
	}

	if fi, err := os.Stat(destPath); err == nil {
		if err := dest.Chmod(fi.Mode()); err != nil {
			log.Fatalf("unable to chmod temp file: %s\n", err)
		}
		if err := dest.Chown(int(fi.Sys().(*syscall.Stat_t).Uid), int(fi.Sys().(*syscall.Stat_t).Gid)); err != nil {
			log.Fatalf("unable to chown temp file: %s\n", err)
		}
	}

	return true
}

func generateDir(templateDir, destDir string) bool {
	if destDir != "" {
		fiDest, err := os.Stat(destDir)
		if err != nil {
			log.Fatalf("unable to stat %s, error: %s", destDir, err)
		}
		if !fiDest.IsDir() {
			log.Fatalf("if template is a directory, dest must also be a directory (or stdout)")
		}
	}

	files, err := ioutil.ReadDir(templateDir)
	if err != nil {
		log.Fatalf("bad directory: %s, error: %s", templateDir, err)
	}

	for _, file := range files {
		if destDir == "" {
			generateFile(filepath.Join(templateDir, file.Name()), "")
		} else {
			generateFile(filepath.Join(templateDir, file.Name()), filepath.Join(destDir, file.Name()))
		}
	}

	return true
}
