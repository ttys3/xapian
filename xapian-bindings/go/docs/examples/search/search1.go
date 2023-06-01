package main

import (
	"fmt"
	"log"

	"xapian.org/xapian"
)

func main() {
	// ref https://github.com/xiaoyifang/goldendict-ng/blob/cd7e16de4b58105a3efe6985d283211b0890309c/src/ftshelpers.cc#L371

	qs := "description:结束生命"
	var offset uint = 0
	var pagesize uint = 5

	db, err := xapian.NewDatabase("../index/db")
	if err != nil {
		log.Fatal(err)
	}

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

	query := qp.Parse_query(qs, uint(xapian.QueryParserFLAG_DEFAULT|xapian.QueryParserFLAG_CJK_NGRAM))

	fmt.Println(query.Serialise(), query.Get_description())

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
	}

	db.Close()
}
