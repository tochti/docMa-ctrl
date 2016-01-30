package cmds

import (
	"github.com/tochti/docMa-handler/accountingData"
	"github.com/tochti/docMa-handler/common"
	"github.com/tochti/docMa-handler/dbVars"
	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
	"github.com/tochti/gin-gum/gumauth"
)

func CreateTables() error {
	db := common.InitMySQL()

	gumauth.AddTables(db)
	dbVars.AddTables(db)
	docs.AddTables(db)
	labels.AddTables(db)
	accountingData.AddTables(db)

	err := db.CreateTablesIfNotExists()
	if err != nil {
		return err
	}

	return nil

}
