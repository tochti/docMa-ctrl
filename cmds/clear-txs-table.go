package cmds

import (
	"fmt"

	"github.com/tochti/docMa-handler/accountingData"
	"github.com/tochti/docMa-handler/common"
)

func ClearAccountingTxsTable() error {
	db := common.InitMySQL()

	_, err := db.Exec(fmt.Sprintf("TRUNCATE TABLE %v", accountingData.AccountingDataTable))
	if err != nil {
		return err
	}

	return nil
}
