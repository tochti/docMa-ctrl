package cmds

import (
	"fmt"
	"io"
	"os"

	"github.com/tochti/docMa-accountant/accountantService/accountingTxsFileReader"
	"github.com/tochti/docMa-handler/accountingData"
	"github.com/tochti/docMa-handler/common"
)

func ImportAccountingTxs(txsFile string) error {
	db := common.InitMySQL()
	accountingData.AddTables(db)

	fh, err := os.Open(txsFile)
	if err != nil {
		return err
	}

	reader := accountingTxsFileReader.NewReader(fh)

	for {
		tx, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}

			return err
		}

		err = db.Insert(&tx)
		if err != nil {
			return fmt.Errorf("Line: %v, Error: %v", tx, err)
		}

	}

	return nil
}
