package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/tochti/docMa-ctrl/cmds"
	"github.com/tochti/gin-gum/gumspecs"
)

type (
	blackhole struct{}
)

func main() {
	var newUser bool
	var createTables bool
	var migrate bool
	var docsPath string
	var txsPath string
	var cleartxstable bool
	var debug bool

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
