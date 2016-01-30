package cmds

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/tochti/gin-gum/gumauth"
	"github.com/tochti/gin-gum/gumspecs"
)

func NewUser() error {
	mysql := gumspecs.ReadMySQL()

	db, err := mysql.DB()
	if err != nil {
		return err
	}
	defer db.Close()

	username, password := userMenu()
	user := &gumauth.User{
		Username: username,
		Password: gumauth.NewSha512Password(password),
		IsActive: true,
	}

	err = gumauth.SQLNewUser(db, user)
	if err != nil {
		return err
	}

	return nil

}

func userMenu() (string, string) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Username: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)
	fmt.Print("Password: ")
	pass, _ := reader.ReadString('\n')
	pass = strings.TrimSpace(pass)
	return username, pass
}
