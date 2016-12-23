package count

import (
	"github.com/dannyvankooten/ana/db"
)

// Pageviews returns the total number of pageviews between the given timestamps
func Pageviews(before int64, after int64) float64 {
	// get total
	stmt, err := db.Conn.Prepare(`
    SELECT
    SUM(a.count)
    FROM archive a
    WHERE a.metric = 'pageviews' AND UNIX_TIMESTAMP(a.date) <= ? AND UNIX_TIMESTAMP(a.date) >= ?`)
	checkError(err)
	defer stmt.Close()
	var total float64
	stmt.QueryRow(before, after).Scan(&total)
	return total
}

// PageviewsPerDay returns a slice of data points representing the number of pageviews per day
func PageviewsPerDay(before int64, after int64) []Point {
	stmt, err := db.Conn.Prepare(`SELECT
      SUM(a.count) AS count,
      DATE_FORMAT(a.date, '%Y-%m-%d') AS date_group
    FROM archive a
    WHERE a.metric = 'pageviews' AND UNIX_TIMESTAMP(a.date) <= ? AND UNIX_TIMESTAMP(a.date) >= ?
    GROUP BY date_group`)
	checkError(err)
	defer stmt.Close()

	rows, err := stmt.Query(before, after)
	checkError(err)
	defer rows.Close()

	var results []Point
	defer rows.Close()
	for rows.Next() {
		p := Point{}
		err = rows.Scan(&p.Value, &p.Label)
		checkError(err)
		results = append(results, p)
	}

	results = fill(after, before, results)
	return results
}

// CreatePageviewArchives aggregates pageview data into daily totals
func CreatePageviewArchives() {
	stmt, err := db.Conn.Prepare(`
    SELECT
      COUNT(*) AS count,
      DATE_FORMAT(pv.timestamp, "%Y-%m-%d") AS date_group
    FROM pageviews pv
    WHERE NOT EXISTS(
      SELECT a.id
      FROM archive a
      WHERE a.metric = 'pageviews' AND a.date = DATE_FORMAT(pv.timestamp, "%Y-%m-%d")
    )
    GROUP BY date_group`)
	checkError(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	checkError(err)
	defer rows.Close()

	db.Conn.Exec("START TRANSACTION")
	for rows.Next() {
		a := Archive{
			Metric: "pageviews",
			Value:  "",
		}
		err = rows.Scan(&a.Count, &a.Date)
		checkError(err)
		a.Save(db.Conn)
	}
	db.Conn.Exec("COMMIT")
}

// CreatePageviewArchivesPerPage aggregates pageview data for each page into daily totals
func CreatePageviewArchivesPerPage() {
	stmt, err := db.Conn.Prepare(`SELECT
      pv.page_id,
      COUNT(*) AS count,
			DATE_FORMAT(pv.timestamp, "%Y-%m-%d") AS date_group
    FROM pageviews pv
    WHERE NOT EXISTS (
			SELECT a.id
			FROM archive a
			WHERE a.metric = 'pageviews.page' AND a.date = DATE_FORMAT(pv.timestamp, "%Y-%m-%d") AND a.value = pv.page_id
		)
    GROUP BY pv.page_id, date_group`)
	checkError(err)
	defer stmt.Close()

	rows, err := stmt.Query()
	checkError(err)
	defer rows.Close()

	for rows.Next() {
		a := Archive{
			Metric: "pageviews.page",
		}
		err = rows.Scan(&a.Value, &a.Count, &a.Date)
		checkError(err)
		a.Save(db.Conn)
	}
}
