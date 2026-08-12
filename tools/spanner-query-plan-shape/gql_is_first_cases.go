package main

const (
	gqlSurfaceISFirstReturnLabel            = "gql-surface/analytic/is-first"
	gqlSurfaceISFirstFilterLabel            = "gql-surface/analytic/is-first-filter"
	gqlSurfaceISFirstEdgeOneHopLabel        = "gql-surface/subquery/edge-in-is-first-one-hop"
	gqlSurfaceISFirstQuantifiedLabel        = "gql-surface/subquery/edge-in-is-first-quantified"
	gqlSurfaceISFirstBeforeNextLabel        = "gql-surface/analytic/is-first-before-next"
	gqlSurfaceISFirstBeforeNextOrderedLabel = "gql-surface/analytic/is-first-before-next-ordered"
)

// gqlISFirstSurfaceQueries separates the boolean-returning, predicate,
// edge-membership, and NEXT-boundary uses of IS_FIRST. The unordered NEXT
// case is retained for acceptance and plan coverage only: the documented
// function leaves results non-deterministic when ORDER BY is omitted.
var gqlISFirstSurfaceQueries = []queryCase{
	{
		Label: gqlSurfaceISFirstReturnLabel,
		SQL: `GRAPH MusicGraph
MATCH (s:Singers)-[e:CollabWith]->(f:Singers)
RETURN s.SingerId AS source_id, f.SingerId AS destination_id,
       IS_FIRST(1) OVER (PARTITION BY s.SingerId ORDER BY e.AlbumTitle DESC) AS selected`,
	},
	{
		Label: gqlSurfaceISFirstFilterLabel,
		SQL: `GRAPH MusicGraph
MATCH (s:Singers)-[e:CollabWith]->(f:Singers)
FILTER IS_FIRST(1) OVER (PARTITION BY s.SingerId ORDER BY e.AlbumTitle DESC)
RETURN s.SingerId AS source_id, f.SingerId AS destination_id`,
	},
	{
		Label: gqlSurfaceISFirstEdgeOneHopLabel,
		SQL: `GRAPH MusicGraph
MATCH (s:Singers)-[e:CollabWith WHERE e IN {
  MATCH -[selected_e:CollabWith]->
  FILTER IS_FIRST(1) OVER (
    PARTITION BY SOURCE_NODE_ID(selected_e)
    ORDER BY selected_e.AlbumTitle DESC)
  RETURN selected_e
}]->(f:Singers)
RETURN s.SingerId AS source_id, f.SingerId AS destination_id`,
	},
	{
		Label: gqlSurfaceISFirstQuantifiedLabel,
		SQL: `GRAPH MusicGraph
MATCH (a:Singers)-[e:CollabWith WHERE e IN {
  MATCH -[selected_e:CollabWith]->
  FILTER IS_FIRST(1) OVER (
    PARTITION BY SOURCE_NODE_ID(selected_e)
    ORDER BY selected_e.AlbumTitle DESC)
  RETURN selected_e
}]->{1,3}(b:Singers)
RETURN a.SingerId AS src_id, b.SingerId AS dst_id`,
	},
	{
		Label: gqlSurfaceISFirstBeforeNextLabel,
		SQL: `GRAPH MusicGraph
MATCH (a1:Singers)-[e1:CollabWith]->(a2:Singers)
FILTER IS_FIRST(1) OVER (PARTITION BY a2)
RETURN a1, a2
NEXT
MATCH (a2)-[e2:CollabWith]->(a3:Singers)
RETURN a1.SingerId AS src_id, a2.SingerId AS mid_id, a3.SingerId AS dst_id`,
	},
	{
		Label: gqlSurfaceISFirstBeforeNextOrderedLabel,
		SQL: `GRAPH MusicGraph
MATCH (a1:Singers)-[e1:CollabWith]->(a2:Singers)
FILTER IS_FIRST(1) OVER (PARTITION BY a2 ORDER BY e1.AlbumTitle DESC)
RETURN a1, a2
NEXT
MATCH (a2)-[e2:CollabWith]->(a3:Singers)
RETURN a1.SingerId AS src_id, a2.SingerId AS mid_id, a3.SingerId AS dst_id`,
	},
}
