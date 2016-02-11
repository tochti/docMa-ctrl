package cmds

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"reflect"
	"time"

	"gopkg.in/gorp.v1"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/kelseyhightower/envconfig"
	"github.com/tochti/docMa-handler/accountingData"
	"github.com/tochti/docMa-handler/common"
	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
)

var (
	DocsColl           = "Docs"
	AccountingDataColl = "AccProcess"
)

type (
	MongoDBSpecs struct {
		Host   string `envconfig:"MONGODB_HOST"`
		DBName string `envconfig:"MONGODB_DB_NAME"`
	}

	DocInfos struct {
		DateOfScan    time.Time
		DateOfReceipt time.Time
	}

	DocNote string

	DocAccountData struct {
		DocNumbers []string
		DocPeriod  DocPeriod
		AccNumber  int
	}

	DocPeriod struct {
		From time.Time
		To   time.Time
	}

	Doc struct {
		ID          bson.ObjectId `bson:"_id,omitempty"`
		Name        string
		Barcode     string
		Infos       DocInfos
		Note        DocNote
		AccountData DocAccountData
		Labels      []string
	}

	AccProcess struct {
		ID               bson.ObjectId `bson:"_id,omitempty"`
		DocDate          time.Time
		DateOfEntry      time.Time
		DocNumberRange   string
		DocNumber        string
		PostingText      string
		AmountPosted     float64
		DebitAcc         int
		CreditAcc        int
		TaxCode          int
		CostUnit1        string
		CostUnit2        string
		AmountPostedEuro float64
		Currency         string
	}
)

// Migrate MongoDB to MySQL
// 1) Read from docs all labels
//	1.1) Make the label list unique
//	1.2) Create labels on MySQL
// 2) Copy docs to MySQL
//	5.1) Join labels with docs
//	5.2) Create all doc numbers
//	5.3) Create all account data
// 3) Create all accounting datas
func Migrate() error {
	sqlDB := common.InitMySQL()
	docs.AddTables(sqlDB)
	labels.AddTables(sqlDB)
	accountingData.AddTables(sqlDB)

	err := sqlDB.CreateTablesIfNotExists()
	if err != nil {
		return err
	}

	mgoSpecs := ReadMongoDBSpecs()
	mgoSession, err := mgo.Dial(mgoSpecs.String())
	if err != nil {
		return err
	}
	mgoDB := mgoSession.DB(MongoDBTestDB)

	err = MigrateLabels(sqlDB, mgoDB)
	if err != nil {
		return err
	}

	err = MigrateDocs(sqlDB, mgoDB)
	if err != nil {
		return err
	}

	err = MigrateAccountingData(sqlDB, mgoDB)
	if err != nil {
		return err
	}

	return nil
}

func MigrateDocs(sqlDB *gorp.DbMap, mgoDB *mgo.Database) error {
	ll := []labels.Label{}
	q := fmt.Sprintf("SELECT * FROM %v", labels.LabelsTable)
	_, err := sqlDB.Select(&ll, q)
	if err != nil {
		return err
	}
	lMap := NewLabelMap(&ll)

	docsColl := mgoDB.C(DocsColl)
	docsIter := docsColl.Find(bson.M{}).Iter()
	mDoc := &Doc{}
	for docsIter.Next(mDoc) {
		err := migrateDoc(sqlDB, mDoc, lMap)
		if err != nil {
			return err
		}
	}
	if err := docsIter.Close(); err != nil {
		return err
	}

	return nil
}

func migrateDoc(sqlDB *gorp.DbMap, mDoc *Doc, lMap map[string]int64) error {

	doc := docs.Doc{
		Name:          mDoc.Name,
		Barcode:       mDoc.Barcode,
		Note:          string(mDoc.Note),
		DateOfScan:    mDoc.Infos.DateOfScan,
		DateOfReceipt: mDoc.Infos.DateOfReceipt,
	}

	err := sqlDB.Insert(&doc)
	if err != nil {
		return err
	}

	// Create doc numbers in sql db
	docNumbers := []interface{}{}
	for _, dn := range mDoc.AccountData.DocNumbers {
		docNumbers = append(docNumbers, docs.DocNumber{DocID: doc.ID, Number: dn})
	}
	fields := "(doc_id,number)"
	fun := func(v interface{}) string {
		d, _ := v.(docs.DocNumber)
		return fmt.Sprintf("(%v, '%v')", d.DocID, d.Number)
	}
	err = BatchInsert(sqlDB, fields, docNumbers, docs.DocNumbersTable, fun)
	if err != nil {
		return err
	}

	// Create account data in sql db
	accountData := docs.DocAccountData{
		DocID:         doc.ID,
		AccountNumber: mDoc.AccountData.AccNumber,
		PeriodFrom:    mDoc.AccountData.DocPeriod.From,
		PeriodTo:      mDoc.AccountData.DocPeriod.To,
	}
	err = sqlDB.Insert(&accountData)
	if err != nil {
		return err
	}

	// Join labels with doc
	docsLabels := []interface{}{}
	for _, v := range mDoc.Labels {
		id, ok := lMap[v]
		if !ok {
			msg := fmt.Sprintf("Missing label %v", v)
			return errors.New(msg)
		}

		docsLabels = append(docsLabels, docs.DocsLabels{
			DocID:   doc.ID,
			LabelID: id,
		})
	}
	fields = "(doc_id,label_id)"
	fun = func(v interface{}) string {
		d, _ := v.(docs.DocsLabels)
		return fmt.Sprintf("(%v, %v)", d.DocID, d.LabelID)
	}
	err = BatchInsert(sqlDB, fields, docsLabels, docs.DocsLabelsTable, fun)
	if err != nil {
		return err
	}

	return nil
}

