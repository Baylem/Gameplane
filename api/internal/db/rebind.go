package db

import "strconv"

// Rebind rewrites the `?` bind placeholders in query into the dialect of
// driver. It is the single placeholder helper behind every runtime query:
// the Postgres connector (db_postgres.go) runs each query string through it,
// so call sites keep writing portable `?` SQL.
//
//   - "postgres": each `?` becomes $1, $2, ... in order of appearance.
//   - anything else ("sqlite"): the query is returned unchanged.
//
// A `?` inside a single-quoted string literal, a double-quoted identifier,
// a `--` line comment or a `/* */` block comment is left alone, so literal
// text and comments never gain a placeholder.
func Rebind(driver, query string) string {
	if driver != "postgres" {
		return query
	}
	out := make([]byte, 0, len(query)+8)
	n := 0
	for i := 0; i < len(query); i++ {
		c := query[i]
		switch {
		case c == '\'' || c == '"':
			// Copy through the closing quote. A doubled quote ('' or "")
			// is an escaped quote and keeps the literal open.
			j := i + 1
			for j < len(query) {
				if query[j] == c {
					if j+1 < len(query) && query[j+1] == c {
						j += 2
						continue
					}
					break
				}
				j++
			}
			end := min(j+1, len(query))
			out = append(out, query[i:end]...)
			i = end - 1
		case c == '-' && i+1 < len(query) && query[i+1] == '-':
			j := i
			for j < len(query) && query[j] != '\n' {
				j++
			}
			out = append(out, query[i:j]...)
			i = j - 1
		case c == '/' && i+1 < len(query) && query[i+1] == '*':
			j := i + 2
			for j+1 < len(query) && (query[j] != '*' || query[j+1] != '/') {
				j++
			}
			end := min(j+2, len(query))
			out = append(out, query[i:end]...)
			i = end - 1
		case c == '?':
			n++
			out = append(out, '$')
			out = strconv.AppendInt(out, int64(n), 10)
		default:
			out = append(out, c)
		}
	}
	return string(out)
}
