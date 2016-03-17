package cmds

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/tochti/docMa-handler/common"
	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
)

var (
	ErrFilenameFormat = errors.New("Wrong filename format")
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
		filename := path.Base(doc.Name())
		date, id, err := ParseFilename(filename)
		if err != nil {
			log.Println(err)
		}
		d := docs.Doc{
			Name:          filename,
			Barcode:       id,
			DateOfScan:    date,
			DateOfReceipt: date,
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

	zeroDate := time.Time{}
	dFn = func(i interface{}) string {
		d, _ := i.(*docs.Doc)
		return fmt.Sprintf("(%v,%v,'%v','%v')", d.ID, 0, zeroDate, zeroDate)
	}
	err = BatchInsert(
		db,
		"(doc_id,account_number,period_from,period_to)",
		newDocs,
		docs.DocAccountDataTable,
		dFn,
	)
	if err != nil {
		return err
	}

	return nil
}

func ParseFilename(n string) (time.Time, string, error) {
	ext := path.Ext(n)
	if len(ext) > 0 {
		n = strings.Replace(n, ext, "", -1)
	}
	r := strings.Split(n, "_")
	zeroDate := time.Time{}

	if len(r) != 2 {
		return zeroDate, "", ErrFilenameFormat
	}
	if len(r[0]) != 8 {
		return zeroDate, "", ErrFilenameFormat
	}
	if len(r[1]) != 7 {
		return zeroDate, "", ErrFilenameFormat
	}

	year, err := strconv.ParseInt(r[0][0:4], 10, 32)
	if err != nil {
		return zeroDate, "", ErrFilenameFormat
	}
	month, err := strconv.ParseInt(r[0][4:6], 10, 32)
	if err != nil {
		return zeroDate, "", ErrFilenameFormat
	}
	day, err := strconv.ParseInt(r[0][6:8], 10, 32)
	if err != nil {
		return zeroDate, "", ErrFilenameFormat
	}
	date := time.Date(int(year), time.Month(int(month)), int(day), 0, 0, 0, 0, time.Local)

	id := r[1]

	return date, id, nil
}
