package main

import (
  "os"
  "fmt"
  "flag"
  "path"
  "bufio"
  "strings"
  "crypto/sha1"

  "github.com/rrawrriw/bebber"
  "gopkg.in/mgo.v2"
  "gopkg.in/mgo.v2/bson"
)

func main() {
  adduser := flag.Bool("adduser", false, "Add default user")
  flag.Parse()

  if *adduser {
    session, err := mgo.Dial(bebber.GetSettings("BEBBER_DB_SERVER"))
    if err != nil {
      fmt.Println(err.Error())
      return
    }
    db := session.DB(bebber.GetSettings("BEBBER_DB_NAME"))
    users := db.C(bebber.UsersCollection)
    username, password, dirs := UserMenu()
    if ExistsUser(username, users) {
      fmt.Println(username, "already exists!")
      return
    }
    user := bebber.User{Username: username,
                        Password: password,
                        Dirs: dirs}
    err = users.Insert(user)
    if err != nil {
      fmt.Println(err.Error())
      return
    } else {
      fmt.Println(username, "save completed!")
    }
  }
}

func UserMenu() (string, string, map[string]string) {
  reader := bufio.NewReader(os.Stdin)
  fmt.Print("Username: ")
  username, _ := reader.ReadString('\n')
  username = strings.TrimSpace(username)
  fmt.Print("Password: ")
  passTmp, _ := reader.ReadString('\n')
  passTmp = strings.TrimSpace(passTmp)
  password := fmt.Sprintf("%x", sha1.Sum([]byte(passTmp)))
  dirs := make(map[string]string)
  fmt.Println("Direcotries - name:\"/rel/path\" - . to finish")
  for {
    fmt.Print("> ")
    input, _ := reader.ReadString('\n')
    if input == ".\n" {
      break
    }

    tmp := strings.Split(strings.TrimSpace(input), ":")
    if len(tmp) != 2 {
      fmt.Println("Wrong input format")
      continue
    }

    name := tmp[0]
    relPath := tmp[1]
    dir := path.Join(bebber.GetSettings("BEBBER_FILES"), relPath)
    if _, err := os.Stat(dir); os.IsNotExist(err) {
      fmt.Println("Cannot find file or directory:", dir)
      continue
    } else {
      dirs[name] = relPath
    }
  }

  return username, password, dirs
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
