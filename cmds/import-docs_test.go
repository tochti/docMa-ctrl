package cmds

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
)

func Test_ImportDocs_Default(t *testing.T) {
	db := initMySQL(t)
	err := db.Insert(&labels.Label{
		ID:   1,
		Name: "Neu",
	})
	if err != nil {
		t.Fatal(err)
	}

	td, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)

	filename := "20140101_0000001.pdf"
	_, err = os.Create(path.Join(td, filename))
	if err != nil {
		t.Fatal(err)
	}

	err = ImportDocs(td)
	if err != nil {
		t.Fatal(err)
	}

	doc := docs.Doc{}
	err = db.SelectOne(&doc, "SELECT * FROM docs WHERE name=?", filename)
	if err != nil {
		t.Fatal(err)
	}

	d := time.Date(2014, 1, 1, 0, 0, 0, 0, time.Local)
	expect := docs.Doc{
		ID:            1,
		Name:          filename,
		Barcode:       "0000001",
		DateOfScan:    d,
		DateOfReceipt: d,
	}
	if doc.Name != filename ||
		doc.Barcode != expect.Barcode ||
		!doc.DateOfScan.Equal(d) ||
		!doc.DateOfReceipt.Equal(d) {
		t.Fatalf("Expect %v was %v", expect, doc)
	}

	labels := []labels.Label{}
	q := `
	SELECT labels.id, labels.name
	FROM labels, docs_labels
	WHERE labels.id=docs_labels.label_id
	AND docs_labels.doc_id=?
	`
	_, err = db.Select(&labels, q, doc.ID)
	if err != nil {
		t.Fatal(err)
	}

	if len(labels) != 1 {
		t.Fatalf("Expect %v was %v", 1, len(labels))
	}

	if labels[0].Name != "Neu" {
		t.Fatalf("Expect %v was %v", "Neu", labels[0].Name)
	}

}

func Test_ImportDocs_DuplicateEntry(t *testing.T) {
	db := initMySQL(t)
	err := db.Insert(&labels.Label{
		ID:   1,
		Name: "Neu",
	})
	if err != nil {
		t.Fatal(err)
	}

	td, err := ioutil.TempDir(".", "")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(td)

	filename := "20140101_0000001.pdf"
	_, err = os.Create(path.Join(td, filename))
	if err != nil {
		t.Fatal(err)
	}

	err = db.Insert(&docs.Doc{
		Name: filename,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ImportDocs(td)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_ParseFilename(t *testing.T) {
	_, _, err := ParseFilename("")
	if err == nil {
		t.Fatalf("Expect error was nil")
	}

	_, _, err = ParseFilename("1_1.pdf")
	if err == nil {
		t.Fatalf("Expect error was nil")
	}

	_, _, err = ParseFilename("20140101_1.pdf")
	if err == nil {
		t.Fatalf("Expect error was nil")
	}

	date, id, err := ParseFilename("20140101_0000001.pdf")
	if err != nil {
		t.Fatal(err)
	}

	d := time.Date(2014, 1, 1, 0, 0, 0, 0, time.Local)
	if date != d {
		t.Fatalf("Expect %v was %v", d, date)
	}

	if id != "0000001" {
		t.Fatalf("Expect %v was %v", "0000001", id)
	}

}
