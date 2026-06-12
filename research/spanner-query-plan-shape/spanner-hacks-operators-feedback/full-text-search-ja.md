# Full Text Search operator evidence

Generated from `go run ./tools/spanner-query-plan-shape --case full_text_search --output reference --continue-on-error` on 2026-05-07 with Spanner Omni and `github.com/apstndb/spannerplan v0.1.9`.

=== full-text-search/search ===
SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "friday OR monday")
+-----+---------------------------------------------------------------------------------+
| ID  | Operator                                                                        |
+-----+---------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                               |
|   1 | +- [Input] VerifyDeterminism <Row>                                              |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                 |
|   3 | |     +- Unit Relation <Row>                                                    |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsTitleIndex <Row>     |
|   8 |    +- Local Distributed Union <Row>                                             |
|   9 |       +- Serialize Result <Row>                                                 |
| *10 |          +- SearchIndex Scan on SearchAlbumsTitleIndex <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleIndex predicate:$oo_tvf_0)


=== full-text-search/force-index ===
SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleIndex} WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony")
+-----+---------------------------------------------------------------------------------+
| ID  | Operator                                                                        |
+-----+---------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                               |
|   1 | +- [Input] VerifyDeterminism <Row>                                              |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                 |
|   3 | |     +- Unit Relation <Row>                                                    |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsTitleIndex <Row>     |
|   8 |    +- Local Distributed Union <Row>                                             |
|   9 |       +- Serialize Result <Row>                                                 |
| *10 |          +- SearchIndex Scan on SearchAlbumsTitleIndex <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleIndex predicate:$oo_tvf_0)


=== full-text-search/snippet ===
SELECT AlbumId, SNIPPET(AlbumTitle, "Fast Car") FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "Fast Car")
+-----+---------------------------------------------------------------------------------------+
| ID  | Operator                                                                              |
+-----+---------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                     |
|   1 | +- [Input] VerifyDeterminism <Row>                                                    |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                       |
|   3 | |     +- Unit Relation <Row>                                                          |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsTitleIndex <Row>           |
|  *8 |    +- Distributed Cross Apply <Row>                                                   |
|   9 |       +- [Input] Create Batch <Batch>                                                 |
|  10 |       |  +- RowToDataBlock                                                            |
|  11 |       |     +- Local Distributed Union <Row>                                          |
| *12 |       |        +- SearchIndex Scan on SearchAlbumsTitleIndex <Row> (scan_method: Row) |
|  20 |       +- [Map] Serialize Result <Row>                                                 |
|  21 |          +- Compute <Row>                                                             |
|  22 |             +- Cross Apply <Row>                                                      |
|  23 |                +- [Input] KeyRangeAccumulator <Row>                                   |
|  24 |                |  +- DataBlockToRow                                                   |
|  25 |                |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                |
|  30 |                +- [Map] Local Distributed Union <Row>                                 |
|  31 |                   +- Filter Scan <Row> (seekable_key_size: 0)                         |
| *32 |                      +- Table Scan on SearchAlbums <Row> (scan_method: Row)           |
+-----+---------------------------------------------------------------------------------------+
Predicates(identified by ID):
  8: Split Range: (($SearchAlbums_key_SingerId'4 = $SearchAlbums_key_SingerId'3) AND ($AlbumId' = $AlbumId))
 12: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleIndex predicate:$oo_tvf_0)
 32: Seek Condition: (($SearchAlbums_key_SingerId'4 = $batched_SearchAlbums_key_SingerId'4) AND ($AlbumId' = $batched_AlbumId'))


=== full-text-search/score-order ===
SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "fifth symphony") ORDER BY SCORE(AlbumTitle_Tokens, "fifth symphony") DESC
+-----+-------------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                                    |
+-----+-------------------------------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                                           |
|   1 | +- [Input] VerifyDeterminism <Row>                                                                          |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                                             |
|   3 | |     +- Unit Relation <Row>                                                                                |
|   7 | +- [Map] Serialize Result <Row>                                                                             |
|   8 |    +- VerifyDeterminism <Row>                                                                               |
|   9 |       +- Distributed Union on _Search2aryIndex_SearchAlbumsTitleIndex <Row> (preserve_subquery_order: true) |
|  10 |          +- VerifyDeterminism <Row>                                                                         |
|  11 |             +- Sort <Row>                                                                                   |
|  12 |                +- Local Distributed Union <Row>                                                             |
| *13 |                   +- SearchIndex Scan on SearchAlbumsTitleIndex <Row> (scan_method: Row)                    |
+-----+-------------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 13: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleIndex predicate:$oo_tvf_0)


