package cmds

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/tochti/docMa-handler/docs"
	"github.com/tochti/docMa-handler/labels"
)

func Test_ImportDocs(t *testing.T) {
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

	tf, err := ioutil.TempFile(td, "")
	if err != nil {
		t.Fatal(err)
	}

	err = ImportDocs(td)
	if err != nil {
		t.Fatal(err)
	}

	doc := docs.Doc{}
	err = db.SelectOne(&doc, "SELECT * FROM docs WHERE name=?", path.Base(tf.Name()))
	if err != nil {
		t.Fatal(err)
	}

	if doc.Name != path.Base(tf.Name()) ||
		doc.DateOfScan.IsZero() ||
		doc.DateOfReceipt.IsZero() {
		t.Fatalf("Expect %v was %v", tf.Name(), doc.Name)
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

	tf, err := ioutil.TempFile(td, "")
	if err != nil {
		t.Fatal(err)
	}

	err = db.Insert(&docs.Doc{
		Name: path.Base(tf.Name()),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = ImportDocs(td)
	if err != nil {
		t.Fatal(err)
	}
}
