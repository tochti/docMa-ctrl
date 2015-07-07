package main

import (
  "os"
  "fmt"
  "flag"
  "path"
  "time"
  "bufio"
  "bytes"
  "errors"
  "io/ioutil"
  "strings"
  "net/http"
  "crypto/sha1"
  "encoding/json"

  "github.com/rrawrriw/bebber"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

func main() {
  var appendUser bool
  var docsPath string
  var accProcessFile string
  var user string

  flag.BoolVar(&appendUser, "adduser", false, "Add default user")
  flag.StringVar(&docsPath, "importtestdocs", "", "Reset docs collection")
  flag.StringVar(&accProcessFile, "importaccprocess", "", "Import account process from csv")
  flag.StringVar(&user, "user", "", "User to access http-server")
  flag.Parse()

  if appendUser {
    AppendUser()
  }

  if (accProcessFile != "") && (user != "") {
    ImportAccProcessFile(accProcessFile, user)
  }

  if (docsPath != "") && (user != "") {
    ImportTestDocs(docsPath, user)
  }
}

func ImportAccProcessFile(accProcessFile string, user string) {
  urlPrefix := ReadURLPrefix()
  client := http.DefaultClient
  token := LoginWithUser(client, user)

  accProcessList, err := bebber.ReadAccProcessFile(accProcessFile)
  if err != nil {
    fmt.Println(err)
    os.Exit(2)
  }

  makeURL := urlPrefix + "/AccProcess"
  authHeader := http.Header{}
  authHeader.Add(bebber.TokenHeaderField, token)
  for _,ap := range accProcessList {
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

func ImportTestDocs(docsPath string, user string) {
  urlPrefix := ReadURLPrefix()
  client := http.DefaultClient
  token := LoginWithUser(client, user)

  docs, err := ioutil.ReadDir(docsPath)
  if err != nil {
    fmt.Println(err)
    os.Exit(2)
  }

  makeUrl := urlPrefix + "/Doc"
  authHeader := http.Header{}
  authHeader.Add(bebber.TokenHeaderField, token)
  for i,f := range docs {
    base := path.Base(f.Name())
    now := time.Now()
    requestBody := fmt.Sprintf(`{
      "Name": "%v",
      "Infos": {"DateOfScan": "%v", "DateOfReceipt": "%v"},
      "Labels": ["Neu"]
    }`, base, now.Format(time.RFC3339), now.Format(time.RFC3339))
    reader := strings.NewReader(requestBody)
    makeRequest, err := http.NewRequest("POST", makeUrl, reader)
    makeRequest.Header = authHeader
    if err != nil {
      fmt.Println(err.Error())
      continue
    }
    fmt.Println(i, base)
    response, err := client.Do(makeRequest)
    if err != nil {
      fmt.Println(response)
      fmt.Println(err.Error())
      continue
    }
    buf := bytes.Buffer{}
    buf.ReadFrom(response.Body)
    fmt.Println(buf.String())
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

func AppendUser() {
  session, err := mgo.Dial(bebber.GetSettings("BEBBER_DB_SERVER"))
  if err != nil {
    fmt.Println(err.Error())
    return
  }
  db := session.DB(bebber.GetSettings("BEBBER_DB_NAME"))
  users := db.C(bebber.UsersColl)
  username, password := UserMenu()
  if ExistsUser(username, users) {
    fmt.Println(username, "already exists!")
    return
  }
  user := bebber.User{Username: username,
                      Password: password}
  err = users.Insert(user)
  if err != nil {
    fmt.Println(err.Error())
    return
  } else {
    fmt.Println(username, "save completed!")
  }
}

func UserMenu() (string, string) {
  reader := bufio.NewReader(os.Stdin)
  fmt.Print("Username: ")
  username, _ := reader.ReadString('\n')
  username = strings.TrimSpace(username)
  fmt.Print("Password: ")
  passTmp, _ := reader.ReadString('\n')
  passTmp = strings.TrimSpace(passTmp)
  password := fmt.Sprintf("%x", sha1.Sum([]byte(passTmp)))
  return username, password
}

func PasswordMenu() (string, error) {
  fmt.Print("Password: ")
  reader := bufio.NewReader(os.Stdin)
  passTmp, err := reader.ReadString('\n')
  if err != nil {
    return "", err
  }
  pass := strings.TrimSpace(passTmp)
  return pass, nil;
}

func ExistsUser(name string, users *mgo.Collection) bool {
  n, err := users.Find(bson.M{"username": name}).Count()
  if (err != nil) {
    return false
  }

  if (n > 0) {
    return true
  } else {
    return false
  }
}

func ReadURLPrefix() string {
  httpHost := bebber.GetSettings("BEBBER_IP")
  httpPort := bebber.GetSettings("BEBBER_PORT")
  return fmt.Sprintf("http://%v:%v", httpHost, httpPort)
}

func LoginWithUser(client *http.Client, user string) string {
  password, err := PasswordMenu()
  if err != nil {
    fmt.Println(err);
    os.Exit(2);
  }
  urlPrefix := ReadURLPrefix()
  token, err :=  Login(user, password, urlPrefix, client)
  if err != nil {
    fmt.Println(err)
    os.Exit(2)
  }
  return token
}
