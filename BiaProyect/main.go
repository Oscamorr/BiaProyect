package main

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

const dsn = "postgres://postgres:<TU_PASSWORD>@localhost:5432/energy_db?sslmode=disable"

type MeterResponse struct {
	MeterID            int       `json:"meter_id"`
	Address            string    `json:"address"`
	Active             []float64 `json:"active"`
	ReactiveInductive  []float64 `json:"reactive_inductive"`
	ReactiveCapacitive []float64 `json:"reactive_capacitive"`
	Exported           []float64 `json:"exported"`
}

type ConsumptionResponse struct {
	Period    []string        `json:"period"`
	DataGraph []MeterResponse `json:"data_graph"`
}

func main() {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("no se pudo conectar a la BD:", err)
	}

	// AQUÍ creamos el servidor HTTP
	r := gin.Default()

	// ruta de prueba
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})

	// ruta que usa la BD
	r.GET("/consumption", func(c *gin.Context) {
		handleGetConsumption(c, db)
	})

	log.Println("Servidor escuchando en :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}

type dailyAgg struct {
	MeterID            int
	Day                time.Time
	Active             float64
	ReactiveInductive  float64
	ReactiveCapacitive float64
	Exported           float64
}

type Period struct {
	Start time.Time
	End   time.Time // exclusivo
	Label string
}

func buildPeriods(start, end time.Time, kind string) []Period {
	var periods []Period

	switch kind {
	case "daily":
		for d := start; d.Before(end); d = d.Add(24 * time.Hour) {
			label := strings.ToUpper(d.Format("Jan 2"))
			periods = append(periods, Period{
				Start: d,
				End:   d.Add(24 * time.Hour),
				Label: label,
			})
		}

	case "weekly":
		cur := start
		for cur.Before(end) {
			weekEnd := cur.AddDate(0, 0, 7)
			label := strings.ToUpper(cur.Format("Jan 2")) + " - " +
				strings.ToUpper(weekEnd.Add(-24*time.Hour).Format("Jan 2"))
			periods = append(periods, Period{
				Start: cur,
				End:   weekEnd,
				Label: label,
			})
			cur = weekEnd
		}

	case "monthly":
		cur := time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, start.Location())
		for cur.Before(end) {
			next := cur.AddDate(0, 1, 0)
			label := strings.ToUpper(cur.Format("Jan 2006"))
			periods = append(periods, Period{
				Start: cur,
				End:   next,
				Label: label,
			})
			cur = next
		}
	}

	return periods
}

func handleGetConsumption(c *gin.Context, db *sql.DB) {
	metersParam := c.Query("meters_ids")
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")
	kind := c.Query("kind_period")

	if metersParam == "" || startStr == "" || endStr == "" || kind == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing query params"})
		return
	}

	// 1) parsear meter_ids
	var meterIDs []int
	for _, s := range strings.Split(metersParam, ",") {
		id, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid meter_id"})
			return
		}
		meterIDs = append(meterIDs, id)
	}

	// 2) parsear fechas
	start, err := time.Parse("2006-01-02", startStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_date"})
		return
	}
	end, err := time.Parse("2006-01-02", endStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_date"})
		return
	}

	// normalizar a UTC, end exclusivo
	start = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.UTC)
	end = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.UTC).Add(24 * time.Hour)

	// 3) consultar agregados diarios en BD
	// por simplicidad, usamos IN dinámico en lugar de ANY + pq.Array
	// construiremos: ... WHERE meter_id IN ($1, $2, ...) ...
	placeholders := []string{}
	args := []interface{}{start, end}
	for i, id := range meterIDs {
		placeholders = append(placeholders, "$"+strconv.Itoa(i+3)) // $3, $4, ...
		args = append(args, id)
	}

	query := `
        SELECT
            meter_id,
            date_trunc('day', measured_at)::date AS day,
            SUM(active) AS active,
            SUM(reactive_inductive) AS reactive_inductive,
            SUM(reactive_capacitive) AS reactive_capacitive,
            SUM(exported) AS exported
        FROM consumptions
        WHERE
            measured_at >= $1
            AND measured_at < $2
            AND meter_id IN (` + strings.Join(placeholders, ",") + `)
        GROUP BY meter_id, day
        ORDER BY meter_id, day;
    `

	rows, err := db.Query(query, args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer rows.Close()

	var agg []dailyAgg
	for rows.Next() {
		var d dailyAgg
		if err := rows.Scan(&d.MeterID, &d.Day, &d.Active, &d.ReactiveInductive, &d.ReactiveCapacitive, &d.Exported); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		agg = append(agg, d)
	}

	// 4) construir periodos según kind_period
	periods := buildPeriods(start, end, kind)
	if len(periods) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "kind_period debe ser daily, weekly o monthly"})
		return
	}

	// etiquetas de periodo
	periodLabels := make([]string, len(periods))
	for i, p := range periods {
		periodLabels[i] = p.Label
	}

	// 5) acumular por medidor y periodo
	type acc struct {
		active, ri, rc, exp float64
	}
	data := make(map[int][]acc)
	for _, id := range meterIDs {
		data[id] = make([]acc, len(periods))
	}

	// recorrer los agregados diarios y sumarlos en el periodo correspondiente
	for _, d := range agg {
		for i, p := range periods {
			if !d.Day.Before(p.Start) && d.Day.Before(p.End) {
				a := data[d.MeterID][i]
				a.active += d.Active
				a.ri += d.ReactiveInductive
				a.rc += d.ReactiveCapacitive
				a.exp += d.Exported
				data[d.MeterID][i] = a
				break
			}
		}
	}

	// 6) construir respuesta
	resp := ConsumptionResponse{
		Period:    periodLabels,
		DataGraph: []MeterResponse{},
	}

	for _, meterID := range meterIDs {
		accs := data[meterID]
		mr := MeterResponse{
			MeterID:            meterID,
			Address:            "Dirección mock", // mock de dirección
			Active:             make([]float64, len(periods)),
			ReactiveInductive:  make([]float64, len(periods)),
			ReactiveCapacitive: make([]float64, len(periods)),
			Exported:           make([]float64, len(periods)),
		}
		for i, a := range accs {
			mr.Active[i] = a.active
			mr.ReactiveInductive[i] = a.ri
			mr.ReactiveCapacitive[i] = a.rc
			mr.Exported[i] = a.exp
		}
		resp.DataGraph = append(resp.DataGraph, mr)
	}

	c.JSON(http.StatusOK, resp)
}
