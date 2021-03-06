package dao

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/elastic/go-elasticsearch/esapi"
	elasticsearch "github.com/elastic/go-elasticsearch/v7"
	"github.com/yamad07/go-modular-monolith/domain/search/pkg/domain/model"
	"github.com/yamad07/go-modular-monolith/domain/search/pkg/domain/value"
	"github.com/yamad07/go-modular-monolith/pkg/apperror"
)

const indexName = "cafe"

type Search struct {
	es *elasticsearch.Client
}

func NewSearch(es *elasticsearch.Client) Search {
	return Search{es}
}

func (c Search) Create(i value.CreateCafeIndex) apperror.Error {

	lat := strconv.FormatFloat(i.Latitude, 'f', 2, 64)
	log := strconv.FormatFloat(i.Longitude, 'f', 2, 64)
	var b strings.Builder
	b.WriteString(`{"latitude" : `)
	b.WriteString(lat)
	b.WriteString(`, "longitude": `)
	b.WriteString(log)
	b.WriteString(`}`)

	req := esapi.IndexRequest{
		Index:      indexName,
		DocumentID: strconv.Itoa(int(i.ID)),
		Body:       strings.NewReader(b.String()),
		Refresh:    "true",
	}
	res, err := req.Do(context.Background(), c.es)
	if err != nil {
		return apperror.New(apperror.CodeError, err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return apperror.New(apperror.CodeError, err)
	}

	return nil
}

func (c Search) Search(q value.RangeQuery) (
	candidates []model.CafeCandidate,
	aerr apperror.Error,
) {
	var buf bytes.Buffer
	query := map[string]interface{}{
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"must": []map[string]interface{}{{
					"range": map[string]interface{}{
						"latitude": map[string]interface{}{
							"gte": q.MinLatitude, "lt": q.MaxLatitude,
						},
					}}, {
					"range": map[string]interface{}{
						"longitude": map[string]interface{}{
							"gte": q.MinLongitude, "lt": q.MaxLongitude,
						},
					}},
				},
			},
		},
	}
	if err := json.NewEncoder(&buf).Encode(query); err != nil {
		return candidates, apperror.New(apperror.CodeError, err)
	}

	res, err := c.es.Search(
		c.es.Search.WithContext(context.Background()),
		c.es.Search.WithIndex(indexName),
		c.es.Search.WithBody(&buf),
		c.es.Search.WithTrackTotalHits(true),
		c.es.Search.WithPretty(),
	)
	if err != nil {
		return candidates, apperror.New(apperror.CodeError, err)
	}
	var r map[string]interface{}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			return candidates, apperror.New(apperror.CodeError, err)
		} else {
			rsn := e["error"].(map[string]interface{})["reason"]
			err := errors.New(fmt.Sprintf("%s", rsn))
			return candidates, apperror.New(apperror.CodeError, err)
		}
	}

	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return candidates, apperror.New(apperror.CodeError, err)
	}

	for _, hit := range r["hits"].(map[string]interface{})["hits"].([]interface{}) {
		idstr := fmt.Sprintf("%s", hit.(map[string]interface{})["_id"])
		id, err := strconv.Atoi(idstr)
		if err != nil {
			return candidates, apperror.New(apperror.CodeError, err)
		}
		cc := model.CafeCandidate{
			ID: int64(id),
		}
		candidates = append(candidates, cc)
	}

	return candidates, nil
}
