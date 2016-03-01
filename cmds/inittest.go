package cmds

import (
	"os"
	"testing"

	"github.com/tochti/docMa-handler/accountingData"
	"github.com/tochti/docMa-handler/common"
	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
	"gopkg.in/gorp.v1"
)

var (
	MySQLTestDB   = "testing"
	MongoDBTestDB = "testing"
)

func setenv() {
	os.Clearenv()

	os.Setenv("MYSQL_USER", "tochti")
	os.Setenv("MYSQL_PASSWORD", "123")
	os.Setenv("MYSQL_HOST", "127.0.0.1")
	os.Setenv("MYSQL_PORT", "3306")
	os.Setenv("MYSQL_DB_NAME", MySQLTestDB)

	os.Setenv("MONGODB_HOST", "127.0.0.1")
	os.Setenv("MONGODB_DB_NAME", MongoDBTestDB)
}

func initMySQL(t *testing.T) *gorp.DbMap {
	setenv()
	dbMap := common.InitMySQL()

	docs.AddTables(dbMap)
	labels.AddTables(dbMap)
	accountingData.AddTables(dbMap)

	err := dbMap.DropTablesIfExists()
	if err != nil {
		t.Fatal(err)
	}

	err = dbMap.CreateTablesIfNotExists()
	if err != nil {
		t.Fatal(err)
	}

	return dbMap
}
