package cmds

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"testing"

	"gopkg.in/gorp.v1"
	"gopkg.in/mgo.v2"

	"github.com/tochti/docMa-handler/accountingData"
	"github.com/tochti/docMa-handler/common"
	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
	"github.com/tochti/gin-gum/gumtest"
)

var (
	MySQLTestDB   = "testing"
	MongoDBTestDB = "testing"
)

func Test_ReadMonogDBSpecs(t *testing.T) {
	setenv()
	s := ReadMongoDBSpecs()

	expect := "mongodb://127.0.0.1/testing"
	if s.String() != expect {
		t.Fatalf("Expect %v was %v", expect, s.String())
	}
}

func Test_ReadAllLabels(t *testing.T) {
	_, mgoDB := initDB(t)
	defer mgoDB.Session.Close()

	docs := mgoDB.C(DocsColl)

	doc := Doc{
		Name: "karl.pdf",
		Labels: []string{
			"l1", "l1", "l2",
		},
	}
	err := docs.Insert(doc)
	if err != nil {
		t.Fatal(err.Error())
	}

	l, err := ReadAllLabels(mgoDB)
	if err != nil {
		t.Fatal(err.Error())
	}

	expect := []string{"l1", "l1", "l2"}
	if !reflect.DeepEqual(expect, l) {
		t.Fatalf("Expect %v was %v", expect, l)
	}

}

func Test_RemoveDeuplicates(t *testing.T) {
	r := []string{"1", "1", "2"}
	RemoveDuplicates(&r)

	expect := []string{"1", "2"}
	if !reflect.DeepEqual(expect, r) {
		t.Fatalf("Expect %v was %v", expect, r)
	}
}

func Test_NewLabelMap(t *testing.T) {
	r := NewLabelMap(&[]labels.Label{{1, "l1"}, {2, "l2"}})

	if i, ok := r["l1"]; !ok || i != 1 {
		t.Fatalf("Expect (%v, %v) was (%v, %v)", true, 1, i, ok)
	}

	if i, ok := r["l2"]; !ok || i != 2 {
		t.Fatalf("Expect (%v, %v) was (%v, %v)", true, 2, i, ok)
	}
}

func Test_MigrateDocs(t *testing.T) {
	sqlDB, mgoDB := initDB(t)
	defer sqlDB.Db.Close()
	defer mgoDB.Session.Close()

	d := gumtest.SimpleNow()
	d1 := Doc{
		Name:    "Name-1",
		Barcode: "Barcode-1",
		Note:    DocNote("Note-1"),
		Labels:  []string{},
		Infos: DocInfos{
			DateOfScan:    d,
			DateOfReceipt: d,
		},
		AccountData: DocAccountData{
			DocNumbers: []string{"DN1-1", "DN2-1"},
			AccNumber:  1,
			DocPeriod: DocPeriod{
				From: d,
				To:   d,
			},
		},
	}

	d2 := Doc{
		Name:    "Name-2",
		Barcode: "Barcode-2",
		Note:    DocNote("Note-2"),
		Labels:  []string{"l1-2", "l2-2"},
		Infos: DocInfos{
			DateOfScan:    d,
			DateOfReceipt: d,
		},
		AccountData: DocAccountData{
			DocNumbers: []string{},
			AccNumber:  2,
			DocPeriod: DocPeriod{
				From: d,
				To:   d,
			},
		},
	}

	l := []*labels.Label{
		{3, "l1-2"},
		{4, "l2-2"},
	}
	err := sqlDB.Insert(gumtest.IfaceSlice(l)...)
	if err != nil {
		t.Fatal(err)
	}

	docsColl := mgoDB.C(DocsColl)
	err = docsColl.Insert(d1, d2)

	err = MigrateDocs(sqlDB, mgoDB)
	if err != nil {
		t.Fatal(err)
	}

	rDocs := []docs.Doc{}
	q := fmt.Sprintf("SELECT * FROM %v", docs.DocsTable)
	_, err = sqlDB.Select(&rDocs, q)

	rAccountData := []docs.DocAccountData{}
	q = fmt.Sprintf("SELECT * FROM %v", docs.DocAccountDataTable)
	_, err = sqlDB.Select(&rAccountData, q)

	rDocNumbers := []docs.DocNumber{}
	q = fmt.Sprintf("SELECT * FROM %v", docs.DocNumbersTable)
	_, err = sqlDB.Select(&rDocNumbers, q)

	eDocs := []docs.Doc{
		{1, "Name-1", "Barcode-1", d, d, "Note-1"},
		{2, "Name-2", "Barcode-2", d, d, "Note-2"},
	}
	if len(rDocs) != len(eDocs) {
		t.Fatalf("Expect %v was %v", len(eDocs), len(rDocs))
	}
	for i, d := range eDocs {
		if d.ID != rDocs[i].ID ||
			d.Name != rDocs[i].Name ||
			d.Barcode != rDocs[i].Barcode ||
			!d.DateOfScan.Equal(rDocs[i].DateOfScan) ||
			!d.DateOfReceipt.Equal(rDocs[i].DateOfReceipt) ||
			d.Note != rDocs[i].Note {
			t.Fatalf("Expect %v was %v", d, rDocs[i])
		}
	}

	eAccountData := []docs.DocAccountData{
		{1, d, d, 1},
		{2, d, d, 2},
	}
	if len(rAccountData) != len(eAccountData) {
		t.Fatalf("Expect %v was %v", len(eAccountData), len(rAccountData))
	}
	for i, a := range eAccountData {
		if a.DocID != rAccountData[i].DocID ||
			!a.PeriodFrom.Equal(rAccountData[i].PeriodFrom) ||
			!a.PeriodTo.Equal(rAccountData[i].PeriodTo) ||
			a.AccountNumber != rAccountData[i].AccountNumber {
			t.Fatalf("Expect %v was %v", a, rAccountData[i])
		}
	}

	eDocNumbers := []docs.DocNumber{
		{1, "DN1-1"}, {2, "DN1-2"},
	}
	if len(rDocNumbers) != len(eDocNumbers) {
		t.Fatalf("Expect %v was %v", len(eDocNumbers), len(rDocNumbers))
	}
	if reflect.DeepEqual(eDocNumbers, rDocNumbers) {
		t.Fatalf("Expect %v was %v", eDocNumbers, rDocNumbers)
	}

}