=== full-text-search/substring ===
SELECT AlbumId FROM SearchAlbums WHERE SEARCH_SUBSTRING(AlbumTitle_SubstringTokens, "happ")
+----+---------------------------------------------------------------------------------------+
| ID | Operator                                                                              |
+----+---------------------------------------------------------------------------------------+
|  0 | Distributed Union on _Search2aryIndex_SearchAlbumsTitleSubstringIndex <Row>           |
|  1 | +- Local Distributed Union <Row>                                                      |
|  2 |    +- Serialize Result <Row>                                                          |
| *3 |       +- SearchIndex Scan on SearchAlbumsTitleSubstringIndex <Row> (scan_method: Row) |
+----+---------------------------------------------------------------------------------------+
Predicates(identified by ID):
 3: Search Predicate: SQUERY(index_name:AlbumTitle_SubstringTokens in SearchAlbumsTitleSubstringIndex predicate:SEARCH_SUBSTRING_QUERY('happ', false, <typed null>, <typed null>))


=== full-text-search/multi-column-conjunction ===
SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "car") AND SEARCH(AlbumStudio_Tokens, "sun")
+-----+---------------------------------------------------------------------------------------+
| ID  | Operator                                                                              |
+-----+---------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                     |
|   1 | +- [Input] VerifyDeterminism <Row>                                                    |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                       |
|   3 | |     +- Unit Relation <Row>                                                          |
|   9 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsTitleStudioIndex <Row>     |
|  10 |    +- Local Distributed Union <Row>                                                   |
|  11 |       +- Serialize Result <Row>                                                       |
| *12 |          +- SearchIndex Scan on SearchAlbumsTitleStudioIndex <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Search Predicate: (SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleStudioIndex predicate:$oo_tvf_0) AND SQUERY(index_name:AlbumStudio_Tokens in SearchAlbumsTitleStudioIndex predicate:$oo_tvf_1))


=== full-text-search/multi-column-disjunction ===
SELECT AlbumId FROM SearchAlbums WHERE SEARCH(AlbumTitle_Tokens, "car") OR SEARCH(AlbumStudio_Tokens, "sun")
+-----+---------------------------------------------------------------------------------------+
| ID  | Operator                                                                              |
+-----+---------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                     |
|   1 | +- [Input] VerifyDeterminism <Row>                                                    |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                       |
|   3 | |     +- Unit Relation <Row>                                                          |
|   9 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsTitleStudioIndex <Row>     |
|  10 |    +- Local Distributed Union <Row>                                                   |
|  11 |       +- Serialize Result <Row>                                                       |
| *12 |          +- SearchIndex Scan on SearchAlbumsTitleStudioIndex <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------------------+
Predicates(identified by ID):
 12: Search Predicate: (SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleStudioIndex predicate:$oo_tvf_0) OR SQUERY(index_name:AlbumStudio_Tokens in SearchAlbumsTitleStudioIndex predicate:$oo_tvf_1))


=== full-text-search/multi-column-negation ===
SELECT AlbumId FROM SearchAlbums WHERE NOT SEARCH(AlbumTitle_Tokens, "car")
+-----+---------------------------------------------------------------------------------+
| ID  | Operator                                                                        |
+-----+---------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                               |
|   1 | +- [Input] VerifyDeterminism <Row>                                              |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                 |
|   3 | |     +- Unit Relation <Row>                                                    |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsTitleIndex <Row>     |
|   8 |    +- Local Distributed Union <Row>                                             |
|   9 |       +- Serialize Result <Row>                                                 |
| *10 |          +- SearchIndex Scan on SearchAlbumsTitleIndex <Row> (scan_method: Row) |
+-----+---------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleIndex predicate:$oo_tvf_0)


=== full-text-search/tokenlist-concat ===
SELECT AlbumId FROM SearchAlbums WHERE SEARCH(TOKENLIST_CONCAT([AlbumTitle_Tokens, AlbumStudio_Tokens]), "blue note") ORDER BY SCORE(TOKENLIST_CONCAT([AlbumTitle_Tokens, AlbumStudio_Tokens]), "blue note") LIMIT 25
+-----+----------------------------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                                             |
+-----+----------------------------------------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                                                    |
|   1 | +- [Input] VerifyDeterminism <Row>                                                                                   |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                                                      |
|   3 | |     +- Unit Relation <Row>                                                                                         |
|   7 | +- [Map] Serialize Result <Row>                                                                                      |
|   8 |    +- Global Limit <Row>                                                                                             |
|   9 |       +- VerifyDeterminism <Row>                                                                                     |
|  10 |          +- Distributed Union on _Search2aryIndex_SearchAlbumsTitleStudioIndex <Row> (preserve_subquery_order: true) |
|  11 |             +- VerifyDeterminism <Row>                                                                               |
|  12 |                +- Local Sort Limit <Row>                                                                             |
|  13 |                   +- Local Distributed Union <Row>                                                                   |
| *14 |                      +- SearchIndex Scan on SearchAlbumsTitleStudioIndex <Row> (scan_method: Row)                    |
+-----+----------------------------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
 14: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens,AlbumStudio_Tokens in SearchAlbumsTitleStudioIndex predicate:SPAN_MAKE_FIELDED_SQUERY($oo_tvf_0, true, '_SearchDerivedTokenSet__SearchTokenIndex_AlbumTitle_Tokens_ON...(length 106)', '_SearchDerivedTokenSet__SearchTokenIndex_AlbumStudio_Tokens_O...(length 107)'))


