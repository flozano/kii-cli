package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path"

	"github.com/codegangsta/cli"
	"github.com/vaughan0/go-ini"
)

type GlobalConfig struct {
	AppId        string
	AppKey       string
	ClientId     string
	ClientSecret string
	Site         string
	endpointUrl  string
	devlogUrl    string
}

func (self *GlobalConfig) EndpointUrl() string {
	if self.endpointUrl != "" {
		return self.endpointUrl
	}
	hosts := map[string]string{
		"us": "api.kii.com",
		"jp": "api-jp.kii.com",
		"cn": "api-cn2.kii.com",
		"sg": "api-sg.kii.com",
	}
	host := hosts[globalConfig.Site]
	if host == "" {
		print("missing site, use --site or set KII_SITE\n")
		os.Exit(ExitMissingParams)
	}
	return fmt.Sprintf("https://%s/api", host)
}

func (self *GlobalConfig) EndpointUrlForApiLog() string {
	if self.devlogUrl != "" {
		return self.devlogUrl
	}
	hosts := map[string]string{
		"us": "apilog.kii.com",
		"jp": "apilog-jp.kii.com",
		"cn": "apilog-cn2.kii.com",
		"sg": "apilog-sg.kii.com",
	}
	host := hosts[globalConfig.Site]
	if host == "" {
		print("missing site, use --site or set KII_SITE\n")
		os.Exit(ExitMissingParams)
	}
	return fmt.Sprintf("wss://%s:443/logs", host)
}

func (self *GlobalConfig) HttpHeaders(contentType string) map[string]string {
	m := map[string]string{
		"x-kii-appid":  globalConfig.AppId,
		"x-kii-appkey": globalConfig.AppKey,
	}
	if len(contentType) > 0 {
		m["content-type"] = contentType
	}
	return m
}

func (self *GlobalConfig) HttpHeadersWithAuthorization(contentType string) map[string]string {
	m := self.HttpHeaders(contentType)
	oauth2 := (&OAuth2Response{}).Load()
	m["authorization"] = fmt.Sprintf("Bearer %s", oauth2.AccessToken)
	return m
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

// Return ~/.kii/${filename}
func metaFilePath(dir string, filename string) string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}
	confdirpath := path.Join(usr.HomeDir, ".kii", dir)
	err = os.MkdirAll(confdirpath, os.ModeDir|0700)
	if err != nil {
		panic(err)
	}
	return path.Join(confdirpath, filename)
}

var globalConfig *GlobalConfig
var _config = `[default]
app_id =
app_key =
client_id =
client_secret =
site = us
`

func loadIniFile() *ini.File {
	configPath := metaFilePath(".", "config")
	if b, _ := exists(configPath); !b {
		ioutil.WriteFile(configPath, []byte(_config), 0600)
	}
	file, err := ini.LoadFile(configPath)
	if err != nil {
		panic(err)
	}
	return &file
}

func pickup(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func setupFlags(app *cli.App) {
	app.Flags = []cli.Flag{
		cli.StringFlag{"app-id", "", "AppID"},
		cli.StringFlag{"app-key", "", "AppKey"},
		cli.StringFlag{"client-id", "", "ClientID"},
		cli.StringFlag{"client-secret", "", "ClientSecret"},
		cli.StringFlag{"site", "", "us,jp,cn,sg"},
		cli.StringFlag{"endpoint-url", "", "Site URL"},
		cli.BoolFlag{"verbose", "Verbosely"},
		cli.StringFlag{"profile", "default", "Profile name for ~/.kii/config"},
	}

	app.Before = func(c *cli.Context) error {
		profile := c.GlobalString("profile")
		inifile := loadIniFile()
		if profile != "default" && len((*inifile)[profile]) == 0 {
			print(fmt.Sprintf("profile %s is not found in ~/.kii/config\n", profile))
			os.Exit(ExitMissingParams)
		}

		getConf := func(gn, en, un string) string {
			ev := os.ExpandEnv("${" + en + "}")
			uv, _ := inifile.Get(profile, un)
			return pickup(c.GlobalString(gn), ev, uv)
		}

		globalConfig = &GlobalConfig{
			AppId:        getConf("app-id", "KII_APP_ID", "app_id"),
			AppKey:       getConf("app-key", "KII_APP_KEY", "app_key"),
			ClientId:     getConf("client-id", "KII_CLIENT_ID", "client_id"),
			ClientSecret: getConf("client-secret", "KII_CLIENT_SECRET", "client_secret"),
			Site:         getConf("site", "KII_SITE", "site"),
			endpointUrl:  getConf("endpoint-url", "KII_ENDPOINT_URL", "endpoint_url"),
			devlogUrl:    getConf("log-url", "KII_LOG_URL", "log_url"),
		}

		// Setup logger
		if c.Bool("verbose") {
			logger = log.New(os.Stderr, "", log.LstdFlags)
		}

		return nil
	}
}
