package cmds

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"path"
	"strconv"
	"strings"
	"time"

	"gopkg.in/gorp.v1"

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
		date, barcode, err := ParseFilename(filename)
		if err != nil {
			log.Println(err)
		}
		d := docs.Doc{
			Name:          filename,
			Barcode:       barcode,
			DateOfScan:    date,
			DateOfReceipt: date,
		}
		id, err := InsertOrUpdateDoc(db, d)
		if err != nil {
			return err
		}
		d.ID = id

		newDocs = append(newDocs, &d)
	}

	dFn := func(i interface{}) string {
		d, _ := i.(*docs.Doc)
		return fmt.Sprintf("(%v,%v)", d.ID, newLabel.ID)
	}
	err = BatchInsertOrIgnore(db, "(doc_id,label_id)", newDocs, docs.DocsLabelsTable, dFn)
	if err != nil {
		return err
	}

	zeroDate := time.Time{}
	dFn = func(i interface{}) string {
		d, _ := i.(*docs.Doc)
		return fmt.Sprintf("(%v,%v,'%v','%v')", d.ID, 0, zeroDate, zeroDate)
	}
	err = BatchInsertOrIgnore(
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

func InsertOrUpdateDoc(db *gorp.DbMap, doc docs.Doc) (int64, error) {
	q := fmt.Sprintf(`
		INSERT INTO %v 
		(name, barcode, date_of_scan, date_of_receipt, note)
		VALUES (?,?,?,?,?)
		ON DUPLICATE KEY UPDATE id=LAST_INSERT_ID(id), barcode=?, date_of_scan=?, date_of_receipt=?, note=?`,
		docs.DocsTable)

	result, err := db.Exec(q,
		doc.Name, doc.Barcode, doc.DateOfScan, doc.DateOfReceipt, doc.Note,
		doc.Barcode, doc.DateOfScan, doc.DateOfReceipt, doc.Note)
	if err != nil {
		return -1, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return -1, err
	}

	return id, nil
}

func BatchInsertOrIgnore(sqlDB *gorp.DbMap, fields string, data []interface{}, table string, valueSta func(interface{}) string) error {
	if len(data) == 0 {
		return nil
	}

	q := fmt.Sprintf("INSERT IGNORE INTO %v %v VALUES", table, fields)
	iData := bytes.NewBufferString(valueSta(data[0]))
	size := len(q) + iData.Len()
	count := 1
	for x := 1; x < len(data); x++ {
		d := fmt.Sprintf(",%v", valueSta(data[x]))
		size += len(d)

		// When the insert query is bigger then 1Mbyte write it to db
		if size > 1000000 {
			e := fmt.Sprintf("%v %v", q, iData.String())
			log.Println(e)
			_, err := sqlDB.Exec(e)
			if err != nil {
				return err
			}

			iData = bytes.NewBufferString(valueSta(data[x]))
			size = len(q)
			count++
			continue
		}

		iData.WriteString(d)
		count++

	}

	if iData.Len() > 0 {
		e := fmt.Sprintf("%v %v", q, iData.String())
		log.Println(e)
		_, err := sqlDB.Exec(e)
		if err != nil {
			return err
		}
	}

	if count != len(data) {
		msg := fmt.Sprintf("Expect %v data was %v - %v", len(data), count, data)
		return errors.New(msg)
	}

	return nil
}
