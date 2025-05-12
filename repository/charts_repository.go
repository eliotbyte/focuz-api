package repository

import (
	"database/sql"
	"focuz-api/models"
	"focuz-api/types"
	"strconv"
	"time"
)

type ChartsRepository struct {
	db *sql.DB
}

func NewChartsRepository(db *sql.DB) *ChartsRepository {
	return &ChartsRepository{db: db}
}

func (r *ChartsRepository) CreateChart(userID, topicID, kindID, activityTypeID, periodID int) (*models.Chart, error) {
	var chart models.Chart
	err := r.db.QueryRow(`
		INSERT INTO chart (user_id, topic_id, kind, activity_type_id, period, created_at, modified_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING id, user_id, topic_id, kind, activity_type_id, period, created_at, modified_at
	`, userID, topicID, kindID, activityTypeID, periodID).Scan(
		&chart.ID,
		&chart.UserID,
		&chart.TopicID,
		&chart.KindID,
		&chart.ActivityTypeID,
		&chart.PeriodID,
		&chart.CreatedAt,
		&chart.ModifiedAt,
	)
	if err != nil {
		return nil, err
	}
	return &chart, nil
}

func (r *ChartsRepository) GetChartByID(id int) (*models.Chart, error) {
	var chart models.Chart
	err := r.db.QueryRow(`
		SELECT id, user_id, topic_id, kind, activity_type_id, period, is_deleted, created_at, modified_at
		FROM chart
		WHERE id = $1
	`, id).Scan(
		&chart.ID,
		&chart.UserID,
		&chart.TopicID,
		&chart.KindID,
		&chart.ActivityTypeID,
		&chart.PeriodID,
		&chart.IsDeleted,
		&chart.CreatedAt,
		&chart.ModifiedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &chart, nil
}

func (r *ChartsRepository) UpdateChartDeleted(id int, isDeleted bool) error {
	_, err := r.db.Exec(`
		UPDATE chart
		SET is_deleted = $1, modified_at = NOW()
		WHERE id = $2
	`, isDeleted, id)
	return err
}

func (r *ChartsRepository) UpdateChart(id int, kindID, activityTypeID, periodID int) error {
	_, err := r.db.Exec(`
		UPDATE chart
		SET kind = $1, activity_type_id = $2, period = $3, modified_at = NOW()
		WHERE id = $4
	`, kindID, activityTypeID, periodID, id)
	return err
}

func (r *ChartsRepository) GetCharts(spaceID int, topicID *int, filters models.ChartFilters) ([]*models.Chart, int, error) {
	offset := (filters.Page - 1) * filters.PageSize
	var conditions []string
	var params []interface{}
	idx := 1

	conditions = append(conditions, "c.is_deleted = FALSE")
	conditions = append(conditions, "t.space_id = $"+strconv.Itoa(idx))
	params = append(params, spaceID)
	idx++

	if topicID != nil {
		conditions = append(conditions, "c.topic_id = $"+strconv.Itoa(idx))
		params = append(params, *topicID)
		idx++
	}

	query := `
		SELECT c.id, c.user_id, c.topic_id, c.kind, c.activity_type_id, c.period, c.created_at, c.modified_at
		FROM chart c
		INNER JOIN topic t ON c.topic_id = t.id
	`

	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions, " AND ")
	}

	query += " ORDER BY c.created_at DESC"
	query += " LIMIT $" + strconv.Itoa(idx) + " OFFSET $" + strconv.Itoa(idx+1)
	params = append(params, filters.PageSize, offset)

	rows, err := r.db.Query(query, params...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var charts []*models.Chart
	for rows.Next() {
		var chart models.Chart
		err := rows.Scan(
			&chart.ID,
			&chart.UserID,
			&chart.TopicID,
			&chart.KindID,
			&chart.ActivityTypeID,
			&chart.PeriodID,
			&chart.CreatedAt,
			&chart.ModifiedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		charts = append(charts, &chart)
	}

	var total int
	countQuery := `
		SELECT COUNT(c.id)
		FROM chart c
		INNER JOIN topic t ON c.topic_id = t.id
	`
	if len(conditions) > 0 {
		countQuery += " WHERE " + joinConditions(conditions, " AND ")
	}
	err = r.db.QueryRow(countQuery, params[:len(params)-2]...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	return charts, total, nil
}

func (r *ChartsRepository) GetChartData(chart *models.Chart) ([]models.ChartDataPoint, error) {
	var startDate time.Time
	var endDate = time.Now()

	periodType := types.GetPeriodTypeByID(chart.PeriodID)
	if periodType == nil {
		return nil, nil
	}

	switch periodType.Name {
	case "day":
		startDate = endDate.AddDate(0, 0, -1)
	case "week":
		startDate = endDate.AddDate(0, 0, -7)
	case "month":
		startDate = endDate.AddDate(0, -1, 0)
	case "year":
		startDate = endDate.AddDate(-1, 0, 0)
	default:
		startDate = endDate.AddDate(0, 0, -7) // Default to week
	}

	// Сначала получаем тип агрегации
	var aggregation string
	err := r.db.QueryRow(`
		SELECT aggregation
		FROM activity_types
		WHERE id = $1
	`, chart.ActivityTypeID).Scan(&aggregation)
	if err != nil {
		return nil, err
	}

	// Формируем выражение агрегации
	var aggExpr string
	switch aggregation {
	case "sum":
		aggExpr = "SUM((a.value->>'data')::float)"
	case "avg":
		aggExpr = "AVG((a.value->>'data')::float)"
	case "count":
		aggExpr = "COUNT(*)"
	case "min":
		aggExpr = "MIN((a.value->>'data')::float)"
	case "max":
		aggExpr = "MAX((a.value->>'data')::float)"
	default:
		aggExpr = "0"
	}

	rows, err := r.db.Query(`
		SELECT date_trunc($1, a.created_at) as period,
		       `+aggExpr+` as value
		FROM activities a
		JOIN note n ON a.note_id = n.id
		JOIN topic t ON n.topic_id = t.id
		WHERE a.type_id = $2
		  AND a.is_deleted = FALSE
		  AND t.space_id = (SELECT space_id FROM topic WHERE id = $3)
		  AND a.created_at BETWEEN $4 AND $5
		GROUP BY period
		ORDER BY period
	`, periodType.Name, chart.ActivityTypeID, chart.TopicID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataPoints []models.ChartDataPoint
	for rows.Next() {
		var point models.ChartDataPoint
		err := rows.Scan(&point.Date, &point.Value)
		if err != nil {
			return nil, err
		}
		dataPoints = append(dataPoints, point)
	}

	return dataPoints, nil
}