func Test_MigrateLabels(t *testing.T) {
	sqlDB, mgoDB := initDB(t)
	defer sqlDB.Db.Close()
	defer mgoDB.Session.Close()

	d := gumtest.SimpleNow()
	d1 := Doc{
		Name:    "Name-1",
		Barcode: "Barcode-1",
		Note:    DocNote("Note-1"),
		Labels:  []string{"l1-1", "l2-1", "l3"},
		Infos: DocInfos{
			DateOfScan:    d,
			DateOfReceipt: d,
		},
		AccountData: DocAccountData{
			DocNumbers: []string{"DN1-1", "DN2-1"},
			AccNumber:  1,
			DocPeriod: DocPeriod{
				From: d,
				To:   d,
			},
		},
	}

	d2 := Doc{
		Name:    "Name-2",
		Barcode: "Barcode-2",
		Note:    DocNote("Note-2"),
		Labels:  []string{"l1-2", "l2-2", "l3"},
		Infos: DocInfos{
			DateOfScan:    d,
			DateOfReceipt: d,
		},
		AccountData: DocAccountData{
			DocNumbers: []string{"DN1-2", "DN2-2"},
			AccNumber:  2,
			DocPeriod: DocPeriod{
				From: d,
				To:   d,
			},
		},
	}

	docsColl := mgoDB.C(DocsColl)
	err := docsColl.Insert(d1, d2)

	err = MigrateLabels(sqlDB, mgoDB)
	if err != nil {
		t.Fatal(err)
	}

	rLabels := []labels.Label{}
	q := fmt.Sprintf("SELECT * FROM %v", labels.LabelsTable)
	_, err = sqlDB.Select(&rLabels, q)

	eLabels := map[string]bool{
		"l1-1": false,
		"l2-2": false,
		"l2-1": false,
		"l3":   false,
		"l1-2": false,
	}
	if len(rLabels) != len(eLabels) {
		t.Fatalf("Expect %v was %v", len(eLabels), len(rLabels))
	}
	for _, v := range rLabels {
		if _, ok := eLabels[v.Name]; !ok {
			t.Fatalf("Found wrong label %v", v.Name)
		}
		eLabels[v.Name] = true
	}
	for _, v := range eLabels {
		if !v {
			t.Fatalf("Expect %v was %v", eLabels, rLabels)
		}
	}

}

