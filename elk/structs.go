package elk

import (
	"github.com/byuoitav/event-forwarding-microservice/state/statedefinition"
	"github.com/byuoitav/event-forwarding-microservice/structs"
)

// UpdateHeader .
type UpdateHeader struct {
	ID    string `json:"_id,omitempty"`
	Index string `json:"_index,omitempty"`
}

// DeviceUpdateInfo .
type DeviceUpdateInfo struct {
	Info string `json:"Info"`
	Name string `json:"Name"`
}

// UpdateBody .
type UpdateBody struct {
	Doc    map[string]interface{} `json:"doc"`
	Upsert bool                   `json:"doc_as_upsert"`
}

// Alert .
type Alert struct {
	Message   string `json:"message,omitempty"`
	AlertSent string `json:"alert-sent,omitempty"`
	Alerting  bool   `json:"alerting,omitempty"`
	Suppress  bool   `json:"Suppress,omitempty"`
}

// StaticDeviceQueryResponse .
type StaticDeviceQueryResponse struct {
	Hits struct {
		Wrappers []struct {
			ID     string                       `json:"_id"`
			Device statedefinition.StaticDevice `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// StaticRoomQueryResponse .
type StaticRoomQueryResponse struct {
	Hits struct {
		Wrappers []struct {
			ID   string                     `json:"_id"`
			Room statedefinition.StaticRoom `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// RoomIssueQueryResponse .
type RoomIssueQueryResponse struct {
	Hits struct {
		Wrappers []struct {
			ID    string            `json:"_id"`
			Alert structs.RoomIssue `json:"_source"`
		} `json:"hits"`
	} `json:"hits"`
}

// GenericQuery .
type GenericQuery struct {
	Query map[string]interface{} `json:"query,omitempty"`
	From  int                    `json:"from,omitempty"`
	Size  int                    `json:"size,omitempty"`
}
