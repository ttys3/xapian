package main

import (
	"fmt"
	"log"

	"xapian.org/xapian"
)

// https://www.swig.org/Doc4.1/Go.html#Go_adding_additional_code
type RangeProcessorWrap struct {
	xapian.NumberRangeProcessor
}

func (r *RangeProcessorWrap) DirectorInterface() interface{} {
	return nil
}

func main() {
	// ref https://github.com/xiaoyifang/goldendict-ng/blob/cd7e16de4b58105a3efe6985d283211b0890309c/src/ftshelpers.cc#L371
	// qs := "description:结束生命 year:1980..2000 id:78"
	qs := "description:机器人 year:1980..1985 id:78"
	var offset uint = 0
	var pagesize uint = 5

	db, err := xapian.NewDatabase("../index/db")
	if err != nil {
		log.Fatal(err)
	}

	// Set_database

	docNum := db.Get_doccount()
	log.Printf("docNum=%v", docNum)

	fmt.Println(db, offset, pagesize)

	qp := xapian.NewQueryParser()
	fmt.Println(qp)
	qp.Set_stemmer(xapian.NewStem("en"))

	var xp xapian.XapianQueryParserStem_strategy

	qp.Set_stemming_strategy(xp)
	qp.Add_prefix("title", "T")
	qp.Add_prefix("description", "D")
	qp.Add_boolean_prefix("id", "Q")

	// https://getting-started-with-xapian.readthedocs.io/en/latest/concepts/introduction.html
	// https://getting-started-with-xapian.readthedocs.io/en/latest/concepts/indexing/index.html
	// https://getting-started-with-xapian.readthedocs.io/en/latest/concepts/search/index.html

	// https://getting-started-with-xapian.readthedocs.io/en/latest/howtos/range_queries.html
	// https://getting-started-with-xapian.readthedocs.io/en/latest/howtos/boolean_filters.html
	// https://getting-started-with-xapian.readthedocs.io/en/latest/howtos/facets.html
	// https://getting-started-with-xapian.readthedocs.io/en/latest/howtos/facets.html
	// https://getting-started-with-xapian.readthedocs.io/en/latest/howtos/sorting.html

	// https://xapian.org/docs/overview.html
	// https://github.com/hightman/xunsearch/blob/8b9191daab6209b2d9281e4438b9d652bf53ba08/src/task.cc#L792
	// NewNumberRangeProcessor ?
	// https://xapian.org/docs/valueranges.html
	// https://xapian.org/docs/apidoc/html/classXapian_1_1NumberRangeProcessor.html

	// https://www.swig.org/Doc4.1/Go.html#Go_director_classes
	// panic: interface conversion: xapian.SwigcptrNumberRangeProcessor is not xapian.RangeProcessor: missing method DirectorInterface

	// rp := xapian.NewRangeProcessor(uint(2), "year:")
	nrp := xapian.NewNumberRangeProcessor(uint(1), "year:")
	qp.Add_rangeprocessor(&RangeProcessorWrap{nrp})

	query := qp.Parse_query(qs, uint(xapian.QueryParserFLAG_DEFAULT|xapian.QueryParserFLAG_CJK_NGRAM))

	log.Printf("query=%v query_desc=%v", query.Serialise(), query.Get_description())

	enquire := xapian.NewEnquire(db)
	enquire.Set_query(query)

	mset := enquire.Get_mset(offset, pagesize)

	log.Printf("results found matches_estimated=%v, size=%v", mset.Get_matches_estimated(), mset.Size())

	mset.Sort_by_relevance()

	fmt.Printf("%T %v\n", mset, mset)

	fmt.Println(mset.Get_description())
	fmt.Println(mset.Get_termfreq(qs))
	fmt.Println(db.Get_avlength())
	log.Printf("mset description=%v, termfreq=%v, avlength=%v", mset.Get_description(), mset.Get_termfreq(qs), db.Get_avlength())

	for m := mset.Begin(); !m.Equals(mset.End()); m.Next() {
		log.Printf("begin fetch")
		fmt.Println(m.Get_docid(), m.Get_percent(), m.Get_rank(), m.Get_weight())
		doc := m.Get_document()
		log.Printf("doc data=%+v", doc.Get_data())

		// pub fn snippet(&mut self, text: &str, length: i32, stem: &mut Stem, flags: i32, hi_start: &str, hi_end: &str, omit: &s
		log.Printf("snippets: %s", mset.Snippet(doc.Get_data(), int64(80), xapian.NewStem("en"), uint(1|2|2048), "<b>", "</b>", "..."))
	}

	db.Close()
}
