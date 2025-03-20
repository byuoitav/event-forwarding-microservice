package elk

import (
	"encoding/json"
	"fmt"

	"github.com/byuoitav/common/state/statedefinition"
)

type roomQueryResponse struct {
	Took     int  `json:"took"`
	TimedOut bool `json:"timed_out"`
	Shards   struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Skipped    int `json:"skipped"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
	Hits struct {
		Total    int     `json:"total"`
		MaxScore float64 `json:"max_score"`
		Hits     []struct {
			Index  string                     `json:"_index"`
			ID     string                     `json:"_id"`
			Score  float64                    `json:"_score"`
			Source statedefinition.StaticRoom `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// GetRoomsBulk .
func GetRoomsBulk(rooms []string) ([]statedefinition.StaticRoom, error) {
	// assume that the rooms is the array of ID's
	query := IDQuery{}
	query.Query.IDS.Type = "room"
	query.Query.IDS.Values = rooms

	endpoint := fmt.Sprintf("/%s/_search", "oit-av-static-rooms")
	body, err := MakeGenericELKRequest("POST", endpoint, query, "", "")
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms bulk: %w", err)
	}

	// we have the body, unmarshal it
	rresp := roomQueryResponse{}
	gerr := json.Unmarshal(body, &rresp)
	if gerr != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", gerr)
	}

	toReturn := []statedefinition.StaticRoom{}
	for i := range rresp.Hits.Hits {
		toReturn = append(toReturn, rresp.Hits.Hits[i].Source)
	}

	return toReturn, nil
}
