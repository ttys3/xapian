package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"

	"xapian.org/xapian"
)

type Movie struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Year        int    `json:"year"`
}

func main() {
	// ref https://github.com/xiaoyifang/goldendict-ng/blob/cd7e16de4b58105a3efe6985d283211b0890309c/src/ftshelpers.cc#L238

	// a csv reader to get the values from csv file
	// provide path for the data (csv file)
	csvfile, err := os.Open("./testdata.csv")
	if err != nil {
		os.Exit(1)
	}
	// Open or Create the database we are goint to write in
	db := xapian.NewWritableDatabase("db", xapian.DB_CREATE_OR_OPEN)
	reader := csv.NewReader(csvfile)
	reader.Comma = '|'
	fields, _ := reader.Read()
	fmt.Println("Fields are ")
	fmt.Println(fields)
	// set up the termgenerator (indexer)
	termgenerator := xapian.NewTermGenerator()

	// https://xapian.org/docs/apidoc/html/classXapian_1_1TermGenerator.html#af958165211e730a6910d6986ac55a825
	termgenerator.Set_flags(xapian.TermGeneratorFLAG_CJK_NGRAM)
	// set up the stem object..with language
	termgenerator.Set_stemmer(xapian.NewStem("en"))
	for {
		fields, err := reader.Read()
		if err == io.EOF {
			break
		}
		// considering id ,tilte and description only
		year, _ := strconv.Atoi(fields[2])
		movie := &Movie{
			ID:          fields[0],
			Title:       fields[1],
			Year:        year,
			Description: fields[3],
		}
		fmt.Println("movie=", movie)

		// when we use := go compiler identifies the type , if we use = we need to specfy the type as below
		var x uint = 1

		doc := xapian.NewDocument()

		docJSON, _ := json.Marshal(movie)
		// save the source document
		doc.Set_data(string(docJSON))

		termgenerator.Set_document(doc)
		// text, wdf_inc, prefix
		termgenerator.Index_text(movie.Title, x, "T")
		// Increase the term position used by index_text.
		// This can be used between indexing text from different fields or other places
		// to prevent phrase searches from spanning between them (e.g. between the title and body text, or between two chapters in a book).
		termgenerator.Increase_termpos()
		termgenerator.Index_text(movie.Description, x, "D")

		fmt.Println("doc data:", doc.Get_data())

		idterm := "Q" + movie.ID

		// It is exactly the same as add_term(term, 0)
		doc.Add_boolean_term(idterm)

		doc.Add_value(1, xapian.Sortable_serialise(float64(movie.Year)))
		doc.Add_value(2, strconv.Itoa(movie.Year))

		/** Replace any documents matching a term.
		 *
		 *  This method replaces any documents indexed by the specified term
		 *  with the specified document.  If any documents are indexed by the
		 *  term, the lowest document ID will be used for the document,
		 *  otherwise a new document ID will be generated as for add_document.
		 *
		 *  One common use is to allow UIDs from another system to easily be
		 *  mapped to terms in Xapian.  Note that this method doesn't
		 *  automatically add unique_term as a term, so you'll need to call
		 *  document.add_term(unique_term) first when using replace_document()
		 *  in this way.
		 *
		 *  Note that changes to the database won't be immediately committed to
		 *  disk; see commit() for more details.
		 *
		 *  As with all database modification operations, the effect is
		 *  atomic: the document(s) will either be fully replaced, or the
		 *  document(s) fail to be replaced and an exception is thrown
		 *  (possibly at a
		 *  later time when commit() is called or the database is closed).
		 *
		 *  @param unique_term    The "unique" term.
		 *  @param document The new document.
		 *
		 *  @return         The document ID that document was given.
		 *
		 *  @exception Xapian::DatabaseError will be thrown if a problem occurs
		 *             while writing to the database.
		 *
		 *  @exception Xapian::DatabaseCorruptError will be thrown if the
		 *             database is in a corrupt state.
		 */
		// Xapian::docid replace_document(const std::string & unique_term, const Xapian::Document & document);

		db.Replace_document(idterm, doc)

		// ser := doc.Serialise()
		// log.Println("ser=", ser)
		// xapian.DeleteDocument(doc)

		// db.Delete_document(idterm)
	}
	fmt.Println("doc count:", db.Get_doccount())

	db.Commit()
	// close the database in order the save the Documents
	db.Close()
}