func BatchInsert(sqlDB *gorp.DbMap, fields string, data []interface{}, table string, valueSta func(interface{}) string) error {
	if len(data) == 0 {
		return nil
	}

	q := fmt.Sprintf("INSERT INTO %v %v VALUES", table, fields)
	iData := bytes.NewBufferString(valueSta(data[0]))
	size := len(q) + iData.Len()
	count := 1
	for x := 1; x < len(data); x++ {
		d := fmt.Sprintf(",%v", valueSta(data[x]))
		size += len(d)

		// When the insert query is bigger then 1Mbyte write it to db
		if size > 1000000 {
			_, err := sqlDB.Exec(fmt.Sprintf("%v %v", q, iData.String()))
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
		_, err := sqlDB.Exec(fmt.Sprintf("%v %v", q, iData.String()))
		if err != nil {
			return err
		}
	}

	q = fmt.Sprintf("SELECT COUNT(*) FROM %v", table)
	c, err := sqlDB.SelectInt(q)
	if err != nil {
		return err
	}

	if c != int64(count) || count != len(data) {
		msg := fmt.Sprintf("Expect %v data was %v", len(data), c)
		return errors.New(msg)
	}

	return nil
}

func MigrateLabels(sqlDB *gorp.DbMap, mgoDB *mgo.Database) error {
	l, err := ReadAllLabels(mgoDB)
	if err != nil {
		return err
	}
	RemoveDuplicates(&l)

	fields := "(name)"
	fun := func(v interface{}) string {
		label, _ := v.(string)
		return fmt.Sprintf("('%v')", label)
	}

	return BatchInsert(sqlDB, fields, IfaceSlice(l), labels.LabelsTable, fun)
}

func MigrateAccountingData(sqlDB *gorp.DbMap, mgoDB *mgo.Database) error {
	c := mgoDB.C(AccountingDataColl)
	d := []AccProcess{}
	err := c.Find(bson.M{}).All(&d)
	if err != nil {
		return err
	}

	fun := func(v interface{}) string {
		a, _ := v.(AccProcess)
		return fmt.Sprintf("('%v','%v','%v','%v','%v',%v,%v,%v,%v,'%v','%v',%v,'%v')",
			a.DocDate,
			a.DateOfEntry,
			a.DocNumberRange,
			a.DocNumber,
			a.PostingText,
			a.AmountPosted,
			a.DebitAcc,
			a.CreditAcc,
			a.TaxCode,
			a.CostUnit1,
			a.CostUnit2,
			a.AmountPostedEuro,
			a.Currency,
		)
	}
	fields := "(doc_date,date_of_entry,doc_number_range,doc_number,posting_text,amount_posted,debit_account,credit_account,tax_code,cost_unit1,cost_unit2,amount_posted_euro,currency)"

	return BatchInsert(sqlDB, fields, IfaceSlice(d), accountingData.AccountingDataTable, fun)
}

func ReadAllLabels(db *mgo.Database) ([]string, error) {
	c := db.C(DocsColl)

	docs := []Doc{}
	err := c.Find(bson.M{}).All(&docs)
	if err != nil {
		return []string{}, err
	}

	r := []string{}
	for _, d := range docs {
		r = append(r, d.Labels...)
	}

	return r, nil
}

func RemoveDuplicates(xs *[]string) {
	found := make(map[string]bool)
	j := 0
	for i, x := range *xs {
		if !found[x] {
			found[x] = true
			(*xs)[j] = (*xs)[i]
			j++
		}
	}
	*xs = (*xs)[:j]
}

func NewLabelMap(l *[]labels.Label) map[string]int64 {
	m := map[string]int64{}
	for _, label := range *l {
		m[label.Name] = label.ID
	}

	return m
}

func ReadMongoDBSpecs() MongoDBSpecs {
	specs := MongoDBSpecs{}
	err := envconfig.Process("", &specs)
	if err != nil {
		log.Fatal(err)
	}

	return specs
}

func (s MongoDBSpecs) String() string {
	return fmt.Sprintf("mongodb://%v/%v", s.Host, s.DBName)
}

// Make []"any type" to []interface{}
func IfaceSlice(slice interface{}) []interface{} {
	s := reflect.ValueOf(slice)
	if s.Kind() != reflect.Slice {
		log.Fatal("InterfaceSlice() given a non-slice type")
	}

	ret := make([]interface{}, s.Len())

	for i := 0; i < s.Len(); i++ {
		ret[i] = s.Index(i).Interface()
	}

	return ret
}