=== full-text-search/partitioned-ordered-index ===
SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsTitleRatingIndex} WHERE SingerId = 1 AND SEARCH(AlbumTitle_Tokens, "fifth symphony") ORDER BY ReleaseTimestamp DESC LIMIT 10
+-----+------------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                       |
+-----+------------------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                              |
|   1 | +- [Input] VerifyDeterminism <Row>                                                             |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                                |
|   3 | |     +- Unit Relation <Row>                                                                   |
|   7 | +- [Map] Global Limit <Row>                                                                    |
|  *8 |    +- Distributed Union on _Search2aryIndex_SearchAlbumsTitleRatingIndex <Row>                 |
|   9 |       +- Serialize Result <Row>                                                                |
|  10 |          +- Local Limit <Row>                                                                  |
|  11 |             +- Local Distributed Union <Row>                                                   |
|  12 |                +- Filter Scan <Row> (seekable_key_size: 1)                                     |
| *13 |                   +- SearchIndex Scan on SearchAlbumsTitleRatingIndex <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  8: Split Range: ($SingerId = 1)
 13: Search Predicate: SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsTitleRatingIndex predicate:$oo_tvf_0)
     Seek Condition: ($SingerId = 1)


=== full-text-search/numeric-array-any ===
SELECT AlbumId FROM SearchAlbums WHERE ARRAY_INCLUDES_ANY(Ratings, [1, 2])
+----+--------------------------------------------------------------------------------+
| ID | Operator                                                                       |
+----+--------------------------------------------------------------------------------+
|  0 | Distributed Union on _Search2aryIndex_SearchAlbumsRatingsIndex <Row>           |
|  1 | +- Local Distributed Union <Row>                                               |
|  2 |    +- Serialize Result <Row>                                                   |
| *3 |       +- SearchIndex Scan on SearchAlbumsRatingsIndex <Row> (scan_method: Row) |
+----+--------------------------------------------------------------------------------+
Predicates(identified by ID):
 3: Search Predicate: SQUERY(index_name:Ratings_Tokens in SearchAlbumsRatingsIndex predicate:SPAN_NUMBER_QUERY_TO_SQUERY(9, false, false, SPAN_MAKE_NUMRANGE(9, [1, 2])))


=== full-text-search/numeric-array-all ===
SELECT AlbumId FROM SearchAlbums WHERE ARRAY_INCLUDES_ALL(Ratings, [1, 5])
+----+--------------------------------------------------------------------------------+
| ID | Operator                                                                       |
+----+--------------------------------------------------------------------------------+
|  0 | Distributed Union on _Search2aryIndex_SearchAlbumsRatingsIndex <Row>           |
|  1 | +- Local Distributed Union <Row>                                               |
|  2 |    +- Serialize Result <Row>                                                   |
| *3 |       +- SearchIndex Scan on SearchAlbumsRatingsIndex <Row> (scan_method: Row) |
+----+--------------------------------------------------------------------------------+
Predicates(identified by ID):
 3: Search Predicate: SQUERY(index_name:Ratings_Tokens in SearchAlbumsRatingsIndex predicate:SPAN_NUMBER_QUERY_TO_SQUERY(8, false, false, SPAN_MAKE_NUMRANGE(8, [1, 5])))