func Test_MigrateLabels_ManyLabels(t *testing.T) {
	sqlDB, mgoDB := initDB(t)
	defer sqlDB.Db.Close()
	defer mgoDB.Session.Close()

	max := 150000
	tmp := []string{}
	for x := 0; x < max; x++ {
		tmp = append(tmp, "label-"+strconv.Itoa(x))
	}
	d := gumtest.SimpleNow()
	d1 := Doc{
		Name:    "Name-1",
		Barcode: "Barcode-1",
		Note:    DocNote("Note-1"),
		Labels:  tmp,
		Infos: DocInfos{
			DateOfScan:    d,
			DateOfReceipt: d,
		},
		AccountData: DocAccountData{
			DocNumbers: []string{"DN1-1", "DN2-1"},
			AccNumber:  1,
			DocPeriod: DocPeriod{
				From: d,
				To:   d,
			},
		},
	}

	docsColl := mgoDB.C(DocsColl)
	err := docsColl.Insert(d1)

	err = MigrateLabels(sqlDB, mgoDB)
	if err != nil {
		t.Fatal(err)
	}

	rLabels := []labels.Label{}
	q := fmt.Sprintf("SELECT * FROM %v", labels.LabelsTable)
	_, err = sqlDB.Select(&rLabels, q)

	if len(rLabels) != max {
		t.Fatalf("Expect %v was %v", max, len(rLabels))
	}

}

func Test_MigrateAccountingData(t *testing.T) {
	sqlDB, mgoDB := initDB(t)

	tmpD := gumtest.SimpleNow()
	accProcess := AccProcess{
		DocDate:          tmpD,
		DateOfEntry:      tmpD,
		DocNumberRange:   "DNR",
		DocNumber:        "123",
		PostingText:      "PT",
		AmountPosted:     1.1,
		DebitAcc:         1400,
		CreditAcc:        1500,
		TaxCode:          1,
		CostUnit1:        "CU1",
		CostUnit2:        "CU2",
		AmountPostedEuro: 1.2,
		Currency:         "EUR",
	}

	c := mgoDB.C(AccountingDataColl)
	err := c.Insert(accProcess)
	if err != nil {
		t.Fatal(err)
	}

	err = MigrateAccountingData(sqlDB, mgoDB)
	if err != nil {
		t.Fatal(err)
	}

	r := accountingData.AccountingData{}
	q := fmt.Sprintf("SELECT * FROM %v WHERE id=?",
		accountingData.AccountingDataTable)
	err = sqlDB.SelectOne(&r, q, 1)
	if err != nil {
		t.Fatal(err)
	}

	expect := accountingData.AccountingData{
		ID:               1,
		DocDate:          tmpD,
		DateOfEntry:      tmpD,
		DocNumberRange:   "DNR",
		DocNumber:        "123",
		PostingText:      "PT",
		AmountPosted:     1.1,
		DebitAccount:     1400,
		CreditAccount:    1500,
		TaxCode:          1,
		CostUnit1:        "CU1",
		CostUnit2:        "CU2",
		AmountPostedEuro: 1.2,
		Currency:         "EUR",
	}

	if r.ID != expect.ID ||
		!r.DocDate.Equal(expect.DocDate) ||
		!r.DateOfEntry.Equal(expect.DateOfEntry) ||
		r.DocNumberRange != expect.DocNumberRange ||
		r.DocNumber != expect.DocNumber ||
		r.PostingText != expect.PostingText ||
		r.AmountPosted != expect.AmountPosted ||
		r.DebitAccount != expect.DebitAccount ||
		r.CreditAccount != expect.CreditAccount ||
		r.TaxCode != expect.TaxCode ||
		r.CostUnit1 != expect.CostUnit1 ||
		r.CostUnit2 != expect.CostUnit2 ||
		r.AmountPostedEuro != expect.AmountPostedEuro ||
		r.Currency != expect.Currency {
		t.Fatalf("Expect %v was %v", expect, r)
	}

}

func Test_IfaceSlice(t *testing.T) {
	result := IfaceSlice([]string{"a", "b"})

	expect := []interface{}{
		"a",
		"b",
	}

	for i, s := range expect {
		if s != result[i] {
			t.Fatalf("Expect %v was %v", s, result[i])
		}
	}
}

func initDB(t *testing.T) (*gorp.DbMap, *mgo.Database) {
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

	mgoSpecs := ReadMongoDBSpecs()
	mgoSession, err := mgo.Dial(mgoSpecs.String())

	if err != nil {
		t.Fatal(err)
	}

	mgoDB := mgoSession.DB(MongoDBTestDB)
	err = mgoDB.DropDatabase()
	if err != nil {
		t.Fatal(err)
	}

	return dbMap, mgoDB
}

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
