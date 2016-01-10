package cmds

import (
	"github.com/tochti/docMa-handler"
	"github.com/tochti/docMa-handler/labels"
)

func CreateTables() error {
	db := bebber.InitMySQL()

	return labels.CreateTables(db)
}