=== full-text-search/mixed-accelerated ===
SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsMixedIndex} WHERE (SEARCH(AlbumTitle_Tokens, "car") OR Rating > 4) AND NOT ARRAY_INCLUDES_ANY(Genres, ["jazz"])
+-----+------------------------------------------------------------------------------------+
| ID  | Operator                                                                           |
+-----+------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                  |
|   1 | +- [Input] VerifyDeterminism <Row>                                                 |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                    |
|   3 | |     +- Unit Relation <Row>                                                       |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsMixedIndex <Row>        |
|   8 |    +- Local Distributed Union <Row>                                                |
|   9 |       +- Serialize Result <Row>                                                    |
| *10 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *11 |             +- SearchIndex Scan on SearchAlbumsMixedIndex <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Residual Condition: ($sq OR (($Rating > 4) AND $sq'))
 11: Search Predicate: ((SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsMixedIndex predicate:$oo_tvf_0) OR SQUERY(index_name:Rating_Tokens in SearchAlbumsMixedIndex predicate:'(o r%01%0E5%FA%93%1A%00%00%01%04 r%01%1Ck%F5&4%00%00%01%02...(length 1371)')) AND SQUERY(index_name:Genres_Tokens in SearchAlbumsMixedIndex predicate:'(n jazz) '))


=== full-text-search/mixed-stored-filter ===
SELECT AlbumId FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsMixedIndex} WHERE SEARCH(AlbumTitle_Tokens, "car") AND Rating > 4 AND Likes >= 1000
+-----+------------------------------------------------------------------------------------+
| ID  | Operator                                                                           |
+-----+------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                  |
|   1 | +- [Input] VerifyDeterminism <Row>                                                 |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                    |
|   3 | |     +- Unit Relation <Row>                                                       |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsMixedIndex <Row>        |
|   8 |    +- Local Distributed Union <Row>                                                |
|   9 |       +- Serialize Result <Row>                                                    |
| *10 |          +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *11 |             +- SearchIndex Scan on SearchAlbumsMixedIndex <Row> (scan_method: Row) |
+-----+------------------------------------------------------------------------------------+
Predicates(identified by ID):
 10: Residual Condition: (($Rating > 4) AND ($Likes >= 1000))
 11: Search Predicate: (SQUERY(index_name:Rating_Tokens in SearchAlbumsMixedIndex predicate:'(o r%01%0E5%FA%93%1A%00%00%01%04 r%01%1Ck%F5&4%00%00%01%02...(length 1371)') AND SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsMixedIndex predicate:$oo_tvf_0))


=== full-text-search/mixed-back-join ===
SELECT AlbumId, Cover FROM SearchAlbums@{FORCE_INDEX=SearchAlbumsMixedIndex} WHERE SEARCH(AlbumTitle_Tokens, "car") AND Rating > 4
+-----+------------------------------------------------------------------------------------------+
| ID  | Operator                                                                                 |
+-----+------------------------------------------------------------------------------------------+
|   0 | Cross Apply <Row>                                                                        |
|   1 | +- [Input] VerifyDeterminism <Row>                                                       |
|   2 | |  +- TVF <Row> (Name: Search Query Conversion)                                          |
|   3 | |     +- Unit Relation <Row>                                                             |
|   7 | +- [Map] Distributed Union on _Search2aryIndex_SearchAlbumsMixedIndex <Row>              |
|  *8 |    +- Distributed Cross Apply <Row>                                                      |
|   9 |       +- [Input] Create Batch <Batch>                                                    |
|  10 |       |  +- RowToDataBlock                                                               |
|  11 |       |     +- Local Distributed Union <Row>                                             |
| *12 |       |        +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *13 |       |           +- SearchIndex Scan on SearchAlbumsMixedIndex <Row> (scan_method: Row) |
|  29 |       +- [Map] Serialize Result <Row>                                                    |
|  30 |          +- Cross Apply <Row>                                                            |
|  31 |             +- [Input] KeyRangeAccumulator <Row>                                         |
|  32 |             |  +- DataBlockToRow                                                         |
|  33 |             |     +- Batch Scan on $v2 <Batch> (scan_method: Batch)                      |
|  38 |             +- [Map] Local Distributed Union <Row>                                       |
|  39 |                +- Filter Scan <Row> (seekable_key_size: 0)                               |
| *40 |                   +- Table Scan on SearchAlbums <Row> (scan_method: Row)                 |
+-----+------------------------------------------------------------------------------------------+
Predicates(identified by ID):
  8: Split Range: (($SearchAlbums_key_SingerId' = $SearchAlbums_key_SingerId) AND ($AlbumId' = $AlbumId))
 12: Residual Condition: ($Rating > 4)
 13: Search Predicate: (SQUERY(index_name:Rating_Tokens in SearchAlbumsMixedIndex predicate:'(o r%01%0E5%FA%93%1A%00%00%01%04 r%01%1Ck%F5&4%00%00%01%02...(length 1371)') AND SQUERY(index_name:AlbumTitle_Tokens in SearchAlbumsMixedIndex predicate:$oo_tvf_0))
 40: Seek Condition: (($SearchAlbums_key_SingerId' = $batched_SearchAlbums_key_SingerId') AND ($AlbumId' = $batched_AlbumId'))

