package cmds

import (
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strings"
	"time"

	"github.com/tochti/docMa-handler/common"
	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
)

func ImportDocs(dir string) error {
	db := common.InitMySQL()

	docs.AddTables(db)
	labels.AddTables(db)

	l, err := ioutil.ReadDir(dir)
	if err != nil {
		return err
	}

	newLabel := labels.Label{}
	err = db.SelectOne(
		&newLabel,
		fmt.Sprintf("SELECT * FROM %v WHERE name='Neu'", labels.LabelsTable),
	)
	if err != nil {
		return err
	}

	newDocs := []interface{}{}
	for _, doc := range l {
		now := time.Now()
		d := docs.Doc{
			Name:          path.Base(doc.Name()),
			DateOfScan:    now,
			DateOfReceipt: now,
		}
		err = db.Insert(&d)
		if err != nil {
			if strings.Contains(err.Error(), "Duplicate entry") {
				log.Println(err)
				continue
			}

			return err
		}

		newDocs = append(newDocs, &d)
	}

	dFn := func(i interface{}) string {
		d, _ := i.(*docs.Doc)
		return fmt.Sprintf("(%v,%v)", d.ID, newLabel.ID)
	}
	err = BatchInsert(db, "(doc_id,label_id)", newDocs, docs.DocsLabelsTable, dFn)
	if err != nil {
		return err
	}

	return nil
}
