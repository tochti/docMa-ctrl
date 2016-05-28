package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/tochti/docMa-ctrl/cmds"
	"github.com/tochti/docMa-handler"
	"github.com/tochti/gin-gum/gumspecs"
)

type (
	blackhole struct{}
)

func main() {
	var accProcessFile string
	var user string
	var password string
	var newUser bool
	var createTables bool
	var migrate bool
	var docsPath string
	var txsPath string
	var cleartxstable bool
	var debug bool

	//flag.StringVar(&accProcessFile, "importaccprocess", "", "Import account process from csv")
	//flag.StringVar(&user, "user", "", "User to access http-server")
	//flag.StringVar(&password, "password", "", "Password")

	flag.BoolVar(&newUser, "newuser", false, "Create new default user")
	flag.BoolVar(&createTables, "createtables", false, "Create all database tables")
	flag.BoolVar(&migrate, "migrate", false, "Migrate from mongodb to mysql")
	flag.BoolVar(&debug, "debug", false, "Enable debugging output")
	flag.StringVar(&docsPath, "importdocs", "", "Import docs")
	flag.StringVar(&txsPath, "importtxs", "", "Import accounting transactions")
	flag.BoolVar(&cleartxstable, "cleartxstable", false, "Clear accounting transaction database table")

	flag.Parse()

	gumspecs.AppName = "docma"

	if !debug {
		log.SetOutput(blackhole{})
	}

	if newUser {
		err := cmds.NewUser()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("New user created!")
		return
	}

	if createTables {
		err := cmds.CreateTables()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Tabels created")
		return
	}

	if migrate {
		err := cmds.Migrate()
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Migration done")
		return
	}

	if (accProcessFile != "") && (user != "") {
		ImportAccProcessFile(accProcessFile, user, password)
	}

	if docsPath != "" {
		err := cmds.ImportDocs(docsPath)
		if err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println("Import done")
		return
	}

	if txsPath != "" {
		err := cmds.ImportAccountingTxs(txsPath)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if cleartxstable {
		err := cmds.ClearAccountingTxsTable()
		if err != nil {
			fmt.Println(err)
			return
		}
	}

}

func (blackhole) Write(b []byte) (int, error) {
	return 0, nil
}

func ImportAccProcessFile(accProcessFile string, user string, password string) {
	urlPrefix := ReadURLPrefix()
	client := http.DefaultClient
	token := LoginWithUser(client, user, password)

	accProcessList, err := bebber.ReadAccProcessFile(accProcessFile)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	makeURL := urlPrefix + "/AccProcess"
	authHeader := http.Header{}
	authHeader.Add(bebber.TokenHeaderField, token)
	for _, ap := range accProcessList {
		accProcessJSON, err := json.Marshal(ap)
		if err != nil {
			fmt.Println(err)
			continue
		}

		body := strings.NewReader(string(accProcessJSON))
		request, err := http.NewRequest("POST", makeURL, body)
		request.Header = authHeader
		if err != nil {
			fmt.Println(err.Error())
			continue
		}
		response, err := client.Do(request)
		if err != nil {
			fmt.Println(err.Error())
		}
		if response.StatusCode != 200 {
			fmt.Println(response)
		} else {
			buf := bytes.Buffer{}
			buf.ReadFrom(response.Body)
			fmt.Println(buf.String())
		}
	}

}

func Login(user, password, urlPrefix string, c *http.Client) (string, error) {
	loginUrl := urlPrefix + "/Login"
	loginRequest := fmt.Sprintf(`{"Username":"%v","Password":"%v"}`, user, password)
	buf := strings.NewReader(loginRequest)
	response, err := c.Post(loginUrl, "application/json", buf)
	if err != nil {
		fmt.Println(err.Error())
		b := bytes.Buffer{}
		b.ReadFrom(response.Body)
		fmt.Println(b.String())
		return "", errors.New("Cannot Login")
	}

	token, err := ReadToken(response)
	if err != nil {
		return "", err
	}

	return token, nil
}

func ReadToken(r *http.Response) (string, error) {
	cookies := r.Cookies()
	token := ""
	for _, v := range cookies {
		if v.Name == bebber.XSRFCookieName {
			token = v.Value
		}
	}
	if token != "" {
		return token, nil
	} else {
		return token, errors.New("Cannot found token!")
	}
}

func PasswordMenu() (string, error) {
	fmt.Print("Password: ")
	reader := bufio.NewReader(os.Stdin)
	passTmp, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	pass := strings.TrimSpace(passTmp)
	return pass, nil
}

func ReadURLPrefix() string {
	httpHost := bebber.GetSettings("BEBBER_IP")
	httpPort := bebber.GetSettings("BEBBER_PORT")
	return fmt.Sprintf("http://%v:%v", httpHost, httpPort)
}

func LoginWithUser(client *http.Client, user string, password string) string {
	if password == "" {
		var err error
		password, err = PasswordMenu()
		if err != nil {
			fmt.Println(err)
			os.Exit(2)
		}
	}

	urlPrefix := ReadURLPrefix()
	token, err := Login(user, password, urlPrefix, client)
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}
	return token
}
